package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ChatMessage represents a message in our chat system
type ChatMessage struct {
	ID                   primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ChannelID            primitive.ObjectID `bson:"channel_id" json:"channel_id" validate:"required"`
	ExternalChannelID    string             `bson:"external_channel_id" json:"external_channel_id"` // chat-api channel ID
	SenderID             string             `bson:"sender_id" json:"sender_id" validate:"required"`
	MessageType          string             `bson:"message_type" json:"message_type"` // "text", "blocks", "system", etc.
	Content              string             `bson:"content" json:"content"`           // plain text content
	Blocks               []MessageBlock     `bson:"blocks" json:"blocks"`             // Slack-style blocks
	ThreadID             *primitive.ObjectID `bson:"thread_id" json:"thread_id"`      // for threaded conversations
	ReplyToMessageID     *primitive.ObjectID `bson:"reply_to_message_id" json:"reply_to_message_id"`
	CreatedAt            time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt            time.Time          `bson:"updated_at" json:"updated_at"`
	IsDeleted            bool               `bson:"is_deleted" json:"is_deleted"`
	IsEdited             bool               `bson:"is_edited" json:"is_edited"`
	EditedAt             *time.Time         `bson:"edited_at" json:"edited_at"`
	Metadata             MessageMetadata    `bson:"metadata" json:"metadata"`
	ExternalMessageID    string             `bson:"external_message_id" json:"external_message_id"` // chat-api message ID
	DeliveryStatus       string             `bson:"delivery_status" json:"delivery_status"`         // "sent", "delivered", "read", "failed"
}

// MessageBlock represents a Slack-style message block
type MessageBlock struct {
	Type     string                 `bson:"type" json:"type"` // "section", "divider", "image", "actions", etc.
	Text     *TextBlock             `bson:"text,omitempty" json:"text,omitempty"`
	Elements []MessageElement       `bson:"elements,omitempty" json:"elements,omitempty"`
	Fields   []TextBlock            `bson:"fields,omitempty" json:"fields,omitempty"`
	ImageURL string                 `bson:"image_url,omitempty" json:"image_url,omitempty"`
	AltText  string                 `bson:"alt_text,omitempty" json:"alt_text,omitempty"`
	Metadata map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
}

// TextBlock represents text content with formatting
type TextBlock struct {
	Type string `bson:"type" json:"type"` // "plain_text", "mrkdwn"
	Text string `bson:"text" json:"text"`
}

// MessageElement represents interactive elements in blocks
type MessageElement struct {
	Type     string                 `bson:"type" json:"type"` // "button", "select", "image", etc.
	Text     *TextBlock             `bson:"text,omitempty" json:"text,omitempty"`
	Value    string                 `bson:"value,omitempty" json:"value,omitempty"`
	URL      string                 `bson:"url,omitempty" json:"url,omitempty"`
	ActionID string                 `bson:"action_id,omitempty" json:"action_id,omitempty"`
	Style    string                 `bson:"style,omitempty" json:"style,omitempty"` // "primary", "danger"
	Metadata map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
}

// MessageMetadata stores additional message information
type MessageMetadata struct {
	Source            string                 `bson:"source" json:"source"`                       // "kafka", "api", "agent"
	IsFromBot         bool                   `bson:"is_from_bot" json:"is_from_bot"`
	BotID             string                 `bson:"bot_id,omitempty" json:"bot_id,omitempty"`
	Tags              []string               `bson:"tags" json:"tags"`
	Priority          string                 `bson:"priority" json:"priority"` // "low", "normal", "high", "urgent"
	CustomData        map[string]interface{} `bson:"custom_data" json:"custom_data"`
	OriginalTimestamp int64                  `bson:"original_timestamp" json:"original_timestamp"` // from external system
}

// MessageEvent represents events for offline sync
type MessageEvent struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ChannelID    primitive.ObjectID `bson:"channel_id" json:"channel_id" validate:"required"`
	EventType    string             `bson:"event_type" json:"event_type"` // "message_sent", "message_updated", "message_deleted", "user_typing"
	MessageID    *primitive.ObjectID `bson:"message_id" json:"message_id"`
	UserID       string             `bson:"user_id" json:"user_id"`
	EventData    map[string]interface{} `bson:"event_data" json:"event_data"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	ExpiresAt    time.Time          `bson:"expires_at" json:"expires_at"` // TTL for temporary events
}

// TypingIndicator tracks who is currently typing
type TypingIndicator struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ChannelID primitive.ObjectID `bson:"channel_id" json:"channel_id" validate:"required"`
	UserID    string             `bson:"user_id" json:"user_id" validate:"required"`
	IsTyping  bool               `bson:"is_typing" json:"is_typing"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
	ExpiresAt time.Time          `bson:"expires_at" json:"expires_at"` // TTL
}