package usecase

import (
	"context"
	"fmt"
	"time"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/chatapi"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/mongodb"
)

type MessageUsecase interface {
	ProcessMessage(ctx context.Context, message models.IncomingMessage) error
}

type messageUsecase struct {
	chatModeRepo     mongodb.ChatModeRepository
	sessionRepo      mongodb.ChatSessionRepository
	activityRepo     mongodb.ChatActivityRepository
	chatAPIClient    chatapi.Client
	llmUsecase       LLMUsecase
	whitelistService WhitelistService
}

func NewMessageUsecase(
	chatModeRepo mongodb.ChatModeRepository,
	sessionRepo mongodb.ChatSessionRepository,
	activityRepo mongodb.ChatActivityRepository,
	chatAPIClient chatapi.Client,
	llmUsecase LLMUsecase,
	whitelistService WhitelistService,
) MessageUsecase {
	return &messageUsecase{
		chatModeRepo:     chatModeRepo,
		sessionRepo:      sessionRepo,
		activityRepo:     activityRepo,
		chatAPIClient:    chatAPIClient,
		llmUsecase:       llmUsecase,
		whitelistService: whitelistService,
	}
}

func (uc *messageUsecase) ProcessMessage(ctx context.Context, message models.IncomingMessage) error {
	log.Infof(ctx, "Processing message from user %s in channel %s", message.SenderID, message.ChannelID)

	// Get channel info first to check seller whitelist
	channelInfo, err := uc.chatAPIClient.GetChannelInfo(ctx, message.ChannelID)
	if err != nil {
		return fmt.Errorf("failed to get channel info: %w", err)
	}

	// Find seller ID from channel participants and check whitelist
	sellerID := findSellerIDFromChannel(channelInfo)
	if sellerID == "" {
		log.Infof(ctx, "No seller found in channel %s, skipping message", message.ChannelID)
		return nil // Skip message if no seller found
	}

	// Check if seller is whitelisted
	if !uc.whitelistService.IsSellerAllowed(sellerID) {
		log.Infof(ctx, "Ignoring message from non-whitelisted seller %s in channel %s", sellerID, message.ChannelID)
		return nil // Skip message if seller not whitelisted
	}

	chatMode, err := uc.chatModeRepo.GetByName(ctx, message.Metadata.LLM.ChatMode)
	if err != nil {
		return fmt.Errorf("failed to get chat mode '%s': %w", message.Metadata.LLM.ChatMode, err)
	}

	session, err := uc.newSession(ctx, message, chatMode)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Find sender role from channel participants
	senderRole := findSenderRole(channelInfo, message.SenderID)

	// Fetch 20 recent messages for context
	recentMessages, err := uc.fetchRecentMessages(ctx, message.SenderID, message.ChannelID)
	if err != nil {
		log.Errorf(ctx, "Failed to fetch recent messages: %v", err)
		// Continue without recent messages rather than failing
		recentMessages = &models.MessageHistory{Messages: []models.HistoryMessage{}}
	}

	promptData := &PromptData{
		ChannelInfo:    channelInfo,
		SessionID:      session.ID.Hex(),
		UserID:         message.SenderID,
		SenderRole:     senderRole,
		Message:        message.Message,
		RecentMessages: recentMessages,
	}

	if err := uc.llmUsecase.ProcessMessage(ctx, chatMode, promptData); err != nil {
		return fmt.Errorf("failed to process with Genkit: %w", err)
	}

	log.Infof(ctx, "Successfully processed message for session %s", session.ID.Hex())
	return nil
}

func (uc *messageUsecase) newSession(ctx context.Context, message models.IncomingMessage, chatMode *models.ChatMode) (*models.ChatSession, error) {
	session := &models.ChatSession{
		ChannelID: message.ChannelID,
		UserID:    message.SenderID,
		ChatMode:  chatMode.Name,
		Status:    models.SessionStatusActive,
		StartedAt: time.Now(),
	}

	if err := uc.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	log.Infof(ctx, "Created new session %s for user %s in channel %s", session.ID.Hex(), message.SenderID, message.ChannelID)
	return session, nil
}

// findSenderRole finds the role of the sender from channel participants
func findSenderRole(channelInfo *models.ChannelInfo, senderID string) string {
	if channelInfo == nil || len(channelInfo.Participants) == 0 {
		return "unknown"
	}

	for _, participant := range channelInfo.Participants {
		if participant.UserID == senderID {
			return participant.Role
		}
	}
	return "unknown"
}

// findSellerIDFromChannel finds the seller ID from channel participants
func findSellerIDFromChannel(channelInfo *models.ChannelInfo) string {
	if channelInfo == nil || len(channelInfo.Participants) == 0 {
		return ""
	}

	for _, participant := range channelInfo.Participants {
		if participant.Role == "seller" {
			return participant.UserID
		}
	}
	return ""
}

func (uc *messageUsecase) fetchRecentMessages(ctx context.Context, userID, channelID string) (*models.MessageHistory, error) {
	req := chatapi.MessageHistoryRequest{
		UserID:    userID,
		ChannelID: channelID,
		Limit:     20,
		BeforeTs:  nil,
	}
	return uc.chatAPIClient.GetMessageHistoryWithParams(ctx, req)
}
