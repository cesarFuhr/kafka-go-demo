package producer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/cesarFuhr/kafka-go-demo/cmd/app/kafka/message"
	"github.com/segmentio/kafka-go"
)

const topic = "my-topic"

func SartProducers(ctx context.Context) error {
	dialer := kafka.Dialer{
		Timeout:   3 * time.Second,
		DualStack: true,
	}

	conn, err := dialer.DialContext(ctx, "tcp", "kafka:9092")
	if err != nil {
		return fmt.Errorf("dialing: %w", err)
	}
	defer conn.Close()

	// Create the topic.
	if err = conn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     10,
		ReplicationFactor: 1,
	}); err != nil {
		return fmt.Errorf("creating topic: %w", err)
	}

	// Build the writer.
	writer := kafka.NewWriter(kafka.WriterConfig{
		Topic:     topic,
		Brokers:   []string{"kafka:9092"},
		Dialer:    &dialer,
		Async:     false,
		BatchSize: 1,
	})

	ticker := time.NewTicker(time.Millisecond * 500)
	defer ticker.Stop()
	var counter uint = 0
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			now := time.Now()
			message := message.Message{
				Timestamp: now.Unix(),
				Attempts:  0,
				Value:     counter,
			}
			bts, err := json.Marshal(message)
			if err != nil {
				return fmt.Errorf("writing messages: %w", err)
			}

			err = writer.WriteMessages(ctx, kafka.Message{
				Value: bts,
			})
			if err != nil {
				return fmt.Errorf("writing messages: %w", err)
			}
			log.Println("sent: ", message.Value, time.Now().Format(time.RFC3339Nano))
			counter += 1
		}
	}
}
