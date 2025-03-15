package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/cesarFuhr/kafka-go-demo/cmd/app/kafka/message"
	"github.com/segmentio/kafka-go"
)

func StartConsumerGroup(ctx context.Context, cfg Cfg) error {
	log.Printf("Consumer Group Config: %+v", cfg)

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
		Topic:             cfg.RetryTopic,
		NumPartitions:     10,
		ReplicationFactor: 1,
	}); err != nil {
		return fmt.Errorf("creating topic: %w", err)
	}

	// Build the writer.
	writer := kafka.NewWriter(kafka.WriterConfig{
		Topic:     cfg.RetryTopic,
		Brokers:   []string{"kafka:9092"},
		Dialer:    &dialer,
		Async:     false,
		BatchSize: 1,
	})

	var wg sync.WaitGroup
	for i := 0; i < cfg.GroupSize; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			reader := kafka.NewReader(kafka.ReaderConfig{
				Brokers:     []string{"kafka:9092"},
				GroupID:     "onlygroup",
				GroupTopics: []string{cfg.MainTopic},
				MaxWait:     time.Millisecond * 500,
				Dialer:      &dialer,
			})

			for {
				select {
				case <-ctx.Done():
					return
				default:
					kafkaMessage, err := reader.FetchMessage(ctx)
					if err != nil {
						log.Println(fmt.Errorf("consumer %d\t| error reading message %w: ", i, err))
					}

					var m message.Message
					if err := json.Unmarshal(kafkaMessage.Value, &m); err != nil {
						return
					}

					// Pretend we are doing someting.
					time.Sleep(time.Duration(rand.IntN(cfg.MaxWorkInMilliseconds)) * time.Millisecond)

					if rand.IntN(100) < cfg.FailPercentage {
						m := m
						m.Attempts += 1

						// Write to the retry topic if failed.
						log.Printf("consumer %02d | partition: %02d | offset: %04d | failed_message: %+v\n", i, kafkaMessage.Partition, kafkaMessage.Offset, m)

						bts, err := json.Marshal(m)
						if err != nil {
							log.Printf("writing messages: %w", err)
							continue
						}
						err = writer.WriteMessages(ctx, kafka.Message{
							Value: bts,
						})
						if err != nil {
							log.Printf("consumer %02d | error writing message to retry: %d : %+v", i, kafkaMessage.Offset, m)
						}
						log.Println("sent to retry: ", kafkaMessage.Offset)
					}

					log.Printf("consumer %02d | partition: %02d | offset: %04d | message: %+v\n", i, kafkaMessage.Partition, kafkaMessage.Offset, m)

					if err := reader.CommitMessages(ctx, kafkaMessage); err != nil {
						log.Printf("consumer %02d | error commiting message: %d : %+v", i, kafkaMessage.Offset, m)
						continue
					}
				}
			}
		}(i)
	}

	wg.Wait()

	return nil
}

type Cfg struct {
	MainTopic             string
	RetryTopic            string
	GroupSize             int
	MaxWaitMilliseconds   int
	MaxWorkInMilliseconds int
	FailPercentage        int
}

func LoadCfg(prefix string) Cfg {
	c := Cfg{
		MainTopic:             readStringEnv("my-topic", prefix+"_CONSUMER_MAIN_TOPIC"),
		RetryTopic:            readStringEnv("retry-topic", prefix+"_CONSUMER_RETRY_TOPIC"),
		GroupSize:             readIntEnv(10, prefix+"_CONSUMER_MAIN_GROUP_SIZE"),
		MaxWaitMilliseconds:   readIntEnv(500, prefix+"_CONSUMER_MAX_WAIT_IN_MILLISECONDS"),
		MaxWorkInMilliseconds: readIntEnv(500, prefix+"_CONSUMER_MAX_WORK_IN_MILLISECONDS"),
		FailPercentage:        readIntEnv(0, prefix+"_CONSUMER_FAIL_PERCENTAGE"),
	}
	return c
}

func readIntEnv(defaultValue int, name string) int {
	v := os.Getenv(name)
	if v != "" {
		value, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			log.Println("failed reading [", name, "]", err)
			return defaultValue
		}
		return int(value)
	}
	return defaultValue
}

func readStringEnv(defaultValue string, name string) string {
	v := os.Getenv(name)
	if v != "" {
		return v
	}
	return defaultValue
}
