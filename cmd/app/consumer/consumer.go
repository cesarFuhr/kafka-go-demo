package consumer

import (
	"context"
	"log"
	"time"
)

func NewConsumer(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	var ticks int
	for {
		select {
		case <-ticker.C:
			ticks += 1
			log.Println("Consumer tick: ", ticks)
		case <-ctx.Done():
			return
		}
	}
}
