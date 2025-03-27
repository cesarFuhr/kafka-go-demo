package consumer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"math/rand/v2"

	"github.com/cesarFuhr/kafka-go-demo/cmd/app/kafka/message"
	"github.com/lib/pq"
)

func NewRetryWork(ctx context.Context) (work func(context.Context, Cfg, message.Message[message.Retry]) error, close func() error) {
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

	// TODO:
	// Audience column/lkp to controll the audience
	// field when retrying.
	createQuery := `
  CREATE TABLE IF NOT EXISTS retries (
    retry_id          SERIAL PRIMARY KEY,
    delivered_at      INT,
    backoff_deadline  INT,
    destination_topic TEXT,
    audience          VARCHAR(255)[],
    message           JSONB
  );
  `

	_, err = db.ExecContext(ctx, createQuery)
	if err != nil {
		panic(err)
	}

	return func(ctx context.Context, cfg Cfg, m message.Message[message.Retry]) error {
		if rand.IntN(100) < cfg.FailPercentage {
			return errors.New("big fail oh no")
		}

		insertQuery := `
    INSERT INTO retries (
      backoff_deadline,
      destination_topic,
      audience,
      message
    ) VALUES (
      $1,
      $2,
      $3,
      $4
    );
    `

		messageBytes, err := json.Marshal(m.Value.Message)
		if err != nil {
			panic(err)
		}

		audience := pq.StringArray(m.Value.Audience)

		res, err := db.ExecContext(ctx, insertQuery, m.Value.BackoffDeadline, m.Value.DestinationTopic, audience, messageBytes)
		if err != nil {
			panic(err)
		}

		if affected, err := res.RowsAffected(); err != nil || affected < 1 {
			log.Println("retrier failed to insert a new record")
		}

		return nil
	}, db.Close
}
