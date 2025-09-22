package partners

import (
	"context"
	"time"

	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/tools/list_products"
)

// PartnerType represents the type of partner
type PartnerType string

const (
	PartnerTypeChotot   PartnerType = "chotot"
	PartnerTypeFacebook PartnerType = "facebook"
	// Future partners: telegram, whatsapp, etc.
)

// PartnerMessage represents a normalized message from any partner
type PartnerMessage struct {
	ID                string                 `json:"id"`
	ChannelID         string                 `json:"channel_id"`
	SenderID          string                 `json:"sender_id"`
	Content           string                 `json:"content"`
	MessageType       string                 `json:"message_type"`
	CreatedAt         time.Time              `json:"created_at"`
	ExternalMessageID string                 `json:"external_message_id"`
	PartnerMetadata   map[string]interface{} `json:"partner_metadata"`
}

// PartnerChannelInfo represents normalized channel info from any partner
type PartnerChannelInfo struct {
	ID           string                 `json:"id"` // external partner channel ID
	Name         string                 `json:"name"`
	Context      string                 `json:"context"`
	Type         string                 `json:"type"` // "direct", "group", etc.
	Participants []models.Participant   `json:"participants"`
	PartnerType  PartnerType            `json:"partner_type"`
	Metadata     map[string]interface{} `json:"metadata"` // ItemName, ItemPrice, etc.
}

// PartnerCapabilities defines what operations a partner supports
type PartnerCapabilities struct {
	CanListMessages    bool `json:"can_list_messages"`
	CanSendMessage     bool `json:"can_send_message"`
	CanGetChannelInfo  bool `json:"can_get_channel_info"`
	CanGetUserProducts bool `json:"can_get_user_products"`
	SupportsRealtime   bool `json:"supports_realtime"`
}

// MessageListParams represents parameters for listing messages
type MessageListParams struct {
	ChannelID string `json:"channel_id"`
	UserID    string `json:"user_id,omitempty"`
	Limit     int    `json:"limit"`
	BeforeTs  *int64 `json:"before_ts,omitempty"`
	AfterTs   *int64 `json:"after_ts,omitempty"`
}

// SendMessageParams represents parameters for sending a message
type SendMessageParams struct {
	ChannelID   string                 `json:"channel_id"`
	SenderID    string                 `json:"sender_id"`
	Content     string                 `json:"content"`
	MessageType string                 `json:"message_type"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UserProductsParams represents parameters for fetching user products
type UserProductsParams struct {
	UserID string `json:"user_id"`
	Limit  int    `json:"limit"`
	Page   int    `json:"page"`
}

// UserProductsResult represents the result of fetching user products
type UserProductsResult struct {
	Products []list_products.Product `json:"products"`
	Total    int                     `json:"total"`
}

// PartnerUserInfo represents normalized user info from any partner
type PartnerUserInfo struct {
	ID          string                 `json:"id"` // external partner user ID
	Name        string                 `json:"name"`
	Email       string                 `json:"email"`
	PartnerType PartnerType            `json:"partner_type"`
	Metadata    map[string]interface{} `json:"metadata"`
	IsActive    bool                   `json:"is_active"`
}

// Partner interface defines the operations that all partners must support
type Partner interface {
	// Core identification
	GetPartnerType() PartnerType
	GetCapabilities() PartnerCapabilities

	// Channel operations
	GetChannelInfo(ctx context.Context, channelID string) (*PartnerChannelInfo, error)

	// User operations
	GetUserInfo(ctx context.Context, userID string) (*PartnerUserInfo, error)

	// Message operations
	SendMessage(ctx context.Context, params SendMessageParams) error

	// Product operations
	GetUserProducts(ctx context.Context, params UserProductsParams) (*UserProductsResult, error)

	// Health check
	HealthCheck(ctx context.Context) error
}

// PartnerError represents partner-specific errors with additional context
type PartnerError struct {
	PartnerType PartnerType `json:"partner_type"`
	Operation   string      `json:"operation"`
	Message     string      `json:"message"`
	Code        string      `json:"code,omitempty"`
	Cause       error       `json:"-"`
}

func (e *PartnerError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *PartnerError) Unwrap() error {
	return e.Cause
}

// NewPartnerError creates a new partner error
func NewPartnerError(partnerType PartnerType, operation, message string, cause error) *PartnerError {
	return &PartnerError{
		PartnerType: partnerType,
		Operation:   operation,
		Message:     message,
		Cause:       cause,
	}
}

// ErrPartnerNotSupported is returned when a partner doesn't support an operation
type ErrPartnerNotSupported struct {
	PartnerType PartnerType
	Operation   string
}

func (e *ErrPartnerNotSupported) Error() string {
	return string(e.PartnerType) + " partner does not support " + e.Operation
}

// NewErrPartnerNotSupported creates a new partner not supported error
func NewErrPartnerNotSupported(partnerType PartnerType, operation string) *ErrPartnerNotSupported {
	return &ErrPartnerNotSupported{
		PartnerType: partnerType,
		Operation:   operation,
	}
}
