package usecase

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nguyentranbao-ct/chat-bot/internal/client"
	"github.com/nguyentranbao-ct/chat-bot/internal/llm"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repository"
)

type messageUsecase struct {
	chatModeRepo  repository.ChatModeRepository
	sessionRepo   repository.ChatSessionRepository
	activityRepo  repository.ChatActivityRepository
	chatAPIClient client.ChatAPIClient
	genkitService *llm.GenkitService
}

func NewMessageUsecase(
	chatModeRepo repository.ChatModeRepository,
	sessionRepo repository.ChatSessionRepository,
	activityRepo repository.ChatActivityRepository,
	chatAPIClient client.ChatAPIClient,
	genkitService *llm.GenkitService,
) MessageUsecase {
	return &messageUsecase{
		chatModeRepo:  chatModeRepo,
		sessionRepo:   sessionRepo,
		activityRepo:  activityRepo,
		chatAPIClient: chatAPIClient,
		genkitService: genkitService,
	}
}

func (uc *messageUsecase) ProcessMessage(ctx context.Context, message *models.IncomingMessage) error {
	log.Printf("Processing message from user %s in channel %s", message.SenderID, message.ChannelID)

	chatMode, err := uc.chatModeRepo.GetByName(ctx, message.Metadata.LLM.ChatMode)
	if err != nil {
		return fmt.Errorf("failed to get chat mode '%s': %w", message.Metadata.LLM.ChatMode, err)
	}

	session, err := uc.getOrCreateSession(ctx, message, chatMode)
	if err != nil {
		return fmt.Errorf("failed to get or create session: %w", err)
	}

	channelInfo, err := uc.chatAPIClient.GetChannelInfo(ctx, message.ChannelID)
	if err != nil {
		return fmt.Errorf("failed to get channel info: %w", err)
	}

	promptData := &llm.PromptData{
		ChannelInfo: channelInfo,
		SessionID:   session.ID.Hex(),
		UserID:      message.SenderID,
		Message:     message.Message,
	}

	if err := uc.genkitService.ProcessMessage(ctx, chatMode, promptData); err != nil {
		return fmt.Errorf("failed to process with Genkit: %w", err)
	}

	log.Printf("Successfully processed message for session %s", session.ID.Hex())
	return nil
}

func (uc *messageUsecase) getOrCreateSession(ctx context.Context, message *models.IncomingMessage, chatMode *models.ChatMode) (*models.ChatSession, error) {
	session, err := uc.sessionRepo.GetByChannelAndUser(ctx, message.ChannelID, message.SenderID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing session: %w", err)
	}

	if session != nil {
		log.Printf("Using existing session %s", session.ID.Hex())
		return session, nil
	}

	session = &models.ChatSession{
		ChannelID: message.ChannelID,
		UserID:    message.SenderID,
		ChatMode:  chatMode.Name,
		Status:    models.SessionStatusActive,
		StartedAt: time.Now(),
	}

	if err := uc.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	log.Printf("Created new session %s for user %s in channel %s", session.ID.Hex(), message.SenderID, message.ChannelID)
	return session, nil
}
