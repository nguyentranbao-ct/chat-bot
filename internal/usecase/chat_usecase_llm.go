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
	TriggerLLM(ctx context.Context, message *models.ChatMessage, roomMember *models.RoomMember) error
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

func (uc *llmUsecaseV2) TriggerLLM(ctx context.Context, message *models.ChatMessage, roomMember *models.RoomMember) error {
	ctx, cancel := util.NewTimeoutContext(ctx, 180*time.Second)
	defer cancel()

	log.Infof(ctx, "Processing LLM trigger for message from user %s in room %s", message.SenderID.Hex(), roomMember.Source.RoomID)

	user, err := uc.userRepo.GetByID(ctx, message.SenderID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user.IsInternal {
		log.Infof(ctx, "Skipping LLM processing - user %s is internal", message.SenderID.Hex())
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

	merchantID, buyerID, err := uc.findMerchantAndBuyer(ctx, roomMember.RoomID, message.SenderID)
	if err != nil {
		return fmt.Errorf("failed to find merchant and buyer: %w", err)
	}

	recentMessages, err := uc.fetchRecentMessages(ctx, roomMember.RoomID, 5)
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

func (uc *llmUsecaseV2) findMerchantAndBuyer(ctx context.Context, roomID primitive.ObjectID, messageSenderID primitive.ObjectID) (merchantID, buyerID primitive.ObjectID, err error) {
	members, err := uc.roomMemberRepo.GetRoomMembersByRoomID(ctx, roomID)
	if err != nil {
		return primitive.NilObjectID, primitive.NilObjectID, fmt.Errorf("failed to get room members: %w", err)
	}

	var internalUserID, externalUserID primitive.ObjectID
	for _, member := range members {
		user, err := uc.userRepo.GetByID(ctx, member.UserID)
		if err != nil {
			log.Warnf(ctx, "Failed to get user %s: %v", member.UserID.Hex(), err)
			continue
		}

		if user.IsInternal {
			internalUserID = member.UserID
		} else {
			externalUserID = member.UserID
		}
	}

	if internalUserID == primitive.NilObjectID || externalUserID == primitive.NilObjectID {
		return primitive.NilObjectID, primitive.NilObjectID, fmt.Errorf("could not find both internal and external users in room")
	}

	return internalUserID, externalUserID, nil
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
