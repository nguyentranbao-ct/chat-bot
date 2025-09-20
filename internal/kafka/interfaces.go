package kafka

import (
	"context"

	"github.com/nguyentranbao-ct/chat-bot/internal/models"
)

// Consumer defines the interface for Kafka message consumption
type Consumer interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// MessageHandler defines the interface for handling incoming Kafka messages
type MessageHandler interface {
	HandleMessage(ctx context.Context, message *models.IncomingMessage) error
}

// WhitelistService defines the interface for channel whitelist management
type WhitelistService interface {
	IsChannelAllowed(channelID string) bool
}
