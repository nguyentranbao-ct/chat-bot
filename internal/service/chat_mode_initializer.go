package service

import (
	"context"
	"encoding/json"
	_ "embed"
	"fmt"

	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repository"
)

//go:embed default_chat_modes.json
var defaultChatModesData []byte

type ChatModeInitializer struct {
	repo repository.ChatModeRepository
}

func NewChatModeInitializer(repo repository.ChatModeRepository) *ChatModeInitializer {
	return &ChatModeInitializer{
		repo: repo,
	}
}

func (s *ChatModeInitializer) InitializeDefaultChatModes(ctx context.Context) error {
	var defaultModes []models.ChatMode
	if err := json.Unmarshal(defaultChatModesData, &defaultModes); err != nil {
		return fmt.Errorf("failed to unmarshal default chat modes: %w", err)
	}

	for _, mode := range defaultModes {
		if err := s.repo.Upsert(ctx, &mode); err != nil {
			return fmt.Errorf("failed to upsert chat mode '%s': %w", mode.Name, err)
		}
	}

	return nil
}