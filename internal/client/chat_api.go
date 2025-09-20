package client

import (
	"context"
	"fmt"
	"time"

	"github.com/carousell/chat-api/handlers/types"
	"github.com/carousell/chat-api/pkg/client"
	"github.com/carousell/ct-go/pkg/logger/log"
	"github.com/nguyentranbao-ct/chat-bot/internal/config"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
)

type MessageHistoryRequest struct {
	UserID    string
	ChannelID string
	Limit     int
	BeforeTs  *int64
}

type ChatAPIClient interface {
	GetChannelInfo(ctx context.Context, channelID string) (*models.ChannelInfo, error)
	GetMessageHistory(ctx context.Context, userID, channelID string, limit int) (*models.MessageHistory, error)
	GetMessageHistoryWithParams(ctx context.Context, req MessageHistoryRequest) (*models.MessageHistory, error)
	SendMessage(ctx context.Context, message *models.OutgoingMessage) error
}

type chatAPIClient struct {
	client    client.InternalAPI
	projectID string
}

func NewChatAPIClient(conf *config.Config) ChatAPIClient {
	cfg := conf.ChatAPI
	log.Info("Creating chat-api client", log.Reflect("config", cfg))
	config := client.Config{
		BaseURL:   cfg.BaseURL,
		Service:   cfg.Service,
		ProjectID: cfg.ProjectID,
		Token:     cfg.APIKey,
	}

	chatClient, err := client.NewClient(config)
	if err != nil {
		panic(fmt.Sprintf("Failed to create chat-api client: %v", err))
	}

	return &chatAPIClient{
		client:    chatClient,
		projectID: cfg.ProjectID,
	}
}

func (c *chatAPIClient) GetChannelInfo(ctx context.Context, channelID string) (*models.ChannelInfo, error) {
	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	request := types.GetPlainUserChannelsRequest{
		ProjectID: c.projectID,
		ChannelID: channelID,
	}

	resp, err := c.client.GetPlainUserChannels(timeoutCtx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to get plain user channels: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("channel not found")
	}

	// Use the first user channel data to get channel info
	firstChannel := resp.Data[0]

	// Build current product info from item name and price
	currentProduct := firstChannel.ItemName
	if firstChannel.ItemPrice != "" {
		currentProduct = fmt.Sprintf("%s - %s", firstChannel.ItemName, firstChannel.ItemPrice)
	}

	// Convert chat-api response to our internal model
	channelInfo := &models.ChannelInfo{
		ID:             firstChannel.ChannelID,
		Name:           firstChannel.Name,
		ItemName:       firstChannel.ItemName,
		ItemPrice:      firstChannel.ItemPrice,
		CurrentProduct: currentProduct,
		Context:        getMetadataString(firstChannel.Metadata, "context"),
		Participants:   make([]models.Participant, 0, len(resp.Data)),
	}

	// Convert all user channels to participants
	for _, userChannel := range resp.Data {
		participant := models.Participant{
			UserID: userChannel.UserID,
			Role:   userChannel.Role,
		}
		channelInfo.Participants = append(channelInfo.Participants, participant)
	}

	return channelInfo, nil
}

func (c *chatAPIClient) GetMessageHistory(ctx context.Context, userID, channelID string, limit int) (*models.MessageHistory, error) {
	req := MessageHistoryRequest{
		UserID:    userID,
		ChannelID: channelID,
		Limit:     limit,
		BeforeTs:  nil,
	}
	return c.GetMessageHistoryWithParams(ctx, req)
}

func (c *chatAPIClient) GetMessageHistoryWithParams(ctx context.Context, req MessageHistoryRequest) (*models.MessageHistory, error) {
	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	request := types.GetChannelMessagesRequest{
		ProjectID: c.projectID,
		UserID:    req.UserID,
		ChannelID: req.ChannelID,
		BeforeTS:  req.BeforeTs,
		Limit:     uint(req.Limit),
		Order:     "desc",
	}

	resp, err := c.client.GetChannelMessages(timeoutCtx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel messages: %w", err)
	}

	// Convert chat-api response to our internal model
	history := &models.MessageHistory{
		Messages: make([]models.HistoryMessage, 0, len(resp.Data)),
		HasMore:  len(resp.Data) == int(request.Limit), // HasMore if we got the full limit
	}

	for _, msg := range resp.Data {
		// Generate message ID from CreatedAt and SenderID
		messageID := fmt.Sprintf("%d_%s", msg.CreatedAt, msg.SenderID)

		historyMsg := models.HistoryMessage{
			ID:        messageID,
			ChannelID: msg.ChannelID,
			SenderID:  msg.SenderID,
			Message:   msg.Message,
			CreatedAt: time.UnixMilli(msg.CreatedAt), // Convert from milliseconds
		}
		history.Messages = append(history.Messages, historyMsg)
	}

	return history, nil
}

func (c *chatAPIClient) SendMessage(ctx context.Context, message *models.OutgoingMessage) error {
	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	request := types.InternalSendMessageRequest{
		ProjectID: c.projectID,
		ChannelID: message.ChannelID,
		SenderID:  message.SenderID, // This should be the bot/system sender ID
		Message:   message.Message,
		Type:      "text",
	}

	_, err := c.client.SendMessage(timeoutCtx, request)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// Helper functions for metadata extraction
func getMetadataString(metadata map[string]interface{}, key string) string {
	if metadata == nil {
		return ""
	}
	if value, ok := metadata[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}
