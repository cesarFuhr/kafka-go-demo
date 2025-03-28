package message

import (
	"strings"

	"github.com/segmentio/kafka-go"
)

type Message[V any] struct {
	Timestamp int64
	Value     V
}

type Headers struct {
	Audience    []string
	Attempts    int
	MaxAttempts int
}

func NewHeaders(messageHeaders []kafka.Header) Headers {
	var h Headers
	for _, mh := range messageHeaders {
		switch mh.Key {
		case "audience":
			h.Audience = strings.Split(string(mh.Value), ",")
		case "attempts":
			if len(mh.Value) == 1 {
				h.Attempts = int(mh.Value[0])
			}
		}
	}
	return h
}

type Retry struct {
	DestinationTopic string
	// Audience of the message, this should be consumer groups that should care about the message.
	Audience        []string
	Attempts        byte
	MaxAttempts     byte
	BackoffDeadline int64
	Message         any
}
