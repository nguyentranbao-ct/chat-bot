package service

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/carousell/ct-go/pkg/logger"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repository"
	"gopkg.in/yaml.v3"
)

//go:embed default_chat_modes.yaml
var defaultChatModesData []byte

type chatModeInitializer struct {
	repo repository.ChatModeRepository
}

func NewChatModeInitializer(repo repository.ChatModeRepository) ChatModeInitializer {
	return &chatModeInitializer{
		repo: repo,
	}
}

func (s *chatModeInitializer) InitializeDefaultChatModes(ctx context.Context) error {
	log := logger.MustNamed("chat_mode_initializer")

	var defaultModes []models.ChatMode
	if err := yaml.Unmarshal(defaultChatModesData, &defaultModes); err != nil {
		return fmt.Errorf("failed to unmarshal default chat modes: %w", err)
	}

	log.Debugw("Loaded chat modes from YAML", "count", len(defaultModes))
	for i, mode := range defaultModes {
		log.Debugw("Chat mode details",
			"index", i,
			"name", mode.Name,
			"model", mode.Model,
			"condition", mode.Condition,
			"max_iterations", mode.MaxIterations,
			"max_prompt_tokens", mode.MaxPromptTokens,
			"max_response_tokens", mode.MaxResponseTokens,
			"prompt_template_length", len(mode.PromptTemplate),
			"tools", mode.Tools,
		)
	}

	for _, mode := range defaultModes {
		if err := s.repo.Upsert(ctx, &mode); err != nil {
			return fmt.Errorf("failed to upsert chat mode '%s': %w", mode.Name, err)
		}
	}

	return nil
}
