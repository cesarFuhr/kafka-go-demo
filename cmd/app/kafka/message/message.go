package message

type Message[V any] struct {
	Timestamp   int64
	Attempts    int
	MaxAttempts int
	// Audience of the message, this should be consumer groups that should care about the message.
	Audience []string
	Value    V
}

type Retry struct {
	DestinationTopic string
	BackoffDeadline  int64
	Message          any
}
