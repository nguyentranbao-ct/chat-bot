package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ChatActivityRepo struct {
	collection *mongo.Collection
}

func NewChatActivityRepository(db *DB) *ChatActivityRepo {
	return &ChatActivityRepo{
		collection: db.database.Collection("chat_activities"),
	}
}

func (r *ChatActivityRepo) Create(ctx context.Context, activity *models.ChatActivity) error {
	activity.ID = primitive.NewObjectID()
	activity.ExecutedAt = time.Now()
	activity.CreatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, activity)
	if err != nil {
		return fmt.Errorf("failed to create chat activity: %w", err)
	}
	return nil
}

func (r *ChatActivityRepo) GetBySessionID(ctx context.Context, sessionID primitive.ObjectID) ([]*models.ChatActivity, error) {
	filter := bson.M{"session_id": sessionID}
	opts := options.Find().SetSort(bson.D{{Key: "executed_at", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get activities by session: %w", err)
	}
	defer cursor.Close(ctx)

	var activities []*models.ChatActivity
	for cursor.Next(ctx) {
		var activity models.ChatActivity
		if err := cursor.Decode(&activity); err != nil {
			return nil, fmt.Errorf("failed to decode chat activity: %w", err)
		}
		activities = append(activities, &activity)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return activities, nil
}

func (r *ChatActivityRepo) GetByChannelID(ctx context.Context, channelID string, limit int) ([]*models.ChatActivity, error) {
	filter := bson.M{"channel_id": channelID}
	opts := options.Find().
		SetSort(bson.D{{Key: "executed_at", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get activities by channel: %w", err)
	}
	defer cursor.Close(ctx)

	var activities []*models.ChatActivity
	for cursor.Next(ctx) {
		var activity models.ChatActivity
		if err := cursor.Decode(&activity); err != nil {
			return nil, fmt.Errorf("failed to decode chat activity: %w", err)
		}
		activities = append(activities, &activity)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return activities, nil
}
