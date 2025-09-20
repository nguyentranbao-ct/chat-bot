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

type ChatModeRepo struct {
	collection *mongo.Collection
}

func NewChatModeRepository(db *DB) *ChatModeRepo {
	return &ChatModeRepo{
		collection: db.Database.Collection("chat_modes"),
	}
}

func (r *ChatModeRepo) GetByName(ctx context.Context, name string) (*models.ChatMode, error) {
	var mode models.ChatMode
	err := r.collection.FindOne(ctx, bson.M{"name": name}).Decode(&mode)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("chat mode '%s' not found", name)
		}
		return nil, fmt.Errorf("failed to get chat mode: %w", err)
	}
	return &mode, nil
}

func (r *ChatModeRepo) Create(ctx context.Context, mode *models.ChatMode) error {
	mode.ID = primitive.NewObjectID()
	mode.CreatedAt = time.Now()
	mode.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, mode)
	if err != nil {
		return fmt.Errorf("failed to create chat mode: %w", err)
	}
	return nil
}

func (r *ChatModeRepo) Update(ctx context.Context, mode *models.ChatMode) error {
	mode.UpdatedAt = time.Now()

	filter := bson.M{"_id": mode.ID}
	update := bson.M{"$set": mode}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update chat mode: %w", err)
	}
	return nil
}

func (r *ChatModeRepo) Upsert(ctx context.Context, mode *models.ChatMode) error {
	now := time.Now()

	filter := bson.M{"name": mode.Name}
	update := bson.M{
		"$set": bson.M{
			"prompt_template":      mode.PromptTemplate,
			"condition":           mode.Condition,
			"model":               mode.Model,
			"tools":               mode.Tools,
			"max_iterations":      mode.MaxIterations,
			"max_prompt_tokens":   mode.MaxPromptTokens,
			"max_response_tokens": mode.MaxResponseTokens,
			"updated_at":          now,
		},
		"$setOnInsert": bson.M{
			"_id":        primitive.NewObjectID(),
			"created_at": now,
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("failed to upsert chat mode: %w", err)
	}
	return nil
}

func (r *ChatModeRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete chat mode: %w", err)
	}
	return nil
}

func (r *ChatModeRepo) List(ctx context.Context) ([]*models.ChatMode, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to list chat modes: %w", err)
	}
	defer cursor.Close(ctx)

	var modes []*models.ChatMode
	for cursor.Next(ctx) {
		var mode models.ChatMode
		if err := cursor.Decode(&mode); err != nil {
			return nil, fmt.Errorf("failed to decode chat mode: %w", err)
		}
		modes = append(modes, &mode)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return modes, nil
}
