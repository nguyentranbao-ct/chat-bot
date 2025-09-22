package vendors

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/chatapi"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/chotot"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/tools/list_products"
)

// ChototVendor implements the Vendor interface for Chotot platform
// It combines both chat-api (Chotot's chat platform) and product services
type ChototVendor struct {
	chatClient    chatapi.Client
	productClient chotot.Client
}

// NewChototVendor creates a new Chotot vendor instance
func NewChototVendor(chatClient chatapi.Client, productClient chotot.Client) *ChototVendor {
	return &ChototVendor{
		chatClient:    chatClient,
		productClient: productClient,
	}
}

// GetVendorType returns the vendor type
func (v *ChototVendor) GetVendorType() VendorType {
	return VendorTypeChotot
}

// GetCapabilities returns the capabilities of the Chotot vendor
func (v *ChototVendor) GetCapabilities() VendorCapabilities {
	return VendorCapabilities{
		CanListMessages:    true,
		CanSendMessage:     true,
		CanGetChannelInfo:  true,
		CanGetUserProducts: true,
		SupportsRealtime:   true,
	}
}

// GetUserInfo retrieves user information from Chotot (returns empty info for now)
func (v *ChototVendor) GetUserInfo(ctx context.Context, userID string) (*VendorUserInfo, error) {
	if userID == "" {
		return nil, NewVendorError(VendorTypeChotot, "GetUserInfo", "user ID cannot be empty", nil)
	}

	// For now, return empty user info as requested
	// In the future, this could call Chotot user API to get user details
	vendorUserInfo := &VendorUserInfo{
		ID:         userID,
		Name:       "", // Empty for now
		Email:      "", // Empty for now
		VendorType: VendorTypeChotot,
		Metadata:   map[string]interface{}{},
		IsActive:   true, // Default to active
	}

	return vendorUserInfo, nil
}

// GetChannelInfo retrieves channel information from Chotot chat-api
func (v *ChototVendor) GetChannelInfo(ctx context.Context, channelID string) (*VendorChannelInfo, error) {
	if channelID == "" {
		return nil, NewVendorError(VendorTypeChotot, "GetChannelInfo", "channel ID cannot be empty", nil)
	}

	channelInfo, err := v.chatClient.GetChannelInfo(ctx, channelID)
	if err != nil {
		return nil, NewVendorError(VendorTypeChotot, "GetChannelInfo", "failed to get channel info", err)
	}

	// Convert to vendor channel info format
	vendorChannelInfo := &VendorChannelInfo{
		ID:           channelInfo.ID,
		Name:         channelInfo.Name,
		Context:      channelInfo.Context,
		Type:         "direct", // Default for Chotot channels
		Participants: channelInfo.Participants,
		VendorType:   VendorTypeChotot,
		Metadata: map[string]interface{}{
			"item_name":  channelInfo.ItemName,
			"item_price": channelInfo.ItemPrice,
		},
	}

	return vendorChannelInfo, nil
}

// ListMessages retrieves messages from Chotot chat-api
func (v *ChototVendor) ListMessages(ctx context.Context, params MessageListParams) ([]VendorMessage, error) {
	if params.ChannelID == "" {
		return nil, NewVendorError(VendorTypeChotot, "ListMessages", "channel ID cannot be empty", nil)
	}

	if params.UserID == "" {
		return nil, NewVendorError(VendorTypeChotot, "ListMessages", "user ID cannot be empty", nil)
	}

	// Prepare chat-api request
	req := chatapi.MessageHistoryRequest{
		UserID:    params.UserID,
		ChannelID: params.ChannelID,
		Limit:     params.Limit,
		BeforeTs:  params.BeforeTs,
	}

	// Set default limit if not specified
	if req.Limit <= 0 {
		req.Limit = 50
	}

	messageHistory, err := v.chatClient.GetMessageHistoryWithParams(ctx, req)
	if err != nil {
		return nil, NewVendorError(VendorTypeChotot, "ListMessages", "failed to get message history", err)
	}

	// Convert to vendor message format
	vendorMessages := make([]VendorMessage, 0, len(messageHistory.Messages))
	for _, msg := range messageHistory.Messages {
		vendorMsg := VendorMessage{
			ID:                msg.ID,
			ChannelID:         msg.ChannelID,
			SenderID:          msg.SenderID,
			Content:           msg.Message,
			MessageType:       "text", // Default for Chotot messages
			CreatedAt:         msg.CreatedAt,
			ExternalMessageID: msg.ID, // Use same ID as external ID
			VendorMetadata: map[string]interface{}{
				"source": "chotot_chat_api",
			},
		}
		vendorMessages = append(vendorMessages, vendorMsg)
	}

	return vendorMessages, nil
}

// SendMessage sends a message through Chotot chat-api
func (v *ChototVendor) SendMessage(ctx context.Context, params SendMessageParams) error {
	if params.ChannelID == "" {
		return NewVendorError(VendorTypeChotot, "SendMessage", "channel ID cannot be empty", nil)
	}

	if params.SenderID == "" {
		return NewVendorError(VendorTypeChotot, "SendMessage", "sender ID cannot be empty", nil)
	}

	if params.Content == "" {
		return NewVendorError(VendorTypeChotot, "SendMessage", "message content cannot be empty", nil)
	}

	// Create outgoing message
	outgoingMsg := &models.OutgoingMessage{
		ChannelID: params.ChannelID,
		SenderID:  params.SenderID,
		Message:   params.Content,
	}

	// Send through chat-api
	if err := v.chatClient.SendMessage(ctx, outgoingMsg); err != nil {
		return NewVendorError(VendorTypeChotot, "SendMessage", "failed to send message", err)
	}

	return nil
}

// GetUserProducts retrieves user products from Chotot product service
func (v *ChototVendor) GetUserProducts(ctx context.Context, params UserProductsParams) (*UserProductsResult, error) {
	if params.UserID == "" {
		return nil, NewVendorError(VendorTypeChotot, "GetUserProducts", "user ID cannot be empty", nil)
	}

	// Set default pagination if not specified
	limit := params.Limit
	if limit <= 0 {
		limit = 10
	}
	page := params.Page
	if page <= 0 {
		page = 1
	}

	// Get products from Chotot product service
	adsResponse, err := v.productClient.GetUserAds(ctx, params.UserID, limit, page)
	if err != nil {
		return nil, NewVendorError(VendorTypeChotot, "GetUserProducts", "failed to get user products", err)
	}

	// Convert to list_products.Product format
	productList := make([]list_products.Product, 0, len(adsResponse.Ads))
	for _, adsResp := range adsResponse.Ads {
		ad := adsResp.Info
		product := list_products.Product{
			ID:          fmt.Sprintf("%d", ad.ListID),
			Name:        ad.Subject,
			Category:    fmt.Sprintf("%d", ad.Category),
			Price:       parsePrice(ad.Price),
			PriceString: ad.PriceString,
			Images:      ad.Images,
			Source:      fmt.Sprintf("chotot://%d", ad.ListID),
		}
		productList = append(productList, product)
	}

	result := &UserProductsResult{
		Products: productList,
		Total:    adsResponse.Total,
	}

	return result, nil
}

// HealthCheck performs a health check on both chat and product services
func (v *ChototVendor) HealthCheck(ctx context.Context) error {
	// Create a timeout context for health check
	healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Test chat service by attempting to get a dummy channel (this will fail gracefully)
	_, err := v.chatClient.GetChannelInfo(healthCtx, "health-check")
	// We expect this to fail, but we want to make sure the service is reachable
	// A timeout or connection error would indicate the service is down
	if err != nil {
		// Check if it's a connection/timeout error vs expected "not found" error
		if isConnectionError(err) {
			return NewVendorError(VendorTypeChotot, "HealthCheck", "chat service unreachable", err)
		}
	}

	return nil
}

// Helper function to parse price from string to int
func parsePrice(priceStr string) int {
	if priceStr == "" {
		return 0
	}

	// Remove common currency symbols and formatting
	cleaned := priceStr
	// Remove Vietnamese dong symbol, commas, dots used as thousand separators
	replacements := []string{"₫", "VND", ".", ",", " "}
	for _, r := range replacements {
		cleaned = strings.ReplaceAll(cleaned, r, "")
	}

	// Try to parse as integer
	if price, err := strconv.Atoi(cleaned); err == nil {
		return price
	}

	return 0
}

// Helper function to check if an error is a connection-related error
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	// Check for common connection error patterns
	return contains(errStr, "connection") ||
		contains(errStr, "timeout") ||
		contains(errStr, "unreachable") ||
		contains(errStr, "dial") ||
		contains(errStr, "network")
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || indexOf(s, substr) >= 0))
}

// Helper function for string index
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
