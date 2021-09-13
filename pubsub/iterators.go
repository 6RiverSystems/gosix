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

func (i *monitoredSubscriptionIterator) NextConfig() (*pubsub.SubscriptionConfig, error) {
	return i.SubscriptionIterator.NextConfig()
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

func (i *monitoredTopicIterator) NextConfig() (*pubsub.TopicConfig, error) {
	return i.TopicIterator.NextConfig()
}
