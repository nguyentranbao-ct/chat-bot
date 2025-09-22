package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Channel struct {
	ID            primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	Vendor        ChannelVendor          `bson:"vendor" json:"vendor" validate:"required"`
	Name          string                 `bson:"name" json:"name" validate:"required"`
	Context       string                 `bson:"context" json:"context"`
	Type          string                 `bson:"type" json:"type"` // "direct", "group", etc.
	Metadata      map[string]any `bson:"metadata" json:"metadata"` // ItemName, ItemPrice, etc.
	CreatedAt     time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time              `bson:"updated_at" json:"updated_at"`
	LastMessageAt *time.Time             `bson:"last_message_at" json:"last_message_at"`
	IsArchived    bool                   `bson:"is_archived" json:"is_archived"`
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

// ChannelVendor represents the vendor information for a channel
type ChannelVendor struct {
	ChannelID string `bson:"channel_id" json:"channel_id" validate:"required"` // external vendor channel ID
	Name      string `bson:"name" json:"name" validate:"required"`             // "chotot", "facebook", etc.
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