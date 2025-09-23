package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/nguyentranbao-ct/chat-bot/internal/models"
)

type AuthTokenRepository interface {
	Create(ctx context.Context, token *models.AuthToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*models.AuthToken, error)
	RevokeToken(ctx context.Context, tokenHash string) error
	DeleteExpiredTokens(ctx context.Context) error
	GetUserTokens(ctx context.Context, userID primitive.ObjectID) ([]*models.AuthToken, error)
	RevokeUserTokens(ctx context.Context, userID primitive.ObjectID) error
}

type authTokenRepo struct {
	collection *mongo.Collection
}

func NewAuthTokenRepository(db *DB) AuthTokenRepository {
	return &authTokenRepo{
		collection: db.Database.Collection("auth_tokens"),
	}
}

func (r *authTokenRepo) Create(ctx context.Context, token *models.AuthToken) error {
	token.ID = primitive.NewObjectID()
	token.CreatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, token)
	if err != nil {
		return fmt.Errorf("failed to create auth token: %w", err)
	}
	return nil
}

func (r *authTokenRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*models.AuthToken, error) {
	var token models.AuthToken
	err := r.collection.FindOne(ctx, bson.M{"token_hash": tokenHash}).Decode(&token)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("token not found")
		}
		return nil, fmt.Errorf("failed to get token: %w", err)
	}
	return &token, nil
}

func (r *authTokenRepo) RevokeToken(ctx context.Context, tokenHash string) error {
	filter := bson.M{"token_hash": tokenHash}
	update := bson.M{"$set": bson.M{"is_revoked": true}}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("token not found")
	}

	return nil
}

func (r *authTokenRepo) DeleteExpiredTokens(ctx context.Context) error {
	filter := bson.M{
		"$or": []bson.M{
			{"expires_at": bson.M{"$lt": time.Now()}},
			{"is_revoked": true},
		},
	}

	_, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete expired tokens: %w", err)
	}
	return nil
}

func (r *authTokenRepo) GetUserTokens(ctx context.Context, userID primitive.ObjectID) ([]*models.AuthToken, error) {
	filter := bson.M{
		"user_id":    userID,
		"is_revoked": false,
		"expires_at": bson.M{"$gt": time.Now()},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get user tokens: %w", err)
	}
	defer cursor.Close(ctx)

	var tokens []*models.AuthToken
	for cursor.Next(ctx) {
		var token models.AuthToken
		if err := cursor.Decode(&token); err != nil {
			return nil, fmt.Errorf("failed to decode token: %w", err)
		}
		tokens = append(tokens, &token)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return tokens, nil
}

func (r *authTokenRepo) RevokeUserTokens(ctx context.Context, userID primitive.ObjectID) error {
	filter := bson.M{"user_id": userID, "is_revoked": false}
	update := bson.M{"$set": bson.M{"is_revoked": true}}

	_, err := r.collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to revoke user tokens: %w", err)
	}
	return nil
}
