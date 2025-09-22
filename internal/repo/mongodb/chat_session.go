package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ChatSessionRepository interface {
	Create(ctx context.Context, session *models.ChatSession) error
	GetByChannelAndUser(ctx context.Context, channelID, userID string) (*models.ChatSession, error)
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.ChatSession, error)
	Update(ctx context.Context, session *models.ChatSession) error
	EndSession(ctx context.Context, id primitive.ObjectID) error
	ListActiveSessions(ctx context.Context) ([]*models.ChatSession, error)
}

type chatSessionRepo struct {
	collection *mongo.Collection
}

func NewChatSessionRepository(db *DB) ChatSessionRepository {
	return &chatSessionRepo{
		collection: db.Database.Collection("chat_sessions"),
	}
}

func (r *chatSessionRepo) Create(ctx context.Context, session *models.ChatSession) error {
	session.ID = primitive.NewObjectID()
	session.CreatedAt = time.Now()
	session.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, session)
	if err != nil {
		return fmt.Errorf("failed to create chat session: %w", err)
	}
	return nil
}

func (r *chatSessionRepo) GetByChannelAndUser(ctx context.Context, channelID, userID string) (*models.ChatSession, error) {
	filter := bson.M{
		"channel_id": channelID,
		"user_id":    userID,
		"status":     models.SessionStatusActive,
	}

	var session models.ChatSession
	err := r.collection.FindOne(ctx, filter).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get chat session: %w", err)
	}
	return &session, nil
}

func (r *chatSessionRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*models.ChatSession, error) {
	var session models.ChatSession
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("chat session not found")
		}
		return nil, fmt.Errorf("failed to get chat session: %w", err)
	}
	return &session, nil
}

func (r *chatSessionRepo) Update(ctx context.Context, session *models.ChatSession) error {
	session.UpdatedAt = time.Now()

	filter := bson.M{"_id": session.ID}
	update := bson.M{"$set": session}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update chat session: %w", err)
	}
	return nil
}

func (r *chatSessionRepo) EndSession(ctx context.Context, id primitive.ObjectID) error {
	now := time.Now()
	filter := bson.M{"_id": id}
	update := bson.M{
		"$set": bson.M{
			"status":     models.SessionStatusEnded,
			"ended_at":   now,
			"updated_at": now,
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to end chat session: %w", err)
	}
	return nil
}

func (r *chatSessionRepo) ListActiveSessions(ctx context.Context) ([]*models.ChatSession, error) {
	filter := bson.M{"status": models.SessionStatusActive}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list active sessions: %w", err)
	}
	defer cursor.Close(ctx)

	var sessions []*models.ChatSession
	for cursor.Next(ctx) {
		var session models.ChatSession
		if err := cursor.Decode(&session); err != nil {
			return nil, fmt.Errorf("failed to decode chat session: %w", err)
		}
		sessions = append(sessions, &session)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return sessions, nil
}
