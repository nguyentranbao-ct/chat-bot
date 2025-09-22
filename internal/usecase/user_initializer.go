package usecase

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/mongodb"
	"gopkg.in/yaml.v3"
)

//go:embed default_users.yaml
var defaultUsersData []byte

//go:embed default_user_attributes.yaml
var defaultUserAttributesData []byte

//go:embed default_channels.yaml
var defaultChannelsData []byte

type DefaultUser struct {
	Name  string `yaml:"name"`
	Email string `yaml:"email"`
}

type DefaultUserAttribute struct {
	UserEmail string   `yaml:"user_email"`
	Key       string   `yaml:"key"`
	Value     string   `yaml:"value"`
	Tags      []string `yaml:"tags"`
}

type DefaultChannel struct {
	ExternalChannelID string `yaml:"external_channel_id"`
	Name              string `yaml:"name"`
	ItemName          string `yaml:"item_name"`
	ItemPrice         string `yaml:"item_price"`
	Context           string `yaml:"context"`
	Type              string `yaml:"type"`
	OwnerEmail        string `yaml:"owner_email"`
}

type DefaultMessage struct {
	ExternalChannelID string `yaml:"external_channel_id"`
	SenderID          string `yaml:"sender_id"`
	Content           string `yaml:"content"`
	MessageType       string `yaml:"message_type"`
	IsFromBot         bool   `yaml:"is_from_bot"`
}

func AutoMigrateUsers(userRepo mongodb.UserRepository, userAttrRepo mongodb.UserAttributeRepository) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Load and create default users
	var defaultUsers []DefaultUser
	if err := yaml.Unmarshal(defaultUsersData, &defaultUsers); err != nil {
		return fmt.Errorf("failed to unmarshal default users: %w", err)
	}

	log.Debugw(ctx, "Loaded users from YAML", "count", len(defaultUsers))
	for i, user := range defaultUsers {
		log.Debugw(ctx, "User details",
			"index", i,
			"name", user.Name,
			"email", user.Email,
		)
	}

	// Create users if they don't exist
	for _, defaultUser := range defaultUsers {
		existingUser, err := userRepo.GetByEmail(ctx, defaultUser.Email)
		if err != nil && !errors.Is(err, models.ErrNotFound) {
			return fmt.Errorf("failed to check existing user: %w", err)
		}

		if existingUser == nil {
			user := &models.User{
				Name:       defaultUser.Name,
				Email:      defaultUser.Email,
				IsInternal: true,
			}
			if err := userRepo.Create(ctx, user); err != nil {
				return fmt.Errorf("failed to create user '%s': %w", defaultUser.Email, err)
			}
			log.Infow(ctx, "Created default user", "email", defaultUser.Email)
		} else {
			log.Debugw(ctx, "User already exists", "email", defaultUser.Email)
		}
	}

	// Load and create default user attributes
	var defaultUserAttrs []DefaultUserAttribute
	if err := yaml.Unmarshal(defaultUserAttributesData, &defaultUserAttrs); err != nil {
		return fmt.Errorf("failed to unmarshal default user attributes: %w", err)
	}

	log.Debugw(ctx, "Loaded user attributes from YAML", "count", len(defaultUserAttrs))
	for i, attr := range defaultUserAttrs {
		log.Debugw(ctx, "User attribute details",
			"index", i,
			"user_email", attr.UserEmail,
			"key", attr.Key,
			"value", attr.Value,
			"tags", attr.Tags,
		)
	}

	// Create user attributes
	for _, defaultAttr := range defaultUserAttrs {
		// Find user by email
		user, err := userRepo.GetByEmail(ctx, defaultAttr.UserEmail)
		if err != nil {
			return fmt.Errorf("failed to find user for attribute: %w", err)
		}
		if user == nil {
			log.Warnw(ctx, "User not found for attribute", "user_email", defaultAttr.UserEmail, "key", defaultAttr.Key)
			continue
		}

		// Upsert user attribute
		attr := &models.UserAttribute{
			UserID: user.ID,
			Key:    defaultAttr.Key,
			Value:  defaultAttr.Value,
			Tags:   defaultAttr.Tags,
		}
		if err := userAttrRepo.Upsert(ctx, attr); err != nil {
			return fmt.Errorf("failed to upsert user attribute '%s' for user '%s': %w", defaultAttr.Key, defaultAttr.UserEmail, err)
		}
		log.Debugw(ctx, "Created/updated default user attribute",
			"user_email", defaultAttr.UserEmail,
			"key", defaultAttr.Key,
			"value", defaultAttr.Value)
	}

	return nil
}

func AutoMigrateChannels(userRepo mongodb.UserRepository, channelRepo mongodb.ChannelRepository, channelMemberRepo mongodb.ChannelMemberRepository, messageRepo mongodb.ChatMessageRepository) error {
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
