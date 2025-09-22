package usecase

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/chatapi"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/mongodb"
)

type ChatUseCase struct {
	channelRepo         mongodb.ChannelRepository
	channelMemberRepo   mongodb.ChannelMemberRepository
	chatMessageRepo     mongodb.ChatMessageRepository
	messageEventRepo    mongodb.MessageEventRepository
	typingIndicatorRepo mongodb.TypingIndicatorRepository
	unreadCountRepo     mongodb.UnreadCountRepository
	chatAPIClient       chatapi.Client
	socketHandler       SocketBroadcaster
}

type SocketBroadcaster interface {
	BroadcastMessage(channelID string, message *models.ChatMessage)
	BroadcastMessageSent(userID string, message *models.ChatMessage)
	BroadcastTyping(channelID, userID string, isTyping bool)
	BroadcastMessageToUsers(userIDs []string, message *models.ChatMessage)
	BroadcastTypingToUsers(userIDs []string, channelID, userID string, isTyping bool)
}

func NewChatUseCase(
	channelRepo mongodb.ChannelRepository,
	channelMemberRepo mongodb.ChannelMemberRepository,
	chatMessageRepo mongodb.ChatMessageRepository,
	messageEventRepo mongodb.MessageEventRepository,
	typingIndicatorRepo mongodb.TypingIndicatorRepository,
	unreadCountRepo mongodb.UnreadCountRepository,
	chatAPIClient chatapi.Client,
	socketHandler SocketBroadcaster,
) *ChatUseCase {
	return &ChatUseCase{
		channelRepo:         channelRepo,
		channelMemberRepo:   channelMemberRepo,
		chatMessageRepo:     chatMessageRepo,
		messageEventRepo:    messageEventRepo,
		typingIndicatorRepo: typingIndicatorRepo,
		unreadCountRepo:     unreadCountRepo,
		chatAPIClient:       chatAPIClient,
		socketHandler:       socketHandler,
	}
}

func (uc *ChatUseCase) GetUserChannels(ctx context.Context, userID string) ([]interface{}, error) {
	results, err := uc.channelRepo.GetChannelsWithUnreadCount(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Convert []bson.M to []interface{}
	channels := make([]interface{}, len(results))
	for i, result := range results {
		channels[i] = result
	}

	return channels, nil
}

func (uc *ChatUseCase) GetChannelMembers(ctx context.Context, channelID primitive.ObjectID) ([]*models.ChannelMember, error) {
	return uc.channelMemberRepo.GetChannelMembers(ctx, channelID)
}

func (uc *ChatUseCase) SendMessage(ctx context.Context, channelID primitive.ObjectID, senderID, content, messageType string, blocks []models.MessageBlock, metadata map[string]interface{}) (*models.ChatMessage, error) {
	// Get channel info
	channel, err := uc.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("channel not found: %w", err)
	}

	// Create message in our database
	message := &models.ChatMessage{
		ChannelID:         channelID,
		ExternalChannelID: channel.ExternalChannelID,
		SenderID:          senderID,
		MessageType:       messageType,
		Content:           content,
		Blocks:            blocks,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		DeliveryStatus:    "sent",
		Metadata: models.MessageMetadata{
			Source:     "api",
			IsFromBot:  false,
			CustomData: metadata,
		},
	}

	if err := uc.chatMessageRepo.Create(ctx, message); err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	// Send to chat-api (async - don't block the response)
	go func() {
		outgoingMsg := models.OutgoingMessage{
			ChannelID: channel.ExternalChannelID,
			SenderID:  senderID,
			Message:   content,
		}

		if err := uc.chatAPIClient.SendMessage(context.Background(), &outgoingMsg); err != nil {
			// Update delivery status on failure
			uc.chatMessageRepo.UpdateDeliveryStatus(context.Background(), message.ID, "failed")
		} else {
			// Update delivery status on success
			uc.chatMessageRepo.UpdateDeliveryStatus(context.Background(), message.ID, "delivered")
		}
	}()

	// Update channel last message time
	if err := uc.channelRepo.UpdateLastMessage(ctx, channelID); err != nil {
		// Log error but don't fail the message send
		fmt.Printf("Failed to update channel last message time: %v\n", err)
	}

	// Increment unread count for other members
	go uc.incrementUnreadCountForOthers(context.Background(), channelID, senderID)

	// Create message event for real-time sync
	go uc.messageEventRepo.CreateEvent(context.Background(), channelID, "message_sent", &message.ID, senderID, map[string]interface{}{
		"message_type": messageType,
		"content":      content,
	})

	return message, nil
}

func (uc *ChatUseCase) GetChannelMessages(ctx context.Context, channelID primitive.ObjectID, limit int, before *primitive.ObjectID) ([]*models.ChatMessage, error) {
	return uc.chatMessageRepo.GetChannelMessages(ctx, channelID, limit, before)
}

func (uc *ChatUseCase) GetChannelEvents(ctx context.Context, channelID primitive.ObjectID, sinceTime time.Time) ([]*models.MessageEvent, error) {
	return uc.messageEventRepo.GetChannelEvents(ctx, channelID, sinceTime)
}

func (uc *ChatUseCase) MarkAsRead(ctx context.Context, channelID primitive.ObjectID, userID string, lastReadMessageID primitive.ObjectID) error {
	return uc.unreadCountRepo.MarkAsRead(ctx, channelID, userID, lastReadMessageID)
}

func (uc *ChatUseCase) SetTyping(ctx context.Context, channelID primitive.ObjectID, userID string, isTyping bool) error {
	if err := uc.typingIndicatorRepo.SetTyping(ctx, channelID, userID, isTyping); err != nil {
		return err
	}

	// Create typing event for real-time updates
	eventType := "user_typing_stop"
	if isTyping {
		eventType = "user_typing_start"
	}

	go uc.messageEventRepo.CreateEvent(context.Background(), channelID, eventType, nil, userID, map[string]interface{}{
		"is_typing": isTyping,
	})

	return nil
}

func (uc *ChatUseCase) ProcessIncomingMessage(ctx context.Context, kafkaMessage models.KafkaMessageData) error {
	// Find or create channel
	channel, err := uc.findOrCreateChannel(ctx, kafkaMessage)
	if err != nil {
		return fmt.Errorf("failed to find/create channel: %w", err)
	}

	// Create message in our database
	message := &models.ChatMessage{
		ChannelID:         channel.ID,
		ExternalChannelID: kafkaMessage.ChannelID,
		SenderID:          kafkaMessage.SenderID,
		MessageType:       kafkaMessage.Type,
		Content:           kafkaMessage.Message,
		CreatedAt:         time.Unix(kafkaMessage.CreatedAt, 0),
		UpdatedAt:         time.Now(),
		DeliveryStatus:    "delivered",
		Metadata: models.MessageMetadata{
			Source:            "kafka",
			IsFromBot:         false,
			OriginalTimestamp: kafkaMessage.CreatedAt,
			CustomData:        kafkaMessage.Metadata,
		},
	}

	if err := uc.chatMessageRepo.Create(ctx, message); err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	// Update channel last message time
	if err := uc.channelRepo.UpdateLastMessage(ctx, channel.ID); err != nil {
		fmt.Printf("Failed to update channel last message time: %v\n", err)
	}

	// Increment unread count for all members except sender
	go uc.incrementUnreadCountForOthers(context.Background(), channel.ID, kafkaMessage.SenderID)

	// Create message event for real-time sync
	go uc.messageEventRepo.CreateEvent(context.Background(), channel.ID, "message_received", &message.ID, kafkaMessage.SenderID, map[string]interface{}{
		"source":       "kafka",
		"message_type": kafkaMessage.Type,
		"content":      kafkaMessage.Message,
	})

	// Broadcast incoming message to all channel members via socket
	if uc.socketHandler != nil {
		go func() {
			members, err := uc.channelMemberRepo.GetChannelMembers(context.Background(), channel.ID)
			if err != nil {
				fmt.Printf("Failed to get channel members for socket broadcast: %v\n", err)
				return
			}

			userIDs := make([]string, 0, len(members))
			for _, member := range members {
				userIDs = append(userIDs, member.UserID)
			}

			uc.socketHandler.BroadcastMessageToUsers(userIDs, message)
		}()
	}

	return nil
}

func (uc *ChatUseCase) findOrCreateChannel(ctx context.Context, kafkaMessage models.KafkaMessageData) (*models.Channel, error) {
	// Try to find existing channel
	channel, err := uc.channelRepo.GetByExternalChannelID(ctx, kafkaMessage.ChannelID)
	if err == nil {
		return channel, nil
	}

	// Channel doesn't exist, create it
	// We'll need to get channel info from chat-api
	channelInfo, err := uc.chatAPIClient.GetChannelInfo(ctx, kafkaMessage.ChannelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel info from chat-api: %w", err)
	}

	channel = &models.Channel{
		ExternalChannelID: kafkaMessage.ChannelID,
		Name:              channelInfo.Name,
		ItemName:          channelInfo.ItemName,
		ItemPrice:         channelInfo.ItemPrice,
		Context:           channelInfo.Context,
		Type:              "direct", // Default type
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		IsArchived:        false,
	}

	if err := uc.channelRepo.Create(ctx, channel); err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	// Add channel members
	for _, participant := range channelInfo.Participants {
		member := &models.ChannelMember{
			ChannelID: channel.ID,
			UserID:    participant.UserID,
			Role:      participant.Role,
			JoinedAt:  time.Now(),
			IsActive:  true,
		}

		if err := uc.channelMemberRepo.Create(ctx, member); err != nil {
			fmt.Printf("Failed to create channel member: %v\n", err)
		}
	}

	return channel, nil
}

func (uc *ChatUseCase) incrementUnreadCountForOthers(ctx context.Context, channelID primitive.ObjectID, senderID string) {
	members, err := uc.channelMemberRepo.GetChannelMembers(ctx, channelID)
	if err != nil {
		fmt.Printf("Failed to get channel members for unread count: %v\n", err)
		return
	}

	for _, member := range members {
		if member.UserID != senderID {
			if err := uc.unreadCountRepo.IncrementUnreadCount(ctx, channelID, member.UserID); err != nil {
				fmt.Printf("Failed to increment unread count for user %s: %v\n", member.UserID, err)
			}
		}
	}
}

// Cleanup methods for background maintenance
func (uc *ChatUseCase) CleanupExpiredEvents(ctx context.Context) error {
	return uc.messageEventRepo.CleanupExpiredEvents(ctx)
}

func (uc *ChatUseCase) CleanupExpiredTyping(ctx context.Context) error {
	return uc.typingIndicatorRepo.CleanupExpiredTyping(ctx)
}