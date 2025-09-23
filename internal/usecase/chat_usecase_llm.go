package usecase

import (
	"context"
	"fmt"
	"time"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/mongodb"
	"github.com/nguyentranbao-ct/chat-bot/pkg/util"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LLMUsecaseV2 interface {
	TriggerLLM(ctx context.Context, message *models.ChatMessage, roomMembers []*models.RoomMember) error
}

type llmUsecaseV2 struct {
	userRepo          mongodb.UserRepository
	userAttributeRepo mongodb.UserAttributeRepository
	chatModeRepo      mongodb.ChatModeRepository
	sessionRepo       mongodb.ChatSessionRepository
	roomMemberRepo    mongodb.RoomMemberRepository
	messageRepo       mongodb.ChatMessageRepository
	llmUsecase        LLMUsecase
}

func NewLLMUsecaseV2(
	userRepo mongodb.UserRepository,
	userAttributeRepo mongodb.UserAttributeRepository,
	chatModeRepo mongodb.ChatModeRepository,
	sessionRepo mongodb.ChatSessionRepository,
	roomMemberRepo mongodb.RoomMemberRepository,
	messageRepo mongodb.ChatMessageRepository,
	llmUsecase LLMUsecase,
) LLMUsecaseV2 {
	return &llmUsecaseV2{
		userRepo:          userRepo,
		userAttributeRepo: userAttributeRepo,
		chatModeRepo:      chatModeRepo,
		sessionRepo:       sessionRepo,
		roomMemberRepo:    roomMemberRepo,
		messageRepo:       messageRepo,
		llmUsecase:        llmUsecase,
	}
}

func (uc *llmUsecaseV2) TriggerLLM(ctx context.Context, message *models.ChatMessage, roomMembers []*models.RoomMember) error {
	ctx, cancel := util.NewTimeoutContext(ctx, 180*time.Second)
	defer cancel()

	// Find the room member that contains the room info (first one should have the source info)
	if len(roomMembers) == 0 {
		return fmt.Errorf("no room members provided")
	}

	roomMember := roomMembers[0]

	merchantID, buyerID, err := uc.findMerchantAndBuyer(ctx, roomMembers, message.SenderID)
	if err != nil {
		return fmt.Errorf("failed to find merchant and buyer: %w", err)
	}
	if merchantID == message.SenderID {
		return nil
	}

	chatModeKey := fmt.Sprintf("%s_chat_mode", roomMember.Source.Name)
	chatMode, err := uc.getChatModeForUser(ctx, message.SenderID, chatModeKey)
	if err != nil {
		return fmt.Errorf("failed to get chat mode for user %s: %w", message.SenderID.Hex(), err)
	}

	session, err := uc.createSession(ctx, message, roomMember, chatMode)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	recentMessages, err := uc.fetchRecentMessages(ctx, roomMember.RoomID, 3)
	if err != nil {
		log.Warnf(ctx, "Failed to fetch recent messages: %v", err)
		recentMessages = &models.MessageHistory{Messages: []models.HistoryMessage{}}
	}

	promptContext := &PromptContext{
		RoomMember:     roomMember,
		SessionID:      session.ID,
		Message:        message.Content,
		RecentMessages: recentMessages,
		MerchantID:     merchantID,
		BuyerID:        buyerID,
	}

	if err := uc.llmUsecase.ProcessMessage(ctx, chatMode, promptContext); err != nil {
		return fmt.Errorf("failed to process with LLM: %w", err)
	}

	log.Infof(ctx, "Successfully processed LLM message for session %s", session.ID.Hex())
	return nil
}

func (uc *llmUsecaseV2) getChatModeForUser(ctx context.Context, userID primitive.ObjectID, chatModeKey string) (*models.ChatMode, error) {
	attr, err := uc.userAttributeRepo.GetByUserIDAndKey(ctx, userID, chatModeKey)
	if err != nil && err != models.ErrNotFound {
		return nil, fmt.Errorf("failed to get user attribute %s: %w", chatModeKey, err)
	}
	mode := "sales_assistant"
	if attr != nil && attr.Value != "" {
		mode = attr.Value
	}

	chatMode, err := uc.chatModeRepo.GetByName(ctx, mode)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat mode '%s': %w", attr.Value, err)
	}

	return chatMode, nil
}

func (uc *llmUsecaseV2) findMerchantAndBuyer(ctx context.Context, roomMembers []*models.RoomMember, messageSenderID primitive.ObjectID) (merchantID, buyerID primitive.ObjectID, err error) {
	if len(roomMembers) == 0 {
		return primitive.NilObjectID, primitive.NilObjectID, fmt.Errorf("no room members provided")
	}
	log.Infow(ctx, "room members", "count", len(roomMembers), "members", roomMembers)
	for _, member := range roomMembers {
		if member.Role == "merchant" || member.Role == "seller" {
			merchantID = member.UserID
		} else {
			buyerID = member.UserID
		}
	}
	if merchantID == primitive.NilObjectID || buyerID == primitive.NilObjectID {
		return primitive.NilObjectID, primitive.NilObjectID, fmt.Errorf("could not find both merchant and buyer users in room")
	}
	return merchantID, buyerID, nil
}

func (uc *llmUsecaseV2) fetchRecentMessages(ctx context.Context, roomID primitive.ObjectID, limit int) (*models.MessageHistory, error) {
	messages, err := uc.messageRepo.GetRoomMessages(ctx, roomID, limit, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages from database: %w", err)
	}

	history := &models.MessageHistory{
		Messages: make([]models.HistoryMessage, len(messages)),
		HasMore:  len(messages) == limit,
	}

	for i, msg := range messages {
		history.Messages[i] = models.HistoryMessage{
			ID:        msg.ID.Hex(),
			RoomID:    msg.RoomID,
			SenderID:  msg.SenderID,
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt,
		}
	}

	return history, nil
}

func (uc *llmUsecaseV2) createSession(ctx context.Context, message *models.ChatMessage, roomMember *models.RoomMember, chatMode *models.ChatMode) (*models.ChatSession, error) {
	session := &models.ChatSession{
		RoomID:   roomMember.RoomID,
		ChatMode: chatMode.ID,
		Status:   models.SessionStatusActive,
	}

	if err := uc.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	log.Infof(ctx, "Created new session %s for user %s in room %s", session.ID.Hex(), message.SenderID.Hex(), roomMember.Source.RoomID)
	return session, nil
}
