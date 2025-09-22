package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ChatMessage represents a message in our chat system
type ChatMessage struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ChannelID primitive.ObjectID `bson:"channel_id" json:"channel_id" validate:"required"`
	SenderID  primitive.ObjectID `bson:"sender_id" json:"sender_id" validate:"required"`
	Content   string             `bson:"content" json:"content"` // plain text content
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
	EditedAt  *time.Time         `bson:"edited_at" json:"edited_at"`
	Metadata  MessageMetadata    `bson:"metadata" json:"metadata"`
	Source    *Source            `bson:"source,omitempty" json:"source,omitempty"`
}

// MessageMetadata stores additional message information
type MessageMetadata struct {
	CustomData        map[string]interface{} `bson:"custom_data" json:"custom_data"`
	OriginalTimestamp int64                  `bson:"original_timestamp" json:"original_timestamp"` // from external system
}

// MessageEvent represents events for offline sync
type MessageEvent struct {
	ID        primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	ChannelID primitive.ObjectID     `bson:"channel_id" json:"channel_id" validate:"required"`
	EventType string                 `bson:"event_type" json:"event_type"` // "message_sent", "message_updated", "message_deleted", "user_typing"
	MessageID *primitive.ObjectID    `bson:"message_id" json:"message_id"`
	UserID    primitive.ObjectID     `bson:"user_id" json:"user_id"`
	EventData map[string]interface{} `bson:"event_data" json:"event_data"`
	CreatedAt time.Time              `bson:"created_at" json:"created_at"`
	ExpiresAt time.Time              `bson:"expires_at" json:"expires_at"` // TTL for temporary events
}
