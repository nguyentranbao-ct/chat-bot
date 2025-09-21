package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/nguyentranbao-ct/chat-bot/internal/models"
)

type ChatMessageRepository interface {
	Create(ctx context.Context, message *models.ChatMessage) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.ChatMessage, error)
	GetChannelMessages(ctx context.Context, channelID primitive.ObjectID, limit int, before *primitive.ObjectID) ([]*models.ChatMessage, error)
	GetByExternalMessageID(ctx context.Context, externalMessageID string) (*models.ChatMessage, error)
	GetLatestMessage(ctx context.Context, channelID primitive.ObjectID) (*models.ChatMessage, error)
	GetUnreadMessages(ctx context.Context, channelID primitive.ObjectID, userID string, lastReadMessageID *primitive.ObjectID) ([]*models.ChatMessage, error)
	UpdateMessage(ctx context.Context, messageID primitive.ObjectID, content string, blocks []models.MessageBlock) error
	SoftDeleteMessage(ctx context.Context, messageID primitive.ObjectID) error
	UpdateDeliveryStatus(ctx context.Context, messageID primitive.ObjectID, status string) error
	GetMessagesByTimeRange(ctx context.Context, channelID primitive.ObjectID, startTime, endTime time.Time) ([]*models.ChatMessage, error)
}

type chatMessageRepo struct {
	collection *mongo.Collection
}

func NewChatMessageRepository(db *DB) ChatMessageRepository {
	return &chatMessageRepo{
		collection: db.Database.Collection("chat_messages"),
	}
}

func (r *chatMessageRepo) Create(ctx context.Context, message *models.ChatMessage) error {
	message.ID = primitive.NewObjectID()
	message.CreatedAt = time.Now()
	message.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}
	return nil
}

func (r *chatMessageRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*models.ChatMessage, error) {
	var message models.ChatMessage
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&message)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("message not found")
		}
		return nil, fmt.Errorf("failed to get message: %w", err)
	}
	return &message, nil
}

func (r *chatMessageRepo) GetChannelMessages(ctx context.Context, channelID primitive.ObjectID, limit int, before *primitive.ObjectID) ([]*models.ChatMessage, error) {
	filter := bson.M{
		"channel_id": channelID,
		"is_deleted": false,
	}

	if before != nil {
		filter["_id"] = bson.M{"$lt": *before}
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel messages: %w", err)
	}
	defer cursor.Close(ctx)

	var messages []*models.ChatMessage
	for cursor.Next(ctx) {
		var message models.ChatMessage
		if err := cursor.Decode(&message); err != nil {
			return nil, fmt.Errorf("failed to decode message: %w", err)
		}
		messages = append(messages, &message)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return messages, nil
}

func (r *chatMessageRepo) GetByExternalMessageID(ctx context.Context, externalMessageID string) (*models.ChatMessage, error) {
	var message models.ChatMessage
	err := r.collection.FindOne(ctx, bson.M{"external_message_id": externalMessageID}).Decode(&message)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("message not found")
		}
		return nil, fmt.Errorf("failed to get message: %w", err)
	}
	return &message, nil
}

func (r *chatMessageRepo) GetLatestMessage(ctx context.Context, channelID primitive.ObjectID) (*models.ChatMessage, error) {
	filter := bson.M{
		"channel_id": channelID,
		"is_deleted": false,
	}

	opts := options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}})
	var message models.ChatMessage
	err := r.collection.FindOne(ctx, filter, opts).Decode(&message)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest message: %w", err)
	}
	return &message, nil
}

func (r *chatMessageRepo) GetUnreadMessages(ctx context.Context, channelID primitive.ObjectID, userID string, lastReadMessageID *primitive.ObjectID) ([]*models.ChatMessage, error) {
	filter := bson.M{
		"channel_id": channelID,
		"is_deleted": false,
		"sender_id":  bson.M{"$ne": userID}, // exclude user's own messages
	}

	if lastReadMessageID != nil {
		filter["_id"] = bson.M{"$gt": *lastReadMessageID}
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get unread messages: %w", err)
	}
	defer cursor.Close(ctx)

	var messages []*models.ChatMessage
	for cursor.Next(ctx) {
		var message models.ChatMessage
		if err := cursor.Decode(&message); err != nil {
			return nil, fmt.Errorf("failed to decode message: %w", err)
		}
		messages = append(messages, &message)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return messages, nil
}

func (r *chatMessageRepo) UpdateMessage(ctx context.Context, messageID primitive.ObjectID, content string, blocks []models.MessageBlock) error {
	filter := bson.M{"_id": messageID}
	update := bson.M{
		"$set": bson.M{
			"content":    content,
			"blocks":     blocks,
			"is_edited":  true,
			"edited_at":  time.Now(),
			"updated_at": time.Now(),
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *chatMessageRepo) SoftDeleteMessage(ctx context.Context, messageID primitive.ObjectID) error {
	filter := bson.M{"_id": messageID}
	update := bson.M{
		"$set": bson.M{
			"is_deleted": true,
			"updated_at": time.Now(),
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *chatMessageRepo) UpdateDeliveryStatus(ctx context.Context, messageID primitive.ObjectID, status string) error {
	filter := bson.M{"_id": messageID}
	update := bson.M{
		"$set": bson.M{
			"delivery_status": status,
			"updated_at":      time.Now(),
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *chatMessageRepo) GetMessagesByTimeRange(ctx context.Context, channelID primitive.ObjectID, startTime, endTime time.Time) ([]*models.ChatMessage, error) {
	filter := bson.M{
		"channel_id": channelID,
		"created_at": bson.M{
			"$gte": startTime,
			"$lte": endTime,
		},
		"is_deleted": false,
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages by time range: %w", err)
	}
	defer cursor.Close(ctx)

	var messages []*models.ChatMessage
	for cursor.Next(ctx) {
		var message models.ChatMessage
		if err := cursor.Decode(&message); err != nil {
			return nil, fmt.Errorf("failed to decode message: %w", err)
		}
		messages = append(messages, &message)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return messages, nil
}

type MessageEventRepository interface {
	Create(ctx context.Context, event *models.MessageEvent) error
	CreateEvent(ctx context.Context, channelID primitive.ObjectID, eventType string, messageID *primitive.ObjectID, userID string, eventData map[string]interface{}) error
	GetChannelEvents(ctx context.Context, channelID primitive.ObjectID, sinceTime time.Time) ([]*models.MessageEvent, error)
	CleanupExpiredEvents(ctx context.Context) error
}

type messageEventRepo struct {
	collection *mongo.Collection
}

func NewMessageEventRepository(db *DB) MessageEventRepository {
	return &messageEventRepo{
		collection: db.Database.Collection("message_events"),
	}
}

func (r *messageEventRepo) Create(ctx context.Context, event *models.MessageEvent) error {
	event.ID = primitive.NewObjectID()
	event.CreatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, event)
	if err != nil {
		return fmt.Errorf("failed to create message event: %w", err)
	}
	return nil
}

func (r *messageEventRepo) CreateEvent(ctx context.Context, channelID primitive.ObjectID, eventType string, messageID *primitive.ObjectID, userID string, eventData map[string]interface{}) error {
	event := &models.MessageEvent{
		ChannelID: channelID,
		EventType: eventType,
		MessageID: messageID,
		UserID:    userID,
		EventData: eventData,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour), // Events expire after 24 hours
	}

	return r.Create(ctx, event)
}

func (r *messageEventRepo) GetChannelEvents(ctx context.Context, channelID primitive.ObjectID, sinceTime time.Time) ([]*models.MessageEvent, error) {
	filter := bson.M{
		"channel_id": channelID,
		"created_at": bson.M{"$gt": sinceTime},
		"expires_at": bson.M{"$gt": time.Now()},
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel events: %w", err)
	}
	defer cursor.Close(ctx)

	var events []*models.MessageEvent
	for cursor.Next(ctx) {
		var event models.MessageEvent
		if err := cursor.Decode(&event); err != nil {
			return nil, fmt.Errorf("failed to decode event: %w", err)
		}
		events = append(events, &event)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return events, nil
}

func (r *messageEventRepo) CleanupExpiredEvents(ctx context.Context) error {
	filter := bson.M{
		"expires_at": bson.M{"$lte": time.Now()},
	}

	_, err := r.collection.DeleteMany(ctx, filter)
	return err
}

type TypingIndicatorRepository interface {
	SetTyping(ctx context.Context, channelID primitive.ObjectID, userID string, isTyping bool) error
	GetTypingUsers(ctx context.Context, channelID primitive.ObjectID) ([]*models.TypingIndicator, error)
	CleanupExpiredTyping(ctx context.Context) error
}

type typingIndicatorRepo struct {
	collection *mongo.Collection
}

func NewTypingIndicatorRepository(db *DB) TypingIndicatorRepository {
	return &typingIndicatorRepo{
		collection: db.Database.Collection("typing_indicators"),
	}
}

func (r *typingIndicatorRepo) SetTyping(ctx context.Context, channelID primitive.ObjectID, userID string, isTyping bool) error {
	filter := bson.M{
		"channel_id": channelID,
		"user_id":    userID,
	}

	update := bson.M{
		"$set": bson.M{
			"is_typing":  isTyping,
			"updated_at": time.Now(),
			"expires_at": time.Now().Add(30 * time.Second), // Typing expires after 30 seconds
		},
		"$setOnInsert": bson.M{
			"channel_id": channelID,
			"user_id":    userID,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (r *typingIndicatorRepo) GetTypingUsers(ctx context.Context, channelID primitive.ObjectID) ([]*models.TypingIndicator, error) {
	filter := bson.M{
		"channel_id": channelID,
		"is_typing":  true,
		"expires_at": bson.M{"$gt": time.Now()},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get typing users: %w", err)
	}
	defer cursor.Close(ctx)

	var indicators []*models.TypingIndicator
	for cursor.Next(ctx) {
		var indicator models.TypingIndicator
		if err := cursor.Decode(&indicator); err != nil {
			return nil, fmt.Errorf("failed to decode typing indicator: %w", err)
		}
		indicators = append(indicators, &indicator)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return indicators, nil
}

func (r *typingIndicatorRepo) CleanupExpiredTyping(ctx context.Context) error {
	filter := bson.M{
		"expires_at": bson.M{"$lte": time.Now()},
	}

	update := bson.M{
		"$set": bson.M{"is_typing": false},
	}

	_, err := r.collection.UpdateMany(ctx, filter, update)
	return err
}