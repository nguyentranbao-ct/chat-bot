package usecase

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

//go:embed default_users.yaml
var defaultUsersData []byte

//go:embed default_user_attributes.yaml
var defaultUserAttributesData []byte

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
		if err != nil {
			return fmt.Errorf("failed to check existing user: %w", err)
		}

		if existingUser == nil {
			user := &models.User{
				Name:  defaultUser.Name,
				Email: defaultUser.Email,
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
		log.Infow(ctx, "Created/updated default user attribute",
			"user_email", defaultAttr.UserEmail,
			"key", defaultAttr.Key,
			"value", defaultAttr.Value)
	}

	return nil
}