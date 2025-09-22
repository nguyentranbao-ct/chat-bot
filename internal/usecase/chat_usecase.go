package usecase

import (
	"context"
	"fmt"
	"time"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/mongodb"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/partners"
	"github.com/nguyentranbao-ct/chat-bot/pkg/util"
)

type ChatUseCase struct {
	roomRepo          mongodb.RoomRepository
	roomMemberRepo    mongodb.RoomMemberRepository
	chatMessageRepo   mongodb.ChatMessageRepository
	messageEventRepo  mongodb.MessageEventRepository
	unreadCountRepo   mongodb.UnreadCountRepository
	userRepo          mongodb.UserRepository
	userAttributeRepo mongodb.UserAttributeRepository
	partnerRegistry   *partners.PartnerRegistry
	socketHandler     SocketBroadcaster
	llmUsecaseV2      LLMUsecaseV2
}

type SocketBroadcaster interface {
	BroadcastMessage(roomID string, message *models.ChatMessage)
	BroadcastMessageSent(userID string, message *models.ChatMessage)
	BroadcastTyping(roomID, userID string, isTyping bool)
	BroadcastMessageToUsers(userIDs []string, message *models.ChatMessage)
	BroadcastTypingToUsers(userIDs []string, roomID, userID string, isTyping bool)
}

func NewChatUseCase(
	roomRepo mongodb.RoomRepository,
	roomMemberRepo mongodb.RoomMemberRepository,
	chatMessageRepo mongodb.ChatMessageRepository,
	messageEventRepo mongodb.MessageEventRepository,
	unreadCountRepo mongodb.UnreadCountRepository,
	userRepo mongodb.UserRepository,
	userAttributeRepo mongodb.UserAttributeRepository,
	partnerRegistry *partners.PartnerRegistry,
	socketHandler SocketBroadcaster,
	llmUsecaseV2 LLMUsecaseV2,
) *ChatUseCase {
	return &ChatUseCase{
		roomRepo:          roomRepo,
		roomMemberRepo:    roomMemberRepo,
		chatMessageRepo:   chatMessageRepo,
		messageEventRepo:  messageEventRepo,
		unreadCountRepo:   unreadCountRepo,
		userRepo:          userRepo,
		userAttributeRepo: userAttributeRepo,
		partnerRegistry:   partnerRegistry,
		socketHandler:     socketHandler,
		llmUsecaseV2:      llmUsecaseV2,
	}
}

func (uc *ChatUseCase) GetUserRooms(ctx context.Context, userID primitive.ObjectID) ([]interface{}, error) {
	results, err := uc.roomRepo.GetRoomsWithUnreadCount(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Convert []bson.M to []interface{}
	rooms := make([]interface{}, len(results))
	for i, result := range results {
		rooms[i] = result
	}

	return rooms, nil
}

func (uc *ChatUseCase) GetRoomMembers(ctx context.Context, roomID primitive.ObjectID) ([]*models.RoomMember, error) {
	return uc.roomMemberRepo.GetRoomMembers(ctx, roomID)
}

// SendMessageParams contains parameters for sending a message
type SendMessageParams struct {
	RoomID      primitive.ObjectID     `json:"room_id"`
	SenderID    primitive.ObjectID     `json:"sender_id"`
	Content     string                 `json:"content"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	SkipPartner bool                   `json:"skip_partner,omitempty"`
}

func (uc *ChatUseCase) SendMessage(ctx context.Context, params SendMessageParams) (*models.ChatMessage, error) {
	// Get room info
	room, err := uc.roomRepo.GetByID(ctx, params.RoomID)
	if err != nil {
		return nil, fmt.Errorf("room not found: %w", err)
	}

	// Create message in our database
	message := &models.ChatMessage{
		RoomID:    params.RoomID,
		SenderID:  params.SenderID,
		Content:   params.Content,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata: models.MessageMetadata{
			CustomData: params.Metadata,
		},
	}

	if err := uc.chatMessageRepo.Create(ctx, message); err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	// Post-process the sent message
	go uc.postProcessSentMessage(ctx, message, room, params)

	// Update room last message time synchronously
	if err := uc.roomRepo.UpdateLastMessage(ctx, params.RoomID); err != nil {
		log.Warnw(ctx, "Failed to update room last message time", "error", err)
	}

	return message, nil
}

// postProcessSentMessage handles all post-processing after a message is sent
func (uc *ChatUseCase) postProcessSentMessage(ctx context.Context, message *models.ChatMessage, room *models.Room, params SendMessageParams) {
	ctx, cancel := util.NewTimeoutContext(ctx, 10*time.Second)
	defer cancel()

	// Increment unread count for other members
	uc.incrementUnreadCountForOthers(ctx, params.RoomID, params.SenderID)

	// Create message event for real-time sync
	uc.createMessageSentEvent(ctx, message, params)

	// Send to external partner asynchronously, unless skipped
	if !params.SkipPartner {
		uc.sendToExternalPartner(ctx, message, room, params)
	}
}

// sendToExternalPartner sends the message to the external partner
func (uc *ChatUseCase) sendToExternalPartner(ctx context.Context, message *models.ChatMessage, room *models.Room, params SendMessageParams) {
	timeoutCtx, cancel := util.NewTimeoutContext(ctx, 15*time.Second)
	defer cancel()

	// Get partner instance
	partnerInstance, err := uc.partnerRegistry.GetPartnerByName(room.Source.Name)
	if err != nil {
		log.Errorw(timeoutCtx, "Failed to get partner for message sending",
			"partner_name", room.Source.Name,
			"room_id", room.Source.RoomID,
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

	// Prepare partner send params
	sendParams := partners.SendMessageParams{
		RoomID:   room.Source.RoomID,
		SenderID: idAttr.Value,
		Content:  params.Content,
		Metadata: params.Metadata,
	}

	// Send via partner
	if err := partnerInstance.SendMessage(timeoutCtx, sendParams); err != nil {
		log.Errorw(timeoutCtx, "Failed to send message via partner",
			"partner_name", room.Source.Name,
			"room_id", params.RoomID,
			"error", err)
		uc.chatMessageRepo.UpdateDeliveryStatus(timeoutCtx, message.ID, "failed")
	} else {
		log.Debugw(timeoutCtx, "Message sent successfully via partner",
			"partner_name", room.Source.Name,
			"room_id", params.RoomID)
		uc.chatMessageRepo.UpdateDeliveryStatus(timeoutCtx, message.ID, "delivered")
	}
}

// createMessageSentEvent creates a real-time event for the sent message
func (uc *ChatUseCase) createMessageSentEvent(ctx context.Context, message *models.ChatMessage, params SendMessageParams) {
	eventParams := mongodb.CreateEventParams{
		RoomID:    params.RoomID,
		EventType: "message_sent",
		MessageID: &message.ID,
		UserID:    params.SenderID,
		EventData: map[string]any{
			"content": params.Content,
		},
	}

	if err := uc.messageEventRepo.CreateEvent(ctx, eventParams); err != nil {
		log.Errorw(ctx, "Failed to create message sent event", "error", err)
	}
}

func (uc *ChatUseCase) GetRoomMessages(ctx context.Context, roomID primitive.ObjectID, limit int, before *primitive.ObjectID) ([]*models.ChatMessage, error) {
	return uc.chatMessageRepo.GetRoomMessages(ctx, roomID, limit, before)
}

func (uc *ChatUseCase) GetRoomEvents(ctx context.Context, roomID primitive.ObjectID, sinceTime time.Time) ([]*models.MessageEvent, error) {
	return uc.messageEventRepo.GetRoomEvents(ctx, roomID, sinceTime)
}

func (uc *ChatUseCase) MarkAsRead(ctx context.Context, roomID primitive.ObjectID, userID primitive.ObjectID, lastReadMessageID primitive.ObjectID) error {
	return uc.unreadCountRepo.MarkAsRead(ctx, roomID, userID, lastReadMessageID)
}

func (uc *ChatUseCase) ProcessIncomingMessage(ctx context.Context, kafkaMessage models.KafkaMessageData) error {
	// Detect partner for deduplication
	partnerType := partners.PartnerTypeChotot

	// Check if sender is internal user (loop prevention)
	if uc.isInternalUser(ctx, kafkaMessage.SenderID, partnerType) {
		log.Debugw(ctx, "Skipping message from internal user", "sender_id", kafkaMessage.SenderID, "partner", partnerType)
		return nil
	}

	// Find or sync user from partner
	user, err := uc.findOrSyncUser(ctx, kafkaMessage.SenderID, partnerType)
	if err != nil {
		log.Warnw(ctx, "Failed to sync user, continuing without user info", "error", err, "sender_id", kafkaMessage.SenderID)
	}

	// Find or create room
	room, err := uc.findOrCreateRoom(ctx, kafkaMessage.ChannelID, partnerType)
	if err != nil {
		return fmt.Errorf("failed to find/create room: %w", err)
	}

	// Create message in our database
	message := &models.ChatMessage{
		RoomID:    room.ID,
		SenderID:  user.ID,
		Content:   kafkaMessage.Message,
		CreatedAt: time.Unix(kafkaMessage.CreatedAt, 0),
		UpdatedAt: time.Now(),
		Metadata: models.MessageMetadata{
			OriginalTimestamp: kafkaMessage.CreatedAt,
			CustomData:        kafkaMessage.Metadata,
		},
	}

	if err := uc.chatMessageRepo.Upsert(ctx, message); err != nil {
		return fmt.Errorf("failed to upsert message: %w", err)
	}

	// Post-process the incoming message
	go uc.postProcessIncomingMessage(ctx, message, room, partnerType)

	// Update room last message time synchronously
	if err := uc.roomRepo.UpdateLastMessage(ctx, room.ID); err != nil {
		log.Warnw(ctx, "Failed to update room last message time", "error", err)
	}

	return nil
}

// postProcessIncomingMessage handles all post-processing after an incoming message is received
func (uc *ChatUseCase) postProcessIncomingMessage(ctx context.Context, message *models.ChatMessage, room *models.Room, partnerType partners.PartnerType) {
	ctx, cancel := util.NewTimeoutContext(ctx, 10*time.Second)
	defer cancel()
	// Increment unread count for all members except sender
	uc.incrementUnreadCountForOthers(ctx, room.ID, message.SenderID)

	// Create message event for real-time sync
	uc.createMessageReceivedEvent(ctx, message, room)

	// Broadcast to socket connections
	uc.broadcastIncomingMessage(ctx, message, room.ID)

	uc.llmUsecaseV2.TriggerLLM(ctx, message, room)
}

// createMessageReceivedEvent creates a real-time event for the received message
func (uc *ChatUseCase) createMessageReceivedEvent(ctx context.Context, message *models.ChatMessage, room *models.Room) {
	eventParams := mongodb.CreateEventParams{
		RoomID:    room.ID,
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

// broadcastIncomingMessage broadcasts the incoming message to all room members via socket
func (uc *ChatUseCase) broadcastIncomingMessage(ctx context.Context, message *models.ChatMessage, roomID primitive.ObjectID) {
	if uc.socketHandler == nil {
		return
	}

	members, err := uc.roomMemberRepo.GetRoomMembers(ctx, roomID)
	if err != nil {
		log.Errorw(ctx, "Failed to get room members for socket broadcast", "error", err)
		return
	}

	userIDs := make([]string, 0, len(members))
	for _, member := range members {
		userIDs = append(userIDs, member.UserID.Hex())
	}

	uc.socketHandler.BroadcastMessageToUsers(userIDs, message)
}

func (uc *ChatUseCase) findOrCreateRoom(ctx context.Context, roomID string, partner partners.PartnerType) (*models.Room,
	error,
) {
	// Try to find existing room using partner info
	room, err := uc.roomRepo.GetByPartnerRoomID(ctx, string(partner), roomID)
	if err == nil {
		return room, nil
	}

	// Room doesn't exist, create it using the appropriate partner
	partnerInstance, err := uc.partnerRegistry.GetPartner(partner)
	if err != nil {
		return nil, fmt.Errorf("failed to get partner %s: %w", partner, err)
	}

	// Get room info from partner
	partnerRoomInfo, err := partnerInstance.GetRoomInfo(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get room info from partner %s: %w", partner, err)
	}

	// Create metadata from partner room info
	metadata := make(map[string]any)
	for key, value := range partnerRoomInfo.Metadata {
		metadata[key] = value
	}
	ts := time.Now()

	room = &models.Room{
		Source: models.RoomPartner{
			RoomID: roomID,
			Name:   string(partner),
		},
		Name:          partnerRoomInfo.Name,
		Context:       partnerRoomInfo.Context,
		LastMessageAt: &ts,
		Metadata:      metadata,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := uc.roomRepo.Create(ctx, room); err != nil {
		return nil, fmt.Errorf("failed to create room: %w", err)
	}

	// Add room members - map partner user IDs to internal user IDs
	for _, participant := range partnerRoomInfo.Participants {
		// Find or sync the user to get internal user ID
		internalUser, err := uc.findOrSyncUser(ctx, participant.UserID, partner)
		if err != nil {
			log.Warnw(ctx, "Failed to sync participant user, skipping", "error", err, "partner_user_id", participant.UserID)
			continue
		}

		member := &models.RoomMember{
			RoomID:   room.ID,
			UserID:   internalUser.ID, // Use internal user ID
			Role:     participant.Role,
			JoinedAt: time.Now(),
		}

		if err := uc.roomMemberRepo.Create(ctx, member); err != nil {
			log.Warnw(ctx, "Failed to create room member", "error", err, "user_id", internalUser.ID.Hex())
		}
	}

	return room, nil
}

func (uc *ChatUseCase) incrementUnreadCountForOthers(ctx context.Context, roomID primitive.ObjectID, senderID primitive.ObjectID) {
	members, err := uc.roomMemberRepo.GetRoomMembers(ctx, roomID)
	if err != nil {
		fmt.Printf("Failed to get room members for unread count: %v\n", err)
		return
	}

	for _, member := range members {
		if member.UserID != senderID {
			if err := uc.unreadCountRepo.IncrementUnreadCount(ctx, roomID, member.UserID); err != nil {
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
func (uc *ChatUseCase) isInternalUser(ctx context.Context, senderID string, partnerType partners.PartnerType) bool {
	switch partnerType {
	case partners.PartnerTypeChotot:
		// Try to find user by chotot_id attribute
		user, err := uc.getUserByChototID(ctx, senderID)
		if err != nil {
			// User not found in our system, so it's external
			return false
		}
		// Check if user is marked as internal
		return user.IsInternal
	default:
		// For future partners, add similar lookup logic
		return false
	}
}

// findOrSyncUser finds or syncs user information from partner
func (uc *ChatUseCase) findOrSyncUser(ctx context.Context, senderID string, partnerType partners.PartnerType) (*models.User, error) {
	switch partnerType {
	case partners.PartnerTypeChotot:
		// Try to find existing user by chotot_id attribute
		user, err := uc.getUserByChototID(ctx, senderID)
		if err == nil {
			return user, nil // User already exists
		}

		// User doesn't exist, sync from partner
		partnerInstance, err := uc.partnerRegistry.GetPartner(partnerType)
		if err != nil {
			return nil, fmt.Errorf("failed to get partner %s: %w", partnerType, err)
		}

		partnerUserInfo, err := partnerInstance.GetUserInfo(ctx, senderID)
		if err != nil {
			return nil, fmt.Errorf("failed to get user info from partner %s: %w", partnerType, err)
		}

		// Create user from partner info
		user = &models.User{
			Name:       partnerUserInfo.Name,
			Email:      partnerUserInfo.Email,
			IsActive:   partnerUserInfo.IsActive,
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

		log.Infow(ctx, "Synced new user from partner", "partner", partnerType, "user_id", senderID)
		return user, nil

	default:
		return nil, fmt.Errorf("user sync not implemented for partner %s", partnerType)
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
