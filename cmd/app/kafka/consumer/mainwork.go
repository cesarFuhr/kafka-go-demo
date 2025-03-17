package consumer

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/cesarFuhr/kafka-go-demo/cmd/app/kafka/message"
)

func MainWork(_ context.Context, cfg Cfg, _ message.Message[uint]) error {
	time.Sleep(time.Duration(rand.IntN(cfg.MaxWorkInMilliseconds)) * time.Millisecond)

	if rand.IntN(100) < cfg.FailPercentage {
		return ErrShouldRetry
	}
	return nil
}
