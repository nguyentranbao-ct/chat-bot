package service

import "context"

// WhitelistService defines the interface for seller whitelist management
type WhitelistService interface {
	IsSellerAllowed(sellerID string) bool
	GetWhitelistedSellers() []string
}

// ChatModeInitializer defines the interface for initializing default chat modes
type ChatModeInitializer interface {
	InitializeDefaultChatModes(ctx context.Context) error
}