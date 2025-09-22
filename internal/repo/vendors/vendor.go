package vendors

import (
	"context"
	"time"

	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/tools/list_products"
)

// VendorType represents the type of vendor
type VendorType string

const (
	VendorTypeChotot   VendorType = "chotot"
	VendorTypeFacebook VendorType = "facebook"
	// Future vendors: telegram, whatsapp, etc.
)

// VendorMessage represents a normalized message from any vendor
type VendorMessage struct {
	ID                string                 `json:"id"`
	ChannelID         string                 `json:"channel_id"`
	SenderID          string                 `json:"sender_id"`
	Content           string                 `json:"content"`
	MessageType       string                 `json:"message_type"`
	CreatedAt         time.Time              `json:"created_at"`
	ExternalMessageID string                 `json:"external_message_id"`
	VendorMetadata    map[string]interface{} `json:"vendor_metadata"`
}

// VendorChannelInfo represents normalized channel info from any vendor
type VendorChannelInfo struct {
	ID           string                 `json:"id"` // external vendor channel ID
	Name         string                 `json:"name"`
	Context      string                 `json:"context"`
	Type         string                 `json:"type"` // "direct", "group", etc.
	Participants []models.Participant   `json:"participants"`
	VendorType   VendorType             `json:"vendor_type"`
	Metadata     map[string]interface{} `json:"metadata"` // ItemName, ItemPrice, etc.
}

// VendorCapabilities defines what operations a vendor supports
type VendorCapabilities struct {
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

// VendorUserInfo represents normalized user info from any vendor
type VendorUserInfo struct {
	ID           string                 `json:"id"` // external vendor user ID
	Name         string                 `json:"name"`
	Email        string                 `json:"email"`
	VendorType   VendorType             `json:"vendor_type"`
	Metadata     map[string]interface{} `json:"metadata"`
	IsActive     bool                   `json:"is_active"`
}

// Vendor interface defines the operations that all vendors must support
type Vendor interface {
	// Core identification
	GetVendorType() VendorType
	GetCapabilities() VendorCapabilities

	// Channel operations
	GetChannelInfo(ctx context.Context, channelID string) (*VendorChannelInfo, error)

	// User operations
	GetUserInfo(ctx context.Context, userID string) (*VendorUserInfo, error)

	// Message operations
	ListMessages(ctx context.Context, params MessageListParams) ([]VendorMessage, error)
	SendMessage(ctx context.Context, params SendMessageParams) error

	// Product operations
	GetUserProducts(ctx context.Context, params UserProductsParams) (*UserProductsResult, error)

	// Health check
	HealthCheck(ctx context.Context) error
}

// VendorError represents vendor-specific errors with additional context
type VendorError struct {
	VendorType VendorType `json:"vendor_type"`
	Operation  string     `json:"operation"`
	Message    string     `json:"message"`
	Code       string     `json:"code,omitempty"`
	Cause      error      `json:"-"`
}

func (e *VendorError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *VendorError) Unwrap() error {
	return e.Cause
}

// NewVendorError creates a new vendor error
func NewVendorError(vendorType VendorType, operation, message string, cause error) *VendorError {
	return &VendorError{
		VendorType: vendorType,
		Operation:  operation,
		Message:    message,
		Cause:      cause,
	}
}

// ErrVendorNotSupported is returned when a vendor doesn't support an operation
type ErrVendorNotSupported struct {
	VendorType VendorType
	Operation  string
}

func (e *ErrVendorNotSupported) Error() string {
	return string(e.VendorType) + " vendor does not support " + e.Operation
}

// NewErrVendorNotSupported creates a new vendor not supported error
func NewErrVendorNotSupported(vendorType VendorType, operation string) *ErrVendorNotSupported {
	return &ErrVendorNotSupported{
		VendorType: vendorType,
		Operation:  operation,
	}
}
