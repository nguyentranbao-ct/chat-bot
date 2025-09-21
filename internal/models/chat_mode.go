package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChatMode struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id" yaml:"-"`
	Name              string             `bson:"name" json:"name" yaml:"name"`
	PromptTemplate    string             `bson:"prompt_template" json:"prompt_template" yaml:"prompt_template"`
	Condition         string             `bson:"condition" json:"condition" yaml:"condition"`
	Model             string             `bson:"model" json:"model" yaml:"model"`
	Tools             []string           `bson:"tools" json:"tools" yaml:"tools"`
	MaxIterations     int                `bson:"max_iterations" json:"max_iterations" yaml:"max_iterations"`
	MaxPromptTokens   int                `bson:"max_prompt_tokens" json:"max_prompt_tokens" yaml:"max_prompt_tokens"`
	MaxResponseTokens int                `bson:"max_response_tokens" json:"max_response_tokens" yaml:"max_response_tokens"`
	CreatedAt         time.Time          `bson:"created_at" json:"created_at" yaml:"-"`
	UpdatedAt         time.Time          `bson:"updated_at" json:"updated_at" yaml:"-"`
}

type ChatSession struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ChannelID string             `bson:"channel_id" json:"channel_id"`
	UserID    string             `bson:"user_id" json:"user_id"`
	ChatMode  string             `bson:"chat_mode" json:"chat_mode"`
	Status    SessionStatus      `bson:"status" json:"status"`
	StartedAt time.Time          `bson:"started_at" json:"started_at"`
	EndedAt   *time.Time         `bson:"ended_at,omitempty" json:"ended_at,omitempty"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

type ChatActivity struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SessionID  primitive.ObjectID `bson:"session_id" json:"session_id"`
	ChannelID  string             `bson:"channel_id" json:"channel_id"`
	MessageID  string             `bson:"message_id,omitempty" json:"message_id,omitempty"`
	Action     ActivityAction     `bson:"action" json:"action"`
	Data       interface{}        `bson:"data,omitempty" json:"data,omitempty"`
	ExecutedAt time.Time          `bson:"executed_at" json:"executed_at"`
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
}

type PurchaseIntent struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SessionID  primitive.ObjectID `bson:"session_id" json:"session_id"`
	ChannelID  string             `bson:"channel_id" json:"channel_id"`
	UserID     string             `bson:"user_id" json:"user_id"`
	ItemName   string             `bson:"item_name" json:"item_name"`
	ItemPrice  string             `bson:"item_price" json:"item_price"`
	Intent     string             `bson:"intent" json:"intent"`
	Percentage int                `bson:"percentage" json:"percentage"`
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
}

type SessionStatus string

const (
	SessionStatusActive    SessionStatus = "active"
	SessionStatusEnded     SessionStatus = "ended"
	SessionStatusAbandoned SessionStatus = "abandoned"
)

type ActivityAction string

const (
	ActivityPurchaseIntent ActivityAction = "purchase_intent"
	ActivityReplyMessage   ActivityAction = "reply_message"
	ActivityFetchMessages  ActivityAction = "fetch_messages"
	ActivityEndSession     ActivityAction = "end_session"
)
