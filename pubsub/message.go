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

var _ MessageCommon = &pubsub.Message{}
var _ Message = &monitoredMessage{}

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
