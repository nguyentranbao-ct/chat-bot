package service

import (
	"strings"

	"github.com/nguyentranbao-ct/chat-bot/internal/config"
)

type whitelistService struct {
	allowedChannels map[string]bool
	whitelist       []string
}

// NewWhitelistService creates a new whitelist service
func NewWhitelistService(cfg *config.KafkaConfig) WhitelistService {
	allowedChannels := make(map[string]bool)
	for _, channelID := range cfg.Whitelist {
		if channelID = strings.TrimSpace(channelID); channelID != "" {
			allowedChannels[channelID] = true
		}
	}

	return &whitelistService{
		allowedChannels: allowedChannels,
		whitelist:       cfg.Whitelist,
	}
}

// IsChannelAllowed checks if a channel is in the whitelist
func (w *whitelistService) IsChannelAllowed(channelID string) bool {
	// If whitelist is empty, allow all channels
	if len(w.allowedChannels) == 0 {
		return true
	}

	return w.allowedChannels[channelID]
}

// GetWhitelistedChannels returns the list of whitelisted channels
func (w *whitelistService) GetWhitelistedChannels() []string {
	return w.whitelist
}