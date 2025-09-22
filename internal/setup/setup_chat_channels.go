package setup

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/mongodb"
	"gopkg.in/yaml.v3"
)

//go:embed data/default_channels.yaml
var defaultChannelsData []byte

func SetupChannels(userRepo mongodb.UserRepository, channelRepo mongodb.ChannelRepository, channelMemberRepo mongodb.ChannelMemberRepository, messageRepo mongodb.ChatMessageRepository) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Load and create default channels
	var defaultChannels []DefaultChannel
	if err := yaml.Unmarshal(defaultChannelsData, &defaultChannels); err != nil {
		return fmt.Errorf("failed to unmarshal default channels: %w", err)
	}

	log.Debugw(ctx, "Loaded channels from YAML", "count", len(defaultChannels))

	// Create channels if they don't exist
	for _, defaultChannel := range defaultChannels {
		// Use GetByVendorChannelID instead of deprecated GetByExternalChannelID for consistency
		existingChannel, err := channelRepo.GetByVendorChannelID(ctx, "chotot", defaultChannel.ExternalChannelID)
		if err != nil && existingChannel == nil {
			// Find owner user by email
			ownerUser, err := userRepo.GetByEmail(ctx, defaultChannel.OwnerEmail)
			if err != nil || ownerUser == nil {
				log.Warnw(ctx, "Owner user not found for channel", "owner_email", defaultChannel.OwnerEmail, "channel_id", defaultChannel.ExternalChannelID)
				continue
			}

			now := time.Now()

			// Create metadata from item name and price
			metadata := make(map[string]any)
			if defaultChannel.ItemName != "" {
				metadata["item_name"] = defaultChannel.ItemName
			}
			if defaultChannel.ItemPrice != "" {
				metadata["item_price"] = defaultChannel.ItemPrice
			}

			channel := &models.Channel{
				Vendor: models.ChannelVendor{
					ChannelID: defaultChannel.ExternalChannelID,
					Name:      "chotot", // Default vendor for demo channels
				},
				Name:       defaultChannel.Name,
				Context:    defaultChannel.Context,
				Metadata:   metadata,
				CreatedAt:  now,
				UpdatedAt:  now,
				IsArchived: false,
			}

			if err := channelRepo.Create(ctx, channel); err != nil {
				return fmt.Errorf("failed to create channel '%s': %w", defaultChannel.ExternalChannelID, err)
			}
			log.Infow(ctx, "Created default channel", "vendor_channel_id", defaultChannel.ExternalChannelID, "name", defaultChannel.Name)

			// Create channel member for the owner
			member := &models.ChannelMember{
				ChannelID: channel.ID,
				UserID:    ownerUser.ID, // Using email as user ID for simplicity
				Role:      "seller",
				JoinedAt:  now,
				IsActive:  true,
			}

			if err := channelMemberRepo.Create(ctx, member); err != nil {
				return fmt.Errorf("failed to create channel member for '%s': %w", defaultChannel.ExternalChannelID, err)
			}
			log.Infow(ctx, "Created channel member", "vendor_channel_id", defaultChannel.ExternalChannelID, "user_id", defaultChannel.OwnerEmail)
		} else {
			log.Debugw(ctx, "Channel already exists", "vendor_channel_id", defaultChannel.ExternalChannelID)
		}
	}
	return nil
}
