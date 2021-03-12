package pubsub

import (
	pubsubv1 "google.golang.org/api/pubsub/v1"
)

type PushMessage = pubsubv1.PubsubMessage

// surprisingly, google doesn't export a type for this
type PushRequest struct {
	Message      PushMessage `json:"message"`
	Subscription string      `json:"subscription"`
}
