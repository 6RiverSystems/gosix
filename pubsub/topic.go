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
	"fmt"

	"cloud.google.com/go/pubsub"
	"github.com/prometheus/client_golang/prometheus"
)

// re-export for ease of use
type TopicConfig = pubsub.TopicConfig
type TopicConfigToUpdate = pubsub.TopicConfigToUpdate
type PublishResult = pubsub.PublishResult
type PublishSettings = pubsub.PublishSettings

type TopicCommon interface {
	fmt.Stringer
	Config(context.Context) (TopicConfig, error)
	Delete(context.Context) error
	Exists(context.Context) (bool, error)
	ID() string
	Publish(context.Context, *RealMessage) *PublishResult
	Stop()
	Update(context.Context, TopicConfigToUpdate) (pubsub.TopicConfig, error)
}

type Topic interface {
	TopicCommon
	PublishSettings() *PublishSettings
	Subscriptions(context.Context) SubscriptionIterator

	// CreateSubscription lives on Client in the real API, but we move it here to
	// avoid worrying about the Topic member on the SubscriptionConfig struct
	CreateSubscription(ctx context.Context, id string, cfg SubscriptionConfig) (Subscription, error)
}

var _ TopicCommon = &pubsub.Topic{} // Doesn't work because of interface return
var _ Topic = &monitoredTopic{}

type monitoredTopic struct {
	c *monitoredClient
	*pubsub.Topic
	publishStarted prometheus.Counter
	published      prometheus.Counter
	publishFailed  prometheus.Counter
}

func (t *monitoredTopic) Publish(ctx context.Context, msg *pubsub.Message) *pubsub.PublishResult {
	// Why does the google api require us to push this button? it is only used to
	// make publishing throw an error if we forgot to push it
	if msg.OrderingKey != "" {
		t.Topic.EnableMessageOrdering = true
	}

	r := t.Topic.Publish(ctx, msg)
	t.publishStarted.Inc()
	// collect the results in the background whether or not the caller cares about
	// the publish
	go func() {
		_, err := r.Get(ctx)
		if err != nil {
			t.publishFailed.Inc()
		} else {
			t.published.Inc()
		}
	}()
	return r
}

func (t *monitoredTopic) Subscriptions(ctx context.Context) SubscriptionIterator {
	return t.c.monitorSubscriptionIterator(t.Topic.Subscriptions(ctx))
}

func (t *monitoredTopic) PublishSettings() *pubsub.PublishSettings {
	return &t.Topic.PublishSettings
}

func (t *monitoredTopic) CreateSubscription(ctx context.Context, id string, cfg SubscriptionConfig) (Subscription, error) {
	// copy & modify config so we don't expose the real Topic object to the caller
	realCfg := cfg
	realCfg.Topic = t.Topic
	return t.c.createSubscription(ctx, id, realCfg)
	// we could call EnsureDefaultConfig here, but that doesn't save much
}
