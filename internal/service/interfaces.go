package service

import "context"

// WhitelistService defines the interface for channel whitelist management
type WhitelistService interface {
	IsChannelAllowed(channelID string) bool
	GetWhitelistedChannels() []string
}

// ChatModeInitializer defines the interface for initializing default chat modes
type ChatModeInitializer interface {
	InitializeDefaultChatModes(ctx context.Context) error
}