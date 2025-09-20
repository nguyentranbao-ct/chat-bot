package kafka

import (
	"context"

	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/usecase"
)

// messageHandler adapts the usecase to implement the Kafka MessageHandler interface
type messageHandler struct {
	messageUsecase usecase.MessageUsecase
}

// NewMessageHandler creates a new Kafka message handler
func NewMessageHandler(messageUsecase usecase.MessageUsecase) MessageHandler {
	return &messageHandler{
		messageUsecase: messageUsecase,
	}
}

// HandleMessage processes an incoming message from Kafka
func (h *messageHandler) HandleMessage(ctx context.Context, message *models.IncomingMessage) error {
	return h.messageUsecase.ProcessMessage(ctx, message)
}