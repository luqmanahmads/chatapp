package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	kafka "github.com/segmentio/kafka-go"
)

func main() {
	w := &kafka.Writer{
		Addr:  kafka.TCP("localhost:9092"),
		Topic: "quickstart-events",
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	for {
		time.Sleep(1 * time.Second)

		currTime := time.Now().Unix()
		writeTimeout(context.Background(), w, kafka.Message{
			Key:   []byte("Key-A"),
			Value: []byte(fmt.Sprintf("%d", currTime)),
		})

		log.Println("a message has been written successfully.")

		select {
		case <-signals:
			log.Println("program exits by interupt...")
			return
		default:

		}
	}
}

func writeTimeout(ctx context.Context, w *kafka.Writer, m kafka.Message) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	return w.WriteMessages(ctx, m)
}
