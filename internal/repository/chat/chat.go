package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/luqmanahmads/chatapp/internal/entity"
	"github.com/nsqio/go-nsq"
)

const (
	topicName   = "chat-%s" // pattern: chat-msg:[receiver]
	channelRead = "read"
)

type nsqProducer interface {
	Publish(topic string, body []byte) error
}

type chatRepo struct {
	nsqProducer nsqProducer
}

func New(nsqProducer nsqProducer) *chatRepo {
	return &chatRepo{
		nsqProducer: nsqProducer,
	}
}

func (p *chatRepo) SendChat(ctx context.Context, chatMsg entity.ChatMessage) error {
	msg, err := json.Marshal(chatMsg)
	if err != nil {
		return err
	}

	return p.nsqProducer.Publish(fmt.Sprintf(topicName, chatMsg.Receiver), msg)
}

func (p *chatRepo) ReadChat(ctx context.Context, receiver string) (chan entity.ChatMessage, error) {
	msgChan := make(chan entity.ChatMessage)

	cfg := nsq.NewConfig()
	consumer, err := nsq.NewConsumer(fmt.Sprintf(topicName, receiver), channelRead, cfg)
	if err != nil {
		return nil, err
	}

	consumer.AddHandler(nsq.HandlerFunc(func(message *nsq.Message) error {
		var chatMsg entity.ChatMessage
		err = json.Unmarshal(message.Body, &chatMsg)
		if err != nil {
			return err
		}

		msgChan <- chatMsg
		return nil
	}))

	err = consumer.ConnectToNSQLookupd("localhost:4161")
	if err != nil {
		return nil, err
	}

	log.Printf("start consuming for receiver: %s...", receiver)

	go func() {
		defer close(msgChan)
		<-ctx.Done()
		log.Printf("stopping consumer for receiver: %s...", receiver)
		consumer.Stop()
	}()

	return msgChan, nil
}
