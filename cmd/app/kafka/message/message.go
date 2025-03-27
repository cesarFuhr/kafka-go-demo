package message

import (
	"strings"

	"github.com/segmentio/kafka-go"
)

type Message[V any] struct {
	Timestamp   int64
	Attempts    int
	MaxAttempts int
	Value       V
}

type Headers struct {
	Audience []string
}

func NewHeaders(messageHeaders []kafka.Header) Headers {
	var h Headers
	for _, mh := range messageHeaders {
		switch mh.Key {
		case "audience":
			h.Audience = strings.Split(string(mh.Value), ",")
		}
	}
	return h
}

type Retry struct {
	DestinationTopic string
	// Audience of the message, this should be consumer groups that should care about the message.
	Audience        []string
	BackoffDeadline int64
	Message         any
}
