package service

// WhitelistService defines the interface for channel whitelist management
type WhitelistService interface {
	IsChannelAllowed(channelID string) bool
	GetWhitelistedChannels() []string
}