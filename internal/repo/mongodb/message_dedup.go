package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MessageDeduplication tracks processed messages to prevent duplicates and loops
type MessageDeduplication struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ExternalMessageID string             `bson:"external_message_id" json:"external_message_id"`
	RoomID            string             `bson:"room_id" json:"room_id"`
	PartnerName       string             `bson:"partner_name" json:"partner_name"`
	MessageHash       string             `bson:"message_hash" json:"message_hash"`
	SenderID          string             `bson:"sender_id" json:"sender_id"`
	Content           string             `bson:"content" json:"content"`
	Source            string             `bson:"source" json:"source"` // "kafka", "api", "partner"
	ProcessedAt       time.Time          `bson:"processed_at" json:"processed_at"`
	ExpiresAt         time.Time          `bson:"expires_at" json:"expires_at"` // TTL index
}

// MessageDedupRepository handles message deduplication operations
type MessageDedupRepository interface {
	IsMessageProcessed(ctx context.Context, externalMessageID, roomID, partnerName string) (bool, error)
	IsMessageDuplicate(ctx context.Context, messageHash, roomID string, withinDuration time.Duration) (bool, error)
	RecordProcessedMessage(ctx context.Context, record *MessageDeduplication) error
	CleanupExpiredRecords(ctx context.Context) error
	GetRecentMessages(ctx context.Context, roomID string, limit int, withinDuration time.Duration) ([]*MessageDeduplication, error)
}

type messageDedupRepo struct {
	collection *mongo.Collection
}

func NewMessageDedupRepository(db *DB) MessageDedupRepository {
	repo := &messageDedupRepo{
		collection: db.Database.Collection("message_deduplication"),
	}

	// Create indexes for better performance
	go repo.createIndexes(context.Background())

	return repo
}

func (r *messageDedupRepo) createIndexes(ctx context.Context) {
	// TTL index for automatic cleanup
	ttlIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "expires_at", Value: 1}},
		Options: options.Index().
			SetExpireAfterSeconds(0).
			SetName("expires_at_ttl"),
	}

	// Compound index for external message lookup
	externalMsgIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "external_message_id", Value: 1},
			{Key: "room_id", Value: 1},
			{Key: "partner_name", Value: 1},
		},
		Options: options.Index().
			SetUnique(true).
			SetName("external_msg_room_partner"),
	}

	// Compound index for message hash lookup
	hashIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "message_hash", Value: 1},
			{Key: "room_id", Value: 1},
			{Key: "processed_at", Value: -1},
		},
		Options: options.Index().SetName("hash_room_time"),
	}

	// Index for recent message queries
	recentIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "room_id", Value: 1},
			{Key: "processed_at", Value: -1},
		},
		Options: options.Index().SetName("room_recent"),
	}

	indexes := []mongo.IndexModel{ttlIndex, externalMsgIndex, hashIndex, recentIndex}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		// Log error but don't fail - indexes are for optimization
		fmt.Printf("Failed to create deduplication indexes: %v\n", err)
	}
}

func (r *messageDedupRepo) IsMessageProcessed(ctx context.Context, externalMessageID, roomID, partnerName string) (bool, error) {
	if externalMessageID == "" {
		return false, nil // Can't check without external ID
	}

	filter := bson.M{
		"external_message_id": externalMessageID,
		"room_id":             roomID,
		"partner_name":        partnerName,
	}

	count, err := r.collection.CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		return false, fmt.Errorf("failed to check message processing status: %w", err)
	}

	return count > 0, nil
}

func (r *messageDedupRepo) IsMessageDuplicate(ctx context.Context, messageHash, roomID string, withinDuration time.Duration) (bool, error) {
	if messageHash == "" {
		return false, nil // Can't check without hash
	}

	cutoffTime := time.Now().Add(-withinDuration)
	filter := bson.M{
		"message_hash": messageHash,
		"room_id":      roomID,
		"processed_at": bson.M{"$gte": cutoffTime},
	}

	count, err := r.collection.CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		return false, fmt.Errorf("failed to check message duplication: %w", err)
	}

	return count > 0, nil
}

func (r *messageDedupRepo) RecordProcessedMessage(ctx context.Context, record *MessageDeduplication) error {
	if record == nil {
		return fmt.Errorf("record cannot be nil")
	}

	// Set timestamps
	now := time.Now()
	record.ProcessedAt = now
	// Set expiration to 24 hours from now (configurable)
	record.ExpiresAt = now.Add(24 * time.Hour)

	// Generate ObjectID if not set
	if record.ID.IsZero() {
		record.ID = primitive.NewObjectID()
	}

	// Use upsert to handle potential race conditions
	filter := bson.M{
		"external_message_id": record.ExternalMessageID,
		"room_id":             record.RoomID,
		"partner_name":        record.PartnerName,
	}

	update := bson.M{
		"$setOnInsert": record,
	}

	opts := options.Update().SetUpsert(true)
	result, err := r.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to record processed message: %w", err)
	}

	// If the document was not inserted (UpsertedCount == 0), it means it already existed
	if result.UpsertedCount == 0 {
		return fmt.Errorf("message already processed: external_id=%s, room=%s, partner=%s",
			record.ExternalMessageID, record.RoomID, record.PartnerName)
	}

	return nil
}

func (r *messageDedupRepo) GetRecentMessages(ctx context.Context, roomID string, limit int, withinDuration time.Duration) ([]*MessageDeduplication, error) {
	cutoffTime := time.Now().Add(-withinDuration)
	filter := bson.M{
		"room_id":      roomID,
		"processed_at": bson.M{"$gte": cutoffTime},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "processed_at", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent messages: %w", err)
	}
	defer cursor.Close(ctx)

	var messages []*MessageDeduplication
	for cursor.Next(ctx) {
		var msg MessageDeduplication
		if err := cursor.Decode(&msg); err != nil {
			return nil, fmt.Errorf("failed to decode message dedup record: %w", err)
		}
		messages = append(messages, &msg)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return messages, nil
}

func (r *messageDedupRepo) CleanupExpiredRecords(ctx context.Context) error {
	// TTL index should handle automatic cleanup, but we can also do manual cleanup
	cutoffTime := time.Now().Add(-25 * time.Hour) // Keep extra hour buffer
	filter := bson.M{
		"expires_at": bson.M{"$lte": cutoffTime},
	}

	result, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired dedup records: %w", err)
	}

	if result.DeletedCount > 0 {
		fmt.Printf("Cleaned up %d expired deduplication records\n", result.DeletedCount)
	}

	return nil
}
