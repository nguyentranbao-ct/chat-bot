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

type PurchaseIntentRepo struct {
	collection *mongo.Collection
}

func NewPurchaseIntentRepository(db *DB) *PurchaseIntentRepo {
	return &PurchaseIntentRepo{
		collection: db.database.Collection("purchase_intents"),
	}
}

func (r *PurchaseIntentRepo) Create(ctx context.Context, intent *models.PurchaseIntent) error {
	intent.ID = primitive.NewObjectID()
	intent.CreatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, intent)
	if err != nil {
		return fmt.Errorf("failed to create purchase intent: %w", err)
	}
	return nil
}

func (r *PurchaseIntentRepo) GetBySessionID(ctx context.Context, sessionID primitive.ObjectID) ([]*models.PurchaseIntent, error) {
	filter := bson.M{"session_id": sessionID}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get purchase intents by session: %w", err)
	}
	defer cursor.Close(ctx)

	var intents []*models.PurchaseIntent
	for cursor.Next(ctx) {
		var intent models.PurchaseIntent
		if err := cursor.Decode(&intent); err != nil {
			return nil, fmt.Errorf("failed to decode purchase intent: %w", err)
		}
		intents = append(intents, &intent)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return intents, nil
}

func (r *PurchaseIntentRepo) GetByChannelID(ctx context.Context, channelID string) ([]*models.PurchaseIntent, error) {
	filter := bson.M{"channel_id": channelID}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get purchase intents by channel: %w", err)
	}
	defer cursor.Close(ctx)

	var intents []*models.PurchaseIntent
	for cursor.Next(ctx) {
		var intent models.PurchaseIntent
		if err := cursor.Decode(&intent); err != nil {
			return nil, fmt.Errorf("failed to decode purchase intent: %w", err)
		}
		intents = append(intents, &intent)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return intents, nil
}
