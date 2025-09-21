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

type UserAttributeRepository interface {
	Create(ctx context.Context, attr *models.UserAttribute) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.UserAttribute, error)
	GetByUserID(ctx context.Context, userID primitive.ObjectID) ([]*models.UserAttribute, error)
	GetByUserIDAndKey(ctx context.Context, userID primitive.ObjectID, key string) (*models.UserAttribute, error)
	GetByKey(ctx context.Context, key string) ([]*models.UserAttribute, error)
	GetByTags(ctx context.Context, tags []string) ([]*models.UserAttribute, error)
	GetByUserIDAndTags(ctx context.Context, userID primitive.ObjectID, tags []string) ([]*models.UserAttribute, error)
	Update(ctx context.Context, attr *models.UserAttribute) error
	Upsert(ctx context.Context, attr *models.UserAttribute) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	DeleteByUserIDAndKey(ctx context.Context, userID primitive.ObjectID, key string) error
}

type userAttributeRepo struct {
	collection *mongo.Collection
}

func NewUserAttributeRepository(db *DB) UserAttributeRepository {
	return &userAttributeRepo{
		collection: db.Database.Collection("user_attributes"),
	}
}

func (r *userAttributeRepo) Create(ctx context.Context, attr *models.UserAttribute) error {
	attr.ID = primitive.NewObjectID()
	attr.CreatedAt = time.Now()
	attr.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, attr)
	if err != nil {
		return fmt.Errorf("failed to create user attribute: %w", err)
	}
	return nil
}

func (r *userAttributeRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*models.UserAttribute, error) {
	var attr models.UserAttribute
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&attr)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user attribute not found")
		}
		return nil, fmt.Errorf("failed to get user attribute: %w", err)
	}
	return &attr, nil
}

func (r *userAttributeRepo) GetByUserID(ctx context.Context, userID primitive.ObjectID) ([]*models.UserAttribute, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, fmt.Errorf("failed to get user attributes: %w", err)
	}
	defer cursor.Close(ctx)

	var attrs []*models.UserAttribute
	for cursor.Next(ctx) {
		var attr models.UserAttribute
		if err := cursor.Decode(&attr); err != nil {
			return nil, fmt.Errorf("failed to decode user attribute: %w", err)
		}
		attrs = append(attrs, &attr)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return attrs, nil
}

func (r *userAttributeRepo) GetByUserIDAndKey(ctx context.Context, userID primitive.ObjectID, key string) (*models.UserAttribute, error) {
	var attr models.UserAttribute
	err := r.collection.FindOne(ctx, bson.M{
		"user_id": userID,
		"key":     key,
	}).Decode(&attr)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user attribute: %w", err)
	}
	return &attr, nil
}

func (r *userAttributeRepo) GetByKey(ctx context.Context, key string) ([]*models.UserAttribute, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"key": key})
	if err != nil {
		return nil, fmt.Errorf("failed to get user attributes by key: %w", err)
	}
	defer cursor.Close(ctx)

	var attrs []*models.UserAttribute
	for cursor.Next(ctx) {
		var attr models.UserAttribute
		if err := cursor.Decode(&attr); err != nil {
			return nil, fmt.Errorf("failed to decode user attribute: %w", err)
		}
		attrs = append(attrs, &attr)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return attrs, nil
}

func (r *userAttributeRepo) GetByTags(ctx context.Context, tags []string) ([]*models.UserAttribute, error) {
	filter := bson.M{"tags": bson.M{"$in": tags}}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get user attributes by tags: %w", err)
	}
	defer cursor.Close(ctx)

	var attrs []*models.UserAttribute
	for cursor.Next(ctx) {
		var attr models.UserAttribute
		if err := cursor.Decode(&attr); err != nil {
			return nil, fmt.Errorf("failed to decode user attribute: %w", err)
		}
		attrs = append(attrs, &attr)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return attrs, nil
}

func (r *userAttributeRepo) GetByUserIDAndTags(ctx context.Context, userID primitive.ObjectID, tags []string) ([]*models.UserAttribute, error) {
	filter := bson.M{
		"user_id": userID,
		"tags":    bson.M{"$in": tags},
	}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get user attributes by user ID and tags: %w", err)
	}
	defer cursor.Close(ctx)

	var attrs []*models.UserAttribute
	for cursor.Next(ctx) {
		var attr models.UserAttribute
		if err := cursor.Decode(&attr); err != nil {
			return nil, fmt.Errorf("failed to decode user attribute: %w", err)
		}
		attrs = append(attrs, &attr)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return attrs, nil
}

func (r *userAttributeRepo) Update(ctx context.Context, attr *models.UserAttribute) error {
	attr.UpdatedAt = time.Now()

	filter := bson.M{"_id": attr.ID}
	update := bson.M{"$set": attr}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update user attribute: %w", err)
	}
	return nil
}

func (r *userAttributeRepo) Upsert(ctx context.Context, attr *models.UserAttribute) error {
	now := time.Now()
	attr.UpdatedAt = now

	filter := bson.M{
		"user_id": attr.UserID,
		"key":     attr.Key,
	}

	update := bson.M{
		"$set": bson.M{
			"value":      attr.Value,
			"tags":       attr.Tags,
			"updated_at": now,
		},
		"$setOnInsert": bson.M{
			"_id":        primitive.NewObjectID(),
			"created_at": now,
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("failed to upsert user attribute: %w", err)
	}
	return nil
}

func (r *userAttributeRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete user attribute: %w", err)
	}
	return nil
}

func (r *userAttributeRepo) DeleteByUserIDAndKey(ctx context.Context, userID primitive.ObjectID, key string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{
		"user_id": userID,
		"key":     key,
	})
	if err != nil {
		return fmt.Errorf("failed to delete user attribute: %w", err)
	}
	return nil
}