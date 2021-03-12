package pubsub

import "cloud.google.com/go/pubsub"

type SubscriptionIterator interface {
	Next() (Subscription, error)
}

// var _ SubscriptionIterator = &pubsub.SubscriptionIterator{} // doesn't work because of interface return
var _ SubscriptionIterator = &monitoredSubscriptionIterator{}

type monitoredSubscriptionIterator struct {
	c *monitoredClient
	*pubsub.SubscriptionIterator
}

func (i *monitoredSubscriptionIterator) Next() (Subscription, error) {
	s, err := i.SubscriptionIterator.Next()
	if err != nil {
		return nil, err
	}
	return i.c.monitorSubscription(s), nil
}

type TopicIterator interface {
	Next() (Topic, error)
}

// var _ TopicIterator = &pubsub.TopicIterator{} // doesn't work because of interface return
var _ TopicIterator = &monitoredTopicIterator{}

type monitoredTopicIterator struct {
	c *monitoredClient
	*pubsub.TopicIterator
}

func (i *monitoredTopicIterator) Next() (Topic, error) {
	s, err := i.TopicIterator.Next()
	if err != nil {
		return nil, err
	}
	return i.c.monitorTopic(s), nil
}
