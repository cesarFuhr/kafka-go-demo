package consumer

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

func StartConsumers(ctx context.Context) error {
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			reader := kafka.NewReader(kafka.ReaderConfig{
				Brokers:     []string{"kafka:9092"},
				GroupID:     "onlygroup",
				GroupTopics: []string{"my-topic"},
				MaxWait:     time.Second,
				Dialer: &kafka.Dialer{
					Timeout:   3 * time.Second,
					DualStack: true,
				},
			})

			for {
				select {
				case <-ctx.Done():
					return
				default:
					message, err := reader.FetchMessage(ctx)
					if err != nil {
						log.Println(fmt.Errorf("consumer %d | error reading message %w: ", i, err))
					}

					log.Printf("consumer %d | message: %s\n", i, string(message.Value))

					// Pretend we are doing someting.
					time.Sleep(time.Duration(rand.IntN(1000)) * time.Millisecond)

					if err := reader.CommitMessages(ctx, message); err != nil {
						log.Printf("consumer %d | error commiting message: %d : %s", i, message.Offset, string(message.Value))
						continue
					}
				}
			}
		}(i)
	}

	wg.Wait()

	return nil
}
