package usecase

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/mongodb"
	"github.com/nguyentranbao-ct/chat-bot/pkg/models"
	"gopkg.in/yaml.v3"
)

//go:embed default_chat_modes.yaml
var defaultChatModesData []byte

func AutoMigrate(repo mongodb.ChatModeRepository) error {
	var defaultModes []models.ChatMode
	if err := yaml.Unmarshal(defaultChatModesData, &defaultModes); err != nil {
		return fmt.Errorf("failed to unmarshal default chat modes: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Debugw(ctx, "Loaded chat modes from YAML", "count", len(defaultModes))
	for i, mode := range defaultModes {
		log.Debugw(ctx, "Chat mode details",
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
		if err := repo.Upsert(ctx, &mode); err != nil {
			return fmt.Errorf("failed to upsert chat mode '%s': %w", mode.Name, err)
		}
	}
	return nil
}
