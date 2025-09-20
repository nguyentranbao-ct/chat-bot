package repository

import (
	"context"

	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChatModeRepository interface {
	GetByName(ctx context.Context, name string) (*models.ChatMode, error)
	Create(ctx context.Context, mode *models.ChatMode) error
	Update(ctx context.Context, mode *models.ChatMode) error
	Upsert(ctx context.Context, mode *models.ChatMode) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	List(ctx context.Context) ([]*models.ChatMode, error)
}

type ChatSessionRepository interface {
	Create(ctx context.Context, session *models.ChatSession) error
	GetByChannelAndUser(ctx context.Context, channelID, userID string) (*models.ChatSession, error)
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.ChatSession, error)
	Update(ctx context.Context, session *models.ChatSession) error
	EndSession(ctx context.Context, id primitive.ObjectID) error
	ListActiveSessions(ctx context.Context) ([]*models.ChatSession, error)
}

type ChatActivityRepository interface {
	Create(ctx context.Context, activity *models.ChatActivity) error
	GetBySessionID(ctx context.Context, sessionID primitive.ObjectID) ([]*models.ChatActivity, error)
	GetByChannelID(ctx context.Context, channelID string, limit int) ([]*models.ChatActivity, error)
}

type PurchaseIntentRepository interface {
	Create(ctx context.Context, intent *models.PurchaseIntent) error
	GetBySessionID(ctx context.Context, sessionID primitive.ObjectID) ([]*models.PurchaseIntent, error)
	GetByChannelID(ctx context.Context, channelID string) ([]*models.PurchaseIntent, error)
}
