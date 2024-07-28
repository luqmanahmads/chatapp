package chat

import (
	"context"

	"github.com/luqmanahmads/chatapp/internal/entity"
)

type chatRepo interface {
	SendChat(context.Context, entity.ChatMessage) error
	ReadChat(ctx context.Context, receiver string) (chan entity.ChatMessage, error)
}

type ChatUsecase struct {
	chatRepo chatRepo
}

func New(publishRepo chatRepo) *ChatUsecase {
	return &ChatUsecase{
		chatRepo: publishRepo,
	}
}

func (pu *ChatUsecase) SendChat(ctx context.Context, message entity.ChatMessage) error {
	return pu.chatRepo.SendChat(ctx, message)
}

func (pu *ChatUsecase) ReadChat(ctx context.Context, message entity.ChatMessage) (chan entity.ChatMessage, error) {
	return pu.chatRepo.ReadChat(ctx, message.Receiver)
}
