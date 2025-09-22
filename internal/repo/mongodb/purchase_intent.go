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

type PurchaseIntentRepository interface {
	Create(ctx context.Context, intent *models.PurchaseIntent) error
	GetBySessionID(ctx context.Context, sessionID primitive.ObjectID) ([]*models.PurchaseIntent, error)
	GetByRoomID(ctx context.Context, roomID string) ([]*models.PurchaseIntent, error)
}

type purchaseIntentRepo struct {
	collection *mongo.Collection
}

func NewPurchaseIntentRepository(db *DB) PurchaseIntentRepository {
	return &purchaseIntentRepo{
		collection: db.Database.Collection("purchase_intents"),
	}
}

func (r *purchaseIntentRepo) Create(ctx context.Context, intent *models.PurchaseIntent) error {
	intent.ID = primitive.NewObjectID()
	intent.CreatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, intent)
	if err != nil {
		return fmt.Errorf("failed to create purchase intent: %w", err)
	}
	return nil
}

func (r *purchaseIntentRepo) GetBySessionID(ctx context.Context, sessionID primitive.ObjectID) ([]*models.PurchaseIntent, error) {
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

func (r *purchaseIntentRepo) GetByRoomID(ctx context.Context, roomID string) ([]*models.PurchaseIntent, error) {
	filter := bson.M{"room_id": roomID}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get purchase intents by room: %w", err)
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
