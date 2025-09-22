package chatapi

import (
	"context"
	"fmt"
	"time"

	"github.com/carousell/chat-api/handlers/types"
	"github.com/carousell/chat-api/pkg/client"
	"github.com/nguyentranbao-ct/chat-bot/internal/config"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MessageHistoryRequest struct {
	UserID    primitive.ObjectID
	ChannelID primitive.ObjectID
	Limit     int
	BeforeTs  *int64
}

type Client interface {
	GetChannelInfo(ctx context.Context, channelID string) (*models.ChannelInfo, error)
	SendMessage(ctx context.Context, message *models.OutgoingMessage) error
}

type chatAPIClient struct {
	client    client.InternalAPI
	projectID string
}

func NewChatAPIClient(conf *config.Config) Client {
	cfg := conf.ChatAPI
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

	// Convert chat-api response to our internal model
	channelInfo := &models.ChannelInfo{
		ID:           firstChannel.ChannelID,
		Name:         firstChannel.Name,
		ItemName:     firstChannel.ItemName,
		ItemPrice:    firstChannel.ItemPrice,
		Context:      getMetadataString(firstChannel.Metadata, "context"),
		Participants: make([]models.Participant, 0, len(resp.Data)),
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
