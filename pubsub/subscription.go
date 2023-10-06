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
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MessageHandler func(context.Context, Message)

type SubscriptionCommon interface {
	fmt.Stringer
	ID() string
	Config(context.Context) (pubsub.SubscriptionConfig, error)
	Delete(context.Context) error
	Exists(context.Context) (bool, error)
	Update(context.Context, pubsub.SubscriptionConfigToUpdate) (pubsub.SubscriptionConfig, error)
	SeekToTime(ctx context.Context, t time.Time) error
}
type Subscription interface {
	SubscriptionCommon
	Receive(ctx context.Context, msg MessageHandler) error
	ReceiveSettings() *pubsub.ReceiveSettings
	EnsureDefaultConfig(context.Context, ...func(*SubscriptionConfigToUpdate)) (SubscriptionConfig, error)
}

var (
	_ SubscriptionCommon = &pubsub.Subscription{} // doesn't work because Receive handler message type
	_ Subscription       = &monitoredSubscription{}
)

/* // wrapper variant to deal with the Topic member variance
type SubscriptionConfig struct {
	pubsub.SubscriptionConfig
	Topic Topic
}
*/

type (
	SubscriptionConfig         = pubsub.SubscriptionConfig
	SubscriptionConfigToUpdate = pubsub.SubscriptionConfigToUpdate
	RetryPolicy                = pubsub.RetryPolicy
	PushConfig                 = pubsub.PushConfig
	AuthenticationMethod       = pubsub.AuthenticationMethod
	OIDCToken                  = pubsub.OIDCToken
)

type monitoredSubscription struct {
	c *monitoredClient
	*pubsub.Subscription
	messagesReceived prometheus.Counter
	messageDuration  *prometheus.HistogramVec
}

func (s *monitoredSubscription) Receive(ctx context.Context, f MessageHandler) error {
	return s.Subscription.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
		s.messagesReceived.Inc()
		acked := false
		observed := false
		timer := prometheus.NewTimer(prometheus.ObserverFunc(func(duration float64) {
			var o prometheus.Observer
			if !observed {
				o = s.messageDuration.WithLabelValues("drop")
			} else if acked {
				o = s.messageDuration.WithLabelValues("ack")
			} else {
				o = s.messageDuration.WithLabelValues("nack")
			}
			o.Observe(duration)
		}))
		observeOutcome := func(ack bool) {
			observed = true
			acked = ack
			timer.ObserveDuration()
		}
		f(ctx, &monitoredMessage{m, observeOutcome})
		if !observed {
			// record "drop" time/count
			timer.ObserveDuration()
		}
	})
}

func (s *monitoredSubscription) ReceiveSettings() *pubsub.ReceiveSettings {
	return &s.Subscription.ReceiveSettings
}

var DefaultRetryPolicy = pubsub.RetryPolicy{
	MinimumBackoff: time.Second,
	MaximumBackoff: 10 * time.Minute,
}

// EnsureDefaultConfig will update the subscription config with standard
// defaults. Option args may be passed to modify these before they are applied
// to the subscription.
func (s *monitoredSubscription) EnsureDefaultConfig(
	ctx context.Context,
	opts ...func(*SubscriptionConfigToUpdate),
) (SubscriptionConfig, error) {
	// make copies of stuff so opts don't affect defaults for future calls
	rp := DefaultRetryPolicy
	cfg := SubscriptionConfigToUpdate{
		RetryPolicy: &rp,
	}
	for _, o := range opts {
		o(&cfg)
	}
	outCfg, err := s.Subscription.Update(ctx, cfg)
	code := status.Code(err)
	if err != nil &&
		(code == codes.InvalidArgument || code == codes.Unimplemented) &&
		s.c.isEmulator {
		// emulator doesn't support lots of subscription update requests, ignore
		// errors it gives us and return the old config, but only on errors, since
		// mmmbbb works fine
		return s.Subscription.Config(ctx)
	}
	return outCfg, err
}
