package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RoomMember struct {
	// Member identity
	ID     primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID primitive.ObjectID `bson:"user_id" json:"user_id" validate:"required"`
	Role   string             `bson:"role" json:"role"` // "merchant", "buyer", "seller", "agent", etc.

	// Room information (denormalized for performance)
	Source      RoomPartner        `bson:"source" json:"source" validate:"required"`
	RoomID      primitive.ObjectID `bson:"room_id" json:"room_id" validate:"required"`
	RoomName    string             `bson:"room_name" json:"room_name" validate:"required"`
	RoomContext string             `bson:"room_context,omitempty" json:"room_context,omitempty"`
	Metadata    map[string]any     `bson:"metadata,omitempty" json:"metadata,omitempty"`

	// Member-specific data
	LastReadAt         time.Time `bson:"last_read_at,omitempty" json:"last_read_at,omitempty"`
	LastMessageAt      time.Time `bson:"last_message_at,omitempty" json:"last_message_at,omitempty"`
	LastMessageContent string    `bson:"last_message_content,omitempty" json:"last_message_content,omitempty"`
	UnreadCount        int       `bson:"unread_count,omitempty" json:"unread_count,omitempty"`

	// Timestamps
	JoinedAt  time.Time `bson:"joined_at" json:"joined_at"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// RoomPartner represents the partner information for a room
type RoomPartner struct {
	RoomID string `bson:"room_id" json:"room_id" validate:"required"` // external partner room ID
	Name   string `bson:"name" json:"name" validate:"required"`       // "chotot", "facebook", etc.
}

// Room represents the client-side view of a room (converted from RoomMember)
type Room struct {
	ID                 string     `json:"id"`
	Name               string     `json:"name"`
	ItemName           string     `json:"item_name,omitempty"`
	ItemPrice          string     `json:"item_price,omitempty"`
	Context            string     `json:"context,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	LastMessageAt      *time.Time `json:"last_message_at,omitempty"`
	LastMessageContent string     `json:"last_message_content"`
	IsArchived         bool       `json:"is_archived"`
	UnreadCount        int        `json:"unread_count"`
}

// ToRoom converts a RoomMember to a client-facing Room
func (rm *RoomMember) ToRoom() *Room {
	room := &Room{
		ID:                 rm.RoomID.Hex(),
		Name:               rm.RoomName,
		Context:            rm.RoomContext,
		CreatedAt:          rm.CreatedAt,
		UpdatedAt:          rm.UpdatedAt,
		IsArchived:         false, // default value
		UnreadCount:        rm.UnreadCount,
		LastMessageContent: rm.LastMessageContent,
	}

	// Set last message time if available
	if !rm.LastMessageAt.IsZero() {
		room.LastMessageAt = &rm.LastMessageAt
	}

	// Extract item info from metadata
	if rm.Metadata != nil {
		if itemName, ok := rm.Metadata["item_name"].(string); ok {
			room.ItemName = itemName
		}
		if itemPrice, ok := rm.Metadata["item_price"].(string); ok {
			room.ItemPrice = itemPrice
		}
	}

	return room
}

// RoomMemberInfo represents basic room member information for clients
type RoomMemberInfo struct {
	ID       string    `json:"id"`
	UserID   string    `json:"user_id"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

// ToRoomMemberInfo converts a RoomMember to basic member info for clients
func (rm *RoomMember) ToRoomMemberInfo() *RoomMemberInfo {
	return &RoomMemberInfo{
		ID:       rm.ID.Hex(),
		UserID:   rm.UserID.Hex(),
		Role:     rm.Role,
		JoinedAt: rm.JoinedAt,
	}
}
