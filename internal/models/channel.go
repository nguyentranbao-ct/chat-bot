package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Channel struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ExternalChannelID string             `bson:"external_channel_id" json:"external_channel_id" validate:"required"` // chat-api channel ID
	Name              string             `bson:"name" json:"name" validate:"required"`
	ItemName          string             `bson:"item_name" json:"item_name"`
	ItemPrice         string             `bson:"item_price" json:"item_price"`
	Context           string             `bson:"context" json:"context"`
	Type              string             `bson:"type" json:"type"` // "direct", "group", etc.
	CreatedAt         time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt         time.Time          `bson:"updated_at" json:"updated_at"`
	LastMessageAt     *time.Time         `bson:"last_message_at" json:"last_message_at"`
	IsArchived        bool               `bson:"is_archived" json:"is_archived"`
}

type ChannelMember struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ChannelID primitive.ObjectID `bson:"channel_id" json:"channel_id" validate:"required"`
	UserID    string             `bson:"user_id" json:"user_id" validate:"required"` // external user ID
	Role      string             `bson:"role" json:"role"`                           // "buyer", "seller", "agent", etc.
	JoinedAt  time.Time          `bson:"joined_at" json:"joined_at"`
	LeftAt    *time.Time         `bson:"left_at" json:"left_at"`
	IsActive  bool               `bson:"is_active" json:"is_active"`
}

// UnreadCount tracks unread messages per user per channel
type UnreadCount struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ChannelID         primitive.ObjectID `bson:"channel_id" json:"channel_id" validate:"required"`
	UserID            string             `bson:"user_id" json:"user_id" validate:"required"`
	Count             int                `bson:"count" json:"count"`
	LastReadMessageID primitive.ObjectID `bson:"last_read_message_id" json:"last_read_message_id"`
	UpdatedAt         time.Time          `bson:"updated_at" json:"updated_at"`
}