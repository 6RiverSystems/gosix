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
}
type Subscription interface {
	SubscriptionCommon
	Receive(ctx context.Context, msg MessageHandler) error
	ReceiveSettings() *pubsub.ReceiveSettings
	EnsureDefaultConfig(context.Context, ...func(*SubscriptionConfigToUpdate)) (SubscriptionConfig, error)
}

var _ SubscriptionCommon = &pubsub.Subscription{} // doesn't work because Receive handler message type
var _ Subscription = &monitoredSubscription{}

/* // wrapper variant to deal with the Topic member variance
type SubscriptionConfig struct {
	pubsub.SubscriptionConfig
	Topic Topic
}
*/

type SubscriptionConfig = pubsub.SubscriptionConfig
type SubscriptionConfigToUpdate = pubsub.SubscriptionConfigToUpdate
type RetryPolicy = pubsub.RetryPolicy

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

var defaultRetryPolicy = pubsub.RetryPolicy{
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
	rp := defaultRetryPolicy
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
