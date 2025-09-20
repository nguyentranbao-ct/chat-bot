package models

import "time"

type IncomingMessage struct {
	ChannelID string               `json:"channel_id" validate:"required"`
	CreatedAt int64                `json:"created_at" validate:"required"`
	SenderID  string               `json:"sender_id" validate:"required"`
	Message   string               `json:"message" validate:"required"`
	Metadata  IncomingMessageMeta  `json:"metadata"`
}

type IncomingMessageMeta struct {
	LLM LLMMetadata `json:"llm"`
}

type LLMMetadata struct {
	ChatMode string `json:"chat_mode" validate:"required"`
}

type OutgoingMessage struct {
	ChannelID string `json:"channel_id" validate:"required"`
	SenderID  string `json:"sender_id" validate:"required"`
	Message   string `json:"message" validate:"required"`
}

type ChannelInfo struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	ItemName        string          `json:"item_name"`
	ItemPrice       float64         `json:"item_price"`
	RoleDescription string          `json:"role_description"`
	Participants    []Participant   `json:"participants"`
}

type Participant struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

type MessageHistory struct {
	Messages []HistoryMessage `json:"messages"`
	HasMore  bool             `json:"has_more"`
}

type HistoryMessage struct {
	ID        string    `json:"id"`
	ChannelID string    `json:"channel_id"`
	SenderID  string    `json:"sender_id"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}