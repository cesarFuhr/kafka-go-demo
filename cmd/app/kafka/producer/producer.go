package producer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cesarFuhr/kafka-go-demo/cmd/app/kafka/message"
	_ "github.com/glebarez/go-sqlite"
	"github.com/lib/pq"
	"github.com/segmentio/kafka-go"
)

const topic = "my-topic"

func StartProducer(ctx context.Context) error {
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
	}, kafka.TopicConfig{
		Topic:             "retries_" + topic,
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
			message := message.Message[uint]{
				Timestamp:   now.Unix(),
				Attempts:    0,
				MaxAttempts: 3,
				Value:       counter,
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

func StartRetryProducer(ctx context.Context) error {
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

	// Define connection string
	connStr := "user=postgres password=root dbname=postgres host=db port=5432 sslmode=disable"

	// Open database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}

	// Ping database to verify connection
	err = db.Ping()
	if err != nil {
		panic(err)
	}

	retrierTicker := time.NewTicker(time.Second * 5)
	defer retrierTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-retrierTicker.C:
			func() {
				now := time.Now()
				insertQuery := `
        SELECT 
          r.retry_id,
          r.backoff_deadline,
          r.destination_topic,
          r.audience,
          r.message
        FROM retries r
        WHERE
          r.backoff_deadline < $1
          AND r.delivered_at IS NULL 
        ORDER BY r.destination_topic ASC
        LIMIT 100;`
				rows, err := db.QueryContext(ctx, insertQuery, now.Unix())
				if err != nil {
					panic(err)
				}
				defer rows.Close()

				type dbRetry struct {
					RetryID          int             `db:"retry_id"`
					BackoffDeadline  int             `db:"backoff_deadline"`
					DestinationTopic string          `db:"destination_topic"`
					Audience         pq.StringArray  `db:"audience"`
					Message          json.RawMessage `db:"message"`
				}

				retries := make([]dbRetry, 0, 100)
				for rows.Next() {
					var r dbRetry
					err := rows.Scan(&r.RetryID, &r.BackoffDeadline, &r.DestinationTopic, &r.Audience, &r.Message)
					if err != nil {
						log.Println("failed to scan: ", err)
						return
					}

					retries = append(retries, r)
				}

				if len(retries) == 0 {
					// Nothing to do here.
					log.Println("Nothing do do this time.")
					return
				}

				log.Println("Oh, we got some retries to write... go go go")

				retryIDs := make([]string, 0, 100)
				writers := map[string]*kafka.Writer{}
				for _, r := range retries {
					writer, ok := writers[r.DestinationTopic]
					if !ok {
						// Build the destination topic writer.
						writer = kafka.NewWriter(kafka.WriterConfig{
							Topic:     topic,
							Brokers:   []string{"kafka:9092"},
							Dialer:    &dialer,
							Async:     false,
							BatchSize: 1,
						})
					}

					var baseMessage message.Message[any]
					err := json.Unmarshal(r.Message, &baseMessage)
					if err != nil {
						log.Println("failed to unmarshal: ", err)
						return
					}

					baseMessage.Attempts += 1

					bts, err := json.Marshal(baseMessage)
					if err != nil {
						log.Println("failed to marshal: ", err)
						return
					}

					kafkaMessage := kafka.Message{
						Headers: []kafka.Header{{
							Key: "audience", Value: []byte(strings.Join(r.Audience, ",")),
						}},
						Value: bts,
					}
					err = writer.WriteMessages(ctx, kafkaMessage)
					if err != nil {
						log.Println("writing messages: %w", err)
						return
					}

					log.Printf("sent: destination: %s | retry: %v | audience: %v | message: %+v", r.DestinationTopic, r.RetryID, string(kafkaMessage.Headers[0].Value), string(bts))
					retryIDs = append(retryIDs, fmt.Sprint(r.RetryID))
				}

				deliveredQuery := `
        UPDATE retries SET
          delivered_at = $1
        WHERE
          retry_id IN (%s);
        `
				res, err := db.ExecContext(ctx, fmt.Sprintf(deliveredQuery, strings.Join(retryIDs, ",")), time.Now().Unix())
				if err != nil {
					log.Printf("writing to the database: %w", err)
				}

				if affected, err := res.RowsAffected(); err != nil || int(affected) != len(retryIDs) {
					log.Println("checking rows affected: %w", err)
				}
			}()
		}
	}
}
