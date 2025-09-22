package usecase

import (
	"context"
	"fmt"
	"time"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/mongodb"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/vendors"
	"github.com/nguyentranbao-ct/chat-bot/pkg/util"
)

type ChatUseCase struct {
	channelRepo       mongodb.ChannelRepository
	channelMemberRepo mongodb.ChannelMemberRepository
	chatMessageRepo   mongodb.ChatMessageRepository
	messageEventRepo  mongodb.MessageEventRepository
	unreadCountRepo   mongodb.UnreadCountRepository
	userRepo          mongodb.UserRepository
	userAttributeRepo mongodb.UserAttributeRepository
	vendorRegistry    *vendors.VendorRegistry
	socketHandler     SocketBroadcaster
	llmUsecaseV2      LLMUsecaseV2
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
	unreadCountRepo mongodb.UnreadCountRepository,
	userRepo mongodb.UserRepository,
	userAttributeRepo mongodb.UserAttributeRepository,
	vendorRegistry *vendors.VendorRegistry,
	socketHandler SocketBroadcaster,
	llmUsecaseV2 LLMUsecaseV2,
) *ChatUseCase {
	return &ChatUseCase{
		channelRepo:       channelRepo,
		channelMemberRepo: channelMemberRepo,
		chatMessageRepo:   chatMessageRepo,
		messageEventRepo:  messageEventRepo,
		unreadCountRepo:   unreadCountRepo,
		userRepo:          userRepo,
		userAttributeRepo: userAttributeRepo,
		vendorRegistry:    vendorRegistry,
		socketHandler:     socketHandler,
		llmUsecaseV2:      llmUsecaseV2,
	}
}

func (uc *ChatUseCase) GetUserChannels(ctx context.Context, userID primitive.ObjectID) ([]interface{}, error) {
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

// SendMessageParams contains parameters for sending a message
type SendMessageParams struct {
	ChannelID   primitive.ObjectID     `json:"channel_id"`
	SenderID    primitive.ObjectID     `json:"sender_id"`
	Content     string                 `json:"content"`
	MessageType string                 `json:"message_type"`
	Blocks      []models.MessageBlock  `json:"blocks,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func (uc *ChatUseCase) SendMessage(ctx context.Context, params SendMessageParams) (*models.ChatMessage, error) {
	// Get channel info
	channel, err := uc.channelRepo.GetByID(ctx, params.ChannelID)
	if err != nil {
		return nil, fmt.Errorf("channel not found: %w", err)
	}

	// Create message in our database
	message := &models.ChatMessage{
		ChannelID:      params.ChannelID,
		SenderID:       params.SenderID,
		Content:        params.Content,
		Blocks:         params.Blocks,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		DeliveryStatus: "sent",
		Metadata: models.MessageMetadata{
			Source:     "api",
			IsFromBot:  false,
			CustomData: params.Metadata,
		},
	}

	if err := uc.chatMessageRepo.Create(ctx, message); err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	// Post-process the sent message
	go uc.postProcessSentMessage(ctx, message, channel, params)

	// Update channel last message time synchronously
	if err := uc.channelRepo.UpdateLastMessage(ctx, params.ChannelID); err != nil {
		log.Warnw(ctx, "Failed to update channel last message time", "error", err)
	}

	return message, nil
}

// postProcessSentMessage handles all post-processing after a message is sent
func (uc *ChatUseCase) postProcessSentMessage(ctx context.Context, message *models.ChatMessage, channel *models.Channel, params SendMessageParams) {
	ctx, cancel := util.NewTimeoutContext(ctx, 10*time.Second)
	defer cancel()

	// Increment unread count for other members
	uc.incrementUnreadCountForOthers(ctx, params.ChannelID, params.SenderID)

	// Create message event for real-time sync
	uc.createMessageSentEvent(ctx, message, params)

	// Send to external vendor asynchronously
	uc.sendToExternalVendor(ctx, message, channel, params)
}

// sendToExternalVendor sends the message to the external vendor
func (uc *ChatUseCase) sendToExternalVendor(ctx context.Context, message *models.ChatMessage, channel *models.Channel, params SendMessageParams) {
	timeoutCtx, cancel := util.NewTimeoutContext(ctx, 15*time.Second)
	defer cancel()

	// Get vendor instance
	vendorInstance, err := uc.vendorRegistry.GetVendorByName(channel.Vendor.Name)
	if err != nil {
		log.Errorw(timeoutCtx, "Failed to get vendor for message sending",
			"vendor_name", channel.Vendor.Name,
			"channel_id", channel.Vendor.ChannelID,
			"error", err)
		uc.chatMessageRepo.UpdateDeliveryStatus(timeoutCtx, message.ID, "failed")
		return
	}

	// get sender chotot id from user attributes
	idAttr, err := uc.userAttributeRepo.GetByUserIDAndKey(timeoutCtx, params.SenderID, "chotot_id")
	if err != nil {
		log.Errorw(timeoutCtx, "Failed to get sender chotot_id attribute",
			"user_id", params.SenderID.Hex(),
			"error", err)
		uc.chatMessageRepo.UpdateDeliveryStatus(timeoutCtx, message.ID, "failed")
		return
	}

	// Prepare vendor send params
	sendParams := vendors.SendMessageParams{
		ChannelID:   channel.Vendor.ChannelID,
		SenderID:    idAttr.Value,
		Content:     params.Content,
		MessageType: params.MessageType,
		Metadata:    params.Metadata,
	}

	// Send via vendor
	if err := vendorInstance.SendMessage(timeoutCtx, sendParams); err != nil {
		log.Errorw(timeoutCtx, "Failed to send message via vendor",
			"vendor_name", channel.Vendor.Name,
			"channel_id", params.ChannelID,
			"error", err)
		uc.chatMessageRepo.UpdateDeliveryStatus(timeoutCtx, message.ID, "failed")
	} else {
		log.Debugw(timeoutCtx, "Message sent successfully via vendor",
			"vendor_name", channel.Vendor.Name,
			"channel_id", params.ChannelID)
		uc.chatMessageRepo.UpdateDeliveryStatus(timeoutCtx, message.ID, "delivered")
	}
}

// createMessageSentEvent creates a real-time event for the sent message
func (uc *ChatUseCase) createMessageSentEvent(ctx context.Context, message *models.ChatMessage, params SendMessageParams) {
	eventParams := mongodb.CreateEventParams{
		ChannelID: params.ChannelID,
		EventType: "message_sent",
		MessageID: &message.ID,
		UserID:    params.SenderID,
		EventData: map[string]any{
			"message_type": params.MessageType,
			"content":      params.Content,
		},
	}

	if err := uc.messageEventRepo.CreateEvent(ctx, eventParams); err != nil {
		log.Errorw(ctx, "Failed to create message sent event", "error", err)
	}
}

func (uc *ChatUseCase) GetChannelMessages(ctx context.Context, channelID primitive.ObjectID, limit int, before *primitive.ObjectID) ([]*models.ChatMessage, error) {
	return uc.chatMessageRepo.GetChannelMessages(ctx, channelID, limit, before)
}

func (uc *ChatUseCase) GetChannelEvents(ctx context.Context, channelID primitive.ObjectID, sinceTime time.Time) ([]*models.MessageEvent, error) {
	return uc.messageEventRepo.GetChannelEvents(ctx, channelID, sinceTime)
}

func (uc *ChatUseCase) MarkAsRead(ctx context.Context, channelID primitive.ObjectID, userID primitive.ObjectID, lastReadMessageID primitive.ObjectID) error {
	return uc.unreadCountRepo.MarkAsRead(ctx, channelID, userID, lastReadMessageID)
}

func (uc *ChatUseCase) ProcessIncomingMessage(ctx context.Context, kafkaMessage models.KafkaMessageData) error {
	// Detect vendor for deduplication
	vendorType := vendors.VendorTypeChotot

	// Check if sender is internal user (loop prevention)
	if uc.isInternalUser(ctx, kafkaMessage.SenderID, vendorType) {
		log.Debugw(ctx, "Skipping message from internal user", "sender_id", kafkaMessage.SenderID, "vendor", vendorType)
		return nil
	}

	// Find or sync user from vendor
	user, err := uc.findOrSyncUser(ctx, kafkaMessage.SenderID, vendorType)
	if err != nil {
		log.Warnw(ctx, "Failed to sync user, continuing without user info", "error", err, "sender_id", kafkaMessage.SenderID)
	}

	// Find or create channel
	channel, err := uc.findOrCreateChannel(ctx, kafkaMessage.ChannelID, vendorType)
	if err != nil {
		return fmt.Errorf("failed to find/create channel: %w", err)
	}

	// Create message in our database
	message := &models.ChatMessage{
		ChannelID:      channel.ID,
		SenderID:       user.ID,
		Content:        kafkaMessage.Message,
		CreatedAt:      time.Unix(kafkaMessage.CreatedAt, 0),
		UpdatedAt:      time.Now(),
		DeliveryStatus: "delivered",
		Metadata: models.MessageMetadata{
			Source:            "kafka",
			IsFromBot:         false,
			OriginalTimestamp: kafkaMessage.CreatedAt,
			CustomData:        kafkaMessage.Metadata,
		},
	}

	if err := uc.chatMessageRepo.Upsert(ctx, message); err != nil {
		return fmt.Errorf("failed to upsert message: %w", err)
	}

	// Post-process the incoming message
	go uc.postProcessIncomingMessage(ctx, message, channel, vendorType)

	// Update channel last message time synchronously
	if err := uc.channelRepo.UpdateLastMessage(ctx, channel.ID); err != nil {
		log.Warnw(ctx, "Failed to update channel last message time", "error", err)
	}

	return nil
}

// postProcessIncomingMessage handles all post-processing after an incoming message is received
func (uc *ChatUseCase) postProcessIncomingMessage(ctx context.Context, message *models.ChatMessage, channel *models.Channel, vendorType vendors.VendorType) {
	ctx, cancel := util.NewTimeoutContext(ctx, 10*time.Second)
	defer cancel()
	// Increment unread count for all members except sender
	uc.incrementUnreadCountForOthers(ctx, channel.ID, message.SenderID)

	// Create message event for real-time sync
	uc.createMessageReceivedEvent(ctx, message, channel)

	// Broadcast to socket connections
	uc.broadcastIncomingMessage(ctx, message, channel.ID)

	uc.llmUsecaseV2.TriggerLLM(ctx, message, channel)
}

// createMessageReceivedEvent creates a real-time event for the received message
func (uc *ChatUseCase) createMessageReceivedEvent(ctx context.Context, message *models.ChatMessage, channel *models.Channel) {
	eventParams := mongodb.CreateEventParams{
		ChannelID: channel.ID,
		EventType: "message_received",
		MessageID: &message.ID,
		UserID:    message.SenderID,
		EventData: map[string]any{
			"source":  "kafka",
			"content": message.Content,
		},
	}

	if err := uc.messageEventRepo.CreateEvent(ctx, eventParams); err != nil {
		log.Errorw(ctx, "Failed to create message received event", "error", err)
	}
}

// broadcastIncomingMessage broadcasts the incoming message to all channel members via socket
func (uc *ChatUseCase) broadcastIncomingMessage(ctx context.Context, message *models.ChatMessage, channelID primitive.ObjectID) {
	if uc.socketHandler == nil {
		return
	}

	members, err := uc.channelMemberRepo.GetChannelMembers(ctx, channelID)
	if err != nil {
		log.Errorw(ctx, "Failed to get channel members for socket broadcast", "error", err)
		return
	}

	userIDs := make([]string, 0, len(members))
	for _, member := range members {
		userIDs = append(userIDs, member.UserID.Hex())
	}

	uc.socketHandler.BroadcastMessageToUsers(userIDs, message)
}

func (uc *ChatUseCase) findOrCreateChannel(ctx context.Context, channelID string, vendorType vendors.VendorType) (*models.Channel,
	error,
) {
	// Try to find existing channel using vendor info
	channel, err := uc.channelRepo.GetByVendorChannelID(ctx, string(vendorType), channelID)
	if err == nil {
		return channel, nil
	}

	// Fallback: try legacy lookup for backward compatibility
	channel, err = uc.channelRepo.GetByExternalChannelID(ctx, channelID)
	if err == nil {
		return channel, nil
	}

	// Channel doesn't exist, create it using the appropriate vendor
	vendorInstance, err := uc.vendorRegistry.GetVendor(vendorType)
	if err != nil {
		return nil, fmt.Errorf("failed to get vendor %s: %w", vendorType, err)
	}

	// Get channel info from vendor
	vendorChannelInfo, err := vendorInstance.GetChannelInfo(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel info from vendor %s: %w", vendorType, err)
	}

	// Create metadata from vendor channel info
	metadata := make(map[string]any)
	for key, value := range vendorChannelInfo.Metadata {
		metadata[key] = value
	}
	ts := time.Now()

	channel = &models.Channel{
		Vendor: models.ChannelVendor{
			ChannelID: channelID,
			Name:      string(vendorType),
		},
		Name:          vendorChannelInfo.Name,
		Context:       vendorChannelInfo.Context,
		LastMessageAt: &ts,
		Metadata:      metadata,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		IsArchived:    false,
	}

	if err := uc.channelRepo.Create(ctx, channel); err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	// Add channel members - map vendor user IDs to internal user IDs
	for _, participant := range vendorChannelInfo.Participants {
		// Find or sync the user to get internal user ID
		internalUser, err := uc.findOrSyncUser(ctx, participant.UserID, vendorType)
		if err != nil {
			log.Warnw(ctx, "Failed to sync participant user, skipping", "error", err, "vendor_user_id", participant.UserID)
			continue
		}

		member := &models.ChannelMember{
			ChannelID: channel.ID,
			UserID:    internalUser.ID, // Use internal user ID
			Role:      participant.Role,
			JoinedAt:  time.Now(),
			IsActive:  true,
		}

		if err := uc.channelMemberRepo.Create(ctx, member); err != nil {
			log.Warnw(ctx, "Failed to create channel member", "error", err, "user_id", internalUser.ID.Hex())
		}
	}

	return channel, nil
}

func (uc *ChatUseCase) incrementUnreadCountForOthers(ctx context.Context, channelID primitive.ObjectID, senderID primitive.ObjectID) {
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

// isInternalUser checks if the sender is an internal user to prevent loops
func (uc *ChatUseCase) isInternalUser(ctx context.Context, senderID string, vendorType vendors.VendorType) bool {
	switch vendorType {
	case vendors.VendorTypeChotot:
		// Try to find user by chotot_id attribute
		user, err := uc.getUserByChototID(ctx, senderID)
		if err != nil {
			// User not found in our system, so it's external
			return false
		}
		// Check if user is marked as internal
		return user.IsInternal
	default:
		// For future vendors, add similar lookup logic
		return false
	}
}

// findOrSyncUser finds or syncs user information from vendor
func (uc *ChatUseCase) findOrSyncUser(ctx context.Context, senderID string, vendorType vendors.VendorType) (*models.User, error) {
	switch vendorType {
	case vendors.VendorTypeChotot:
		// Try to find existing user by chotot_id attribute
		user, err := uc.getUserByChototID(ctx, senderID)
		if err == nil {
			return user, nil // User already exists
		}

		// User doesn't exist, sync from vendor
		vendorInstance, err := uc.vendorRegistry.GetVendor(vendorType)
		if err != nil {
			return nil, fmt.Errorf("failed to get vendor %s: %w", vendorType, err)
		}

		vendorUserInfo, err := vendorInstance.GetUserInfo(ctx, senderID)
		if err != nil {
			return nil, fmt.Errorf("failed to get user info from vendor %s: %w", vendorType, err)
		}

		// Create user from vendor info
		user = &models.User{
			Name:       vendorUserInfo.Name,
			Email:      vendorUserInfo.Email,
			IsActive:   vendorUserInfo.IsActive,
			IsInternal: false, // External users are never internal
		}

		// Upsert user to database
		if err := uc.userRepo.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to upsert user: %w", err)
		}

		// Create chotot_id attribute after user is created
		if err := uc.createUserAttribute(ctx, user.ID, "chotot_id", senderID, []string{"chotot", "link_id"}); err != nil {
			log.Warnw(ctx, "Failed to create chotot_id attribute for user", "error", err, "user_id", user.ID.Hex())
		}

		log.Infow(ctx, "Synced new user from vendor", "vendor", vendorType, "user_id", senderID)
		return user, nil

	default:
		return nil, fmt.Errorf("user sync not implemented for vendor %s", vendorType)
	}
}

// getUserByChototID finds a user by their chotot_id attribute
func (uc *ChatUseCase) getUserByChototID(ctx context.Context, chototID string) (*models.User, error) {
	// Find user attribute with key="chotot_id" and value=chototID
	attr, err := uc.getUserAttributeByKeyAndValue(ctx, "chotot_id", chototID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user attribute for chotot_id %s: %w", chototID, err)
	}

	// Get the user by ID
	user, err := uc.userRepo.GetByID(ctx, attr.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by ID %s: %w", attr.UserID.Hex(), err)
	}

	return user, nil
}

// getUserAttributeByKeyAndValue gets a user attribute by key and value
func (uc *ChatUseCase) getUserAttributeByKeyAndValue(ctx context.Context, key, value string) (*models.UserAttribute, error) {
	return uc.userAttributeRepo.GetByKeyAndValue(ctx, key, value)
}

// createUserAttribute creates a user attribute
func (uc *ChatUseCase) createUserAttribute(ctx context.Context, userID primitive.ObjectID, key, value string, tags []string) error {
	attr := &models.UserAttribute{
		UserID: userID,
		Key:    key,
		Value:  value,
		Tags:   tags,
	}

	return uc.userAttributeRepo.Upsert(ctx, attr)
}
