// Copyright (c) 2021 6 River Systems
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package pubsub

import (
	"context"
	"os"

	"cloud.google.com/go/pubsub"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/api/option"
)

type ClientCommon interface {
	Close() error
}
type Client interface {
	ClientCommon
	IsEmulator() bool
	CreateTopic(ctx context.Context, topicID string) (Topic, error)
	Subscription(id string) Subscription
	Subscriptions(context.Context) SubscriptionIterator
	Topic(id string) Topic
	Topics(context.Context) TopicIterator
	// TODO: CreateTopicWithConfig

	// CreateSubscription moved to Topic to avoid problems with the Topic member
	// of the SubscriptionConfig
	/* CreateSubscription(ctx context.Context, id string, cfg SubscriptionConfig) (Subscription, error) */
}

var (
	_ ClientCommon = &pubsub.Client{} // doesn't work because of interface return types
	_ Client       = &monitoredClient{}
)

type monitoredClient struct {
	*pubsub.Client
	isEmulator       bool
	publishStarted   *prometheus.CounterVec
	published        *prometheus.CounterVec
	publishFailed    *prometheus.CounterVec
	messagesReceived *prometheus.CounterVec
	messageDuration  *prometheus.HistogramVec
}

// TODO: we'll get a panic somewhere if NewClient is called twice with the same
// namespace and constLabels and touches the same topic/subscriptions due to
// trying to register duplicate prometheus metrics

func NewClient(
	ctx context.Context,
	projectID string,
	promReg prometheus.Registerer,
	promNamespace string,
	promLabels prometheus.Labels,
	opts ...option.ClientOption,
) (Client, error) {
	if projectID == "" {
		projectID = DefaultProjectId()
	}
	// Use LookupEnv here so you can force test/dev environments to use real
	// PubSub by setting the env var to the empty string
	emulatorVal, emulatorSet := os.LookupEnv(EmulatorHostEnvVar)
	isEmulator := emulatorVal != ""
	// use the emulator in dev/test unless explicitly requested otherwise
	switch os.Getenv("NODE_ENV") {
	case "test", "acceptance", "development":
		if !emulatorSet {
			os.Setenv(EmulatorHostEnvVar, DefaultEmulatorHost)
			isEmulator = true
		}
	}
	// TODO: recognize an alternate variable instead of PUBSUB_EMULATOR_HOST to
	// change the target endpoint for pointing at mmmbbb, but being able to use
	// insecure and a connection pool
	ps, err := pubsub.NewClient(ctx, projectID, opts...)
	if err != nil {
		return nil, err
	}
	po := func(name, help string) (prometheus.CounterOpts, []string) {
		return prometheus.CounterOpts{
			Namespace:   promNamespace,
			Subsystem:   "pubsub",
			Name:        name,
			Help:        help,
			ConstLabels: promLabels,
		}, []string{"topic"}
	}

	client := &monitoredClient{
		Client:     ps,
		isEmulator: isEmulator,

		publishStarted: prometheus.NewCounterVec(po("publish_started", "Number of pubsub messages attempted to send")),
		published:      prometheus.NewCounterVec(po("published", "Number of pubsub messages successfully sent")),
		publishFailed:  prometheus.NewCounterVec(po("publish_failed", "Number of pubsub messages failed to send")),

		messagesReceived: prometheus.NewCounterVec(po("messages_received", "Number of pubsub messages received")),

		messageDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace:   promNamespace,
				Subsystem:   "pubsub",
				Name:        "message_duration",
				Help:        "How long a message takes to process, by outcome",
				ConstLabels: promLabels,
				Buckets: []float64{
					// prometheus.DefBuckets:
					.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10,
					// expanded range
					15, 30, 60, 120, 300,
				},
			},
			[]string{"subscription", "outcome"},
		),
	}
	if promReg != nil {
		for _, c := range []prometheus.Collector{
			client.publishStarted, client.published, client.publishFailed,
			client.messagesReceived, client.messageDuration,
		} {
			if err := promReg.Register(c); err != nil {
				return client, err
			}
		}
	}
	return client, nil
}

func (c *monitoredClient) IsEmulator() bool {
	return c.isEmulator
}

func (c *monitoredClient) Topic(id string) Topic {
	return c.monitorTopic(c.Client.Topic(id))
}

func (c *monitoredClient) Topics(ctx context.Context) TopicIterator {
	return c.monitorTopicIterator(c.Client.Topics(ctx))
}

func (c *monitoredClient) CreateTopic(ctx context.Context, topicID string) (Topic, error) {
	t, err := c.Client.CreateTopic(ctx, topicID)
	return c.monitorTopic(t), err
}

// TODO: TopicsInProject, SubscriptionsInProject

func (c *monitoredClient) Subscription(id string) Subscription {
	return c.monitorSubscription(c.Client.Subscription(id))
}

func (c *monitoredClient) Subscriptions(ctx context.Context) SubscriptionIterator {
	return c.monitorSubscriptionIterator(c.Client.Subscriptions(ctx))
}

// non-exported because it uses the real SubscriptionConfig type and thus
// exposes the real Topic object
func (c *monitoredClient) createSubscription(ctx context.Context, id string, cfg pubsub.SubscriptionConfig) (Subscription, error) {
	s, err := c.Client.CreateSubscription(ctx, id, cfg)
	return c.monitorSubscription(s), err
}

func (c *monitoredClient) monitorTopic(t *pubsub.Topic) Topic {
	if t == nil {
		return nil
	}
	id := t.ID()
	return &monitoredTopic{
		c:              c,
		Topic:          t,
		publishStarted: c.publishStarted.WithLabelValues(id),
		published:      c.published.WithLabelValues(id),
		publishFailed:  c.publishFailed.WithLabelValues(id),
	}
}

func (c *monitoredClient) monitorTopicIterator(i *pubsub.TopicIterator) TopicIterator {
	if i == nil {
		return nil
	}
	return &monitoredTopicIterator{c, i}
}

func (c *monitoredClient) monitorSubscription(s *pubsub.Subscription) Subscription {
	if s == nil {
		return nil
	}
	id := s.ID()
	return &monitoredSubscription{
		c:                c,
		Subscription:     s,
		messagesReceived: c.messagesReceived.WithLabelValues(id),
		messageDuration:  c.messageDuration.MustCurryWith(prometheus.Labels{"subscription": id}).(*prometheus.HistogramVec),
	}
}

func (c *monitoredClient) monitorSubscriptionIterator(i *pubsub.SubscriptionIterator) SubscriptionIterator {
	if i == nil {
		return nil
	}
	return &monitoredSubscriptionIterator{c, i}
}
