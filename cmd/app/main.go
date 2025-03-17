package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/cesarFuhr/kafka-go-demo/cmd/app/kafka/consumer"
	"github.com/cesarFuhr/kafka-go-demo/cmd/app/kafka/producer"
)

// TODO:
// - Mermaid chart with the consumer and retriers flows.
// - Figure out dead lettering.

func main() {
	if err := run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	if len(args) <= 1 {
		return fmt.Errorf("nothing to do here, missing operation mode: %v", args)
	}

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer cancel()

		done := make(chan os.Signal, 1)
		signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
		<-done
	}()

	switch args[1] {
	case producerMode:
		runPublisher(ctx)
	case mainConsumerMode:
		runMainConsumerGroup(ctx)
	case retryConsumerMode:
		runRetryConsumerGroup(ctx)
	case retryProducerMode:
		runRetryProducer(ctx)
	default:
		return fmt.Errorf("nothing to do here, invalid operation: %v", args)
	}

	return nil
}

const producerMode = "producer"

func runPublisher(ctx context.Context) {
	log.Println("Starting the producer...")
	err := producer.StartProducer(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Stopping the producer...")
}

const mainConsumerMode = "consumer"

func runMainConsumerGroup(ctx context.Context) {
	log.Println("Starting the consumer...")
	cfg := consumer.LoadCfg("MAIN")
	err := consumer.StartConsumerGroup(ctx, cfg, consumer.MainWork)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Stopping the consumer...")
}

const retryConsumerMode = "retrier"

func runRetryConsumerGroup(ctx context.Context) {
	log.Println("Starting the consumer...")
	cfg := consumer.LoadCfg("RETRY")

	retryWork, close := consumer.NewRetryWork(ctx)
	defer close()

	err := consumer.StartConsumerGroup(ctx, cfg, retryWork)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Stopping the consumer...")
}

const retryProducerMode = "retrier_producer"

func runRetryProducer(ctx context.Context) {
	log.Println("Starting the producer...")

	err := producer.StartRetryProducer(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Stopping the producer...")
}
