package usecase

import (
	"context"

	"github.com/nguyentranbao-ct/chat-bot/internal/models"
)

type MessageUsecase interface {
	ProcessMessage(ctx context.Context, message *models.IncomingMessage) error
}
