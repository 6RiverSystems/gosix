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
	"cloud.google.com/go/pubsub"
)

type MessageCommon interface {
	Ack()
	Nack()
}
type Message interface {
	MessageCommon
	RealMessage() *RealMessage
	// TODO: actually access the message fields
}

// for imports to have access to the real one they need for sending
type RealMessage = pubsub.Message

var (
	_ MessageCommon = &pubsub.Message{}
	_ Message       = &monitoredMessage{}
)

type monitoredMessage struct {
	*pubsub.Message
	observeOutcome func(ack bool)
}

// TODO: these two won't cover that the underlying client only (n)acks a message
// once

func (m *monitoredMessage) Ack() {
	m.Message.Ack()
	m.observeOutcome(true)
}

func (m *monitoredMessage) Nack() {
	m.Message.Nack()
	m.observeOutcome(false)
}

func (m *monitoredMessage) RealMessage() *RealMessage {
	return m.Message
}
