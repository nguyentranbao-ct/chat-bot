package partners

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

// ChototPartner implements the Partner interface for Chotot platform
// It combines both chat-api (Chotot's chat platform) and product services
type ChototPartner struct {
	chatClient    chatapi.Client
	productClient chotot.Client
}

// NewChototPartner creates a new Chotot partner instance
func NewChototPartner(chatClient chatapi.Client, productClient chotot.Client) *ChototPartner {
	return &ChototPartner{
		chatClient:    chatClient,
		productClient: productClient,
	}
}

// GetPartnerType returns the partner type
func (v *ChototPartner) GetPartnerType() PartnerType {
	return PartnerTypeChotot
}

// GetCapabilities returns the capabilities of the Chotot partner
func (v *ChototPartner) GetCapabilities() PartnerCapabilities {
	return PartnerCapabilities{
		CanListMessages:    true,
		CanSendMessage:     true,
		CanGetChannelInfo:  true,
		CanGetUserProducts: true,
		SupportsRealtime:   true,
	}
}

// GetUserInfo retrieves user information from Chotot (returns empty info for now)
func (v *ChototPartner) GetUserInfo(ctx context.Context, userID string) (*PartnerUserInfo, error) {
	if userID == "" {
		return nil, NewPartnerError(PartnerTypeChotot, "GetUserInfo", "user ID cannot be empty", nil)
	}

	// For now, return empty user info as requested
	// In the future, this could call Chotot user API to get user details
	partnerUserInfo := &PartnerUserInfo{
		ID:          userID,
		Name:        "", // Empty for now
		Email:       "", // Empty for now
		PartnerType: PartnerTypeChotot,
		Metadata:    map[string]interface{}{},
		IsActive:    true, // Default to active
	}

	return partnerUserInfo, nil
}

// GetChannelInfo retrieves channel information from Chotot chat-api
func (v *ChototPartner) GetChannelInfo(ctx context.Context, channelID string) (*PartnerChannelInfo, error) {
	if channelID == "" {
		return nil, NewPartnerError(PartnerTypeChotot, "GetChannelInfo", "channel ID cannot be empty", nil)
	}

	channelInfo, err := v.chatClient.GetChannelInfo(ctx, channelID)
	if err != nil {
		return nil, NewPartnerError(PartnerTypeChotot, "GetChannelInfo", "failed to get channel info", err)
	}

	// Convert to partner channel info format
	partnerChannelInfo := &PartnerChannelInfo{
		ID:           channelInfo.ID,
		Name:         channelInfo.Name,
		Context:      channelInfo.Context,
		Type:         "direct", // Default for Chotot channels
		Participants: channelInfo.Participants,
		PartnerType:  PartnerTypeChotot,
		Metadata: map[string]interface{}{
			"item_name":  channelInfo.ItemName,
			"item_price": channelInfo.ItemPrice,
		},
	}

	return partnerChannelInfo, nil
}

// SendMessage sends a message through Chotot chat-api
func (v *ChototPartner) SendMessage(ctx context.Context, params SendMessageParams) error {
	if params.ChannelID == "" {
		return NewPartnerError(PartnerTypeChotot, "SendMessage", "channel ID cannot be empty", nil)
	}

	if params.SenderID == "" {
		return NewPartnerError(PartnerTypeChotot, "SendMessage", "sender ID cannot be empty", nil)
	}

	if params.Content == "" {
		return NewPartnerError(PartnerTypeChotot, "SendMessage", "message content cannot be empty", nil)
	}

	// Create outgoing message
	outgoingMsg := &models.OutgoingMessage{
		ChannelID: params.ChannelID,
		SenderID:  params.SenderID,
		Message:   params.Content,
	}

	// Send through chat-api
	if err := v.chatClient.SendMessage(ctx, outgoingMsg); err != nil {
		return NewPartnerError(PartnerTypeChotot, "SendMessage", "failed to send message", err)
	}

	return nil
}

// GetUserProducts retrieves user products from Chotot product service
func (v *ChototPartner) GetUserProducts(ctx context.Context, params UserProductsParams) (*UserProductsResult, error) {
	if params.UserID == "" {
		return nil, NewPartnerError(PartnerTypeChotot, "GetUserProducts", "user ID cannot be empty", nil)
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
		return nil, NewPartnerError(PartnerTypeChotot, "GetUserProducts", "failed to get user products", err)
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
func (v *ChototPartner) HealthCheck(ctx context.Context) error {
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
			return NewPartnerError(PartnerTypeChotot, "HealthCheck", "chat service unreachable", err)
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
