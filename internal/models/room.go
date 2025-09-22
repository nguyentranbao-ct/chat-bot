package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Room struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Source        RoomPartner        `bson:"partner" json:"partner" validate:"required"`
	Name          string             `bson:"name" json:"name" validate:"required"`
	Context       string             `bson:"context" json:"context"`
	Metadata      map[string]any     `bson:"metadata" json:"metadata"` // ItemName, ItemPrice, etc.
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updated_at"`
	LastMessageAt *time.Time         `bson:"last_message_at" json:"last_message_at"`
}

type RoomMember struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	RoomID   primitive.ObjectID `bson:"room_id" json:"room_id" validate:"required"`
	UserID   primitive.ObjectID `bson:"user_id" json:"user_id" validate:"required"` // external user ID
	Role     string             `bson:"role" json:"role"`                           // "buyer", "seller", "agent", etc.
	JoinedAt time.Time          `bson:"joined_at" json:"joined_at"`
}

// RoomPartner represents the partner information for a room
type RoomPartner struct {
	RoomID string `bson:"room_id" json:"room_id" validate:"required"` // external partner room ID
	Name   string `bson:"name" json:"name" validate:"required"`       // "chotot", "facebook", etc.
}

// UnreadCount tracks unread messages per user per room
type UnreadCount struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	RoomID            primitive.ObjectID `bson:"room_id" json:"room_id" validate:"required"`
	UserID            primitive.ObjectID `bson:"user_id" json:"user_id" validate:"required"`
	Count             int                `bson:"count" json:"count"`
	LastReadMessageID primitive.ObjectID `bson:"last_read_message_id" json:"last_read_message_id"`
	UpdatedAt         time.Time          `bson:"updated_at" json:"updated_at"`
}
