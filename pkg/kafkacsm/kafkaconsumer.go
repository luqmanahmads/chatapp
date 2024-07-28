package kafkascm

import (
	"context"
	"fmt"
	"log"

	"github.com/segmentio/kafka-go"
)

type Handler func(kafka.Message)

type KafkaHandler struct {
	config  kafka.ReaderConfig
	handler Handler
	reader  *kafka.Reader
}

type KafkaConsumer struct {
	handlers []*KafkaHandler
}

func New() *KafkaConsumer {
	return &KafkaConsumer{}
}

func (kc *KafkaConsumer) Register(cfg kafka.ReaderConfig, handler Handler) {
	kc.handlers = append(kc.handlers, &KafkaHandler{
		config:  cfg,
		handler: handler,
	})
}

func (kc *KafkaConsumer) Listen(ctx context.Context) {
	for _, kh := range kc.handlers {
		r := kafka.NewReader(kh.config)
		kh.reader = r

		go func(r *kafka.Reader) {
			log.Printf("starting to listen kafka topic: %s partition: %d...", kh.config.Topic, kh.config.Partition)
			for {
				m, err := r.ReadMessage(ctx)
				if err != nil {
					break
				}

				fmt.Printf("message at topic/partition/offset %v/%v/%v: %s = %s\n", m.Topic, m.Partition, m.Offset, string(m.Key), string(m.Value))

				kh.handler(m)
			}
		}(r)
	}
}

func (kc *KafkaConsumer) Shutdown() {
	for _, kh := range kc.handlers {
		kh.reader.Close()
	}
}
