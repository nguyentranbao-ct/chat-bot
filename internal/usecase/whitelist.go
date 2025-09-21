package usecase

import (
	"strings"

	"github.com/nguyentranbao-ct/chat-bot/internal/config"
)

type WhitelistService interface {
	IsSellerAllowed(sellerID string) bool
	GetWhitelistedSellers() []string
}

type whitelistService struct {
	allowedSellers map[string]bool
	whitelist      []string
}

// NewWhitelistService creates a new whitelist service
func NewWhitelistService(cfg *config.Config) WhitelistService {
	allowedSellers := make(map[string]bool)
	for _, sellerID := range cfg.Kafka.Whitelist {
		if sellerID = strings.TrimSpace(sellerID); sellerID != "" {
			allowedSellers[sellerID] = true
		}
	}

	return &whitelistService{
		allowedSellers: allowedSellers,
		whitelist:      cfg.Kafka.Whitelist,
	}
}

// IsSellerAllowed checks if a seller is in the whitelist
func (w *whitelistService) IsSellerAllowed(sellerID string) bool {
	if w.allowedSellers["all"] {
		return true
	}

	// If whitelist is empty, allow all sellers
	if len(w.allowedSellers) == 0 {
		return true
	}

	return w.allowedSellers[sellerID]
}

// GetWhitelistedSellers returns the list of whitelisted sellers
func (w *whitelistService) GetWhitelistedSellers() []string {
	return w.whitelist
}
