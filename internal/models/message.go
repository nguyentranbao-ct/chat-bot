package models

import "time"

// KafkaMessage represents the top-level Kafka message structure
type KafkaMessage struct {
	Pattern string           `json:"pattern"`
	Data    KafkaMessageData `json:"data"`
}

// KafkaMessageData represents the actual message data from Kafka
type KafkaMessageData struct {
	ChannelID                 string                 `json:"channel_id" validate:"required"`
	SenderID                  string                 `json:"sender_id" validate:"required"`
	CreatedAt                 int64                  `json:"created_at" validate:"required"`
	Type                      string                 `json:"type"`
	Message                   string                 `json:"message" validate:"required"`
	FilterMsg                 *string                `json:"filter_msg"`
	Metadata                  map[string]interface{} `json:"metadata"`
	Attachment                interface{}            `json:"attachment"`
	ReceiverIDs               []string               `json:"receiver_ids"`
	ReceiverIDsForSpamMessage []string               `json:"receiver_ids_for_spam_message"`
	ClientGenID               string                 `json:"client_gen_id"`
	PreviousMessageCreatedAt  int64                  `json:"previous_message_created_at"`
	NumberID                  int                    `json:"number_id"`
	ClientMetadata            map[string]interface{} `json:"client_metadata"`
}

// IncomingMessage represents the simplified message structure for internal processing
type IncomingMessage struct {
	ChannelID string              `json:"channel_id" validate:"required"`
	CreatedAt int64               `json:"created_at" validate:"required"`
	SenderID  string              `json:"sender_id" validate:"required"`
	Content   string              `json:"content" validate:"required"`
	Metadata  IncomingMessageMeta `json:"metadata"`
	Vendor    VendorInfo          `json:"vendor"`
}

type IncomingMessageMeta struct {
	LLM LLMMetadata `json:"llm"`
}

type LLMMetadata struct {
	ChatMode string `json:"chat_mode" validate:"required"`
}

type VendorInfo struct {
	Name      string `json:"name"`
	ChannelID string `json:"channel_id"`
	MsgID     string `json:"msg_id"`
}

// ProcessIncomingMessageParams contains parameters for processing incoming messages with vendor detection
type ProcessIncomingMessageParams struct {
	ChannelID string                 `json:"channel_id" validate:"required"`
	SenderID  string                 `json:"sender_id" validate:"required"`
	Content   string                 `json:"content" validate:"required"`
	CreatedAt int64                  `json:"created_at" validate:"required"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Vendor    VendorInfo             `json:"vendor" validate:"required"`
}

type OutgoingMessage struct {
	ChannelID string     `json:"channel_id" validate:"required"`
	SenderID  string     `json:"sender_id" validate:"required"`
	Message   string     `json:"message" validate:"required"`
	Vendor    VendorInfo `json:"vendor"`
}

type ChannelInfo struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	ItemName     string        `json:"item_name"`
	ItemPrice    string        `json:"item_price"`
	Context      string        `json:"context"`
	Participants []Participant `json:"participants"`
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
