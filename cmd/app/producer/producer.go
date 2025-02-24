package producer

import (
	"context"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

const topic = "my-topic"

func NewProducer(ctx context.Context) {
	conn, err := kafka.DialContext(ctx, "tcp", "kafka:9092")
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	if err = conn.CreateTopics(kafka.TopicConfig{
		Topic:         topic,
		NumPartitions: 1,
	}); err != nil {
		log.Fatalln(err)
	}

	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			_, err := conn.WriteMessages(kafka.Message{
				Topic: topic,
				Value: []byte(time.Now().Format(time.RFC3339)),
			})
			if err != nil {
				log.Fatalln(err)
			}
		case <-ctx.Done():
			return
		}
	}
}
