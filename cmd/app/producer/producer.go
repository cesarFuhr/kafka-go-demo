package producer

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

const topic = "my-topic"

func SartProducers(ctx context.Context) error {
	dialer := kafka.Dialer{
		Timeout:   3 * time.Second,
		DualStack: true,
	}

	// Create the topic.
	conn, err := dialer.DialContext(ctx, "tcp", "kafka:9092")
	if err != nil {
		return fmt.Errorf("dialing: %w", err)
	}
	defer conn.Close()

	if err = conn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     10,
		ReplicationFactor: 1,
	}); err != nil {
		return fmt.Errorf("creating topic: %w", err)
	}

	// Build the writer.
	writer := kafka.NewWriter(kafka.WriterConfig{
		Topic:   topic,
		Brokers: []string{"kafka:9092"},
		Dialer:  &dialer,
		Async:   false,
	})

	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			err := writer.WriteMessages(ctx, kafka.Message{
				Value: []byte(time.Now().Format(time.RFC3339)),
			})
			if err != nil {
				return fmt.Errorf("writing messages: %w", err)
			}
			log.Println("ping: ", time.Now().Format(time.RFC3339))
		}
	}
}
