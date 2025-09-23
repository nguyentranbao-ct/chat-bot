package usecase

import (
	"context"
	"fmt"
	"regexp"
	"slices"

	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/mongodb"
	"github.com/nguyentranbao-ct/chat-bot/pkg/crypto"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserUsecase interface {
	CreateUser(ctx context.Context, name, email string) (*models.User, error)
	GetUser(ctx context.Context, id primitive.ObjectID) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
	DeleteUser(ctx context.Context, id primitive.ObjectID) error

	SetUserAttribute(ctx context.Context, userID primitive.ObjectID, key, value string, tags []string) error
	GetUserAttributes(ctx context.Context, userID primitive.ObjectID) ([]*models.UserAttribute, error)
	GetUserAttributeByKey(ctx context.Context, userID primitive.ObjectID, key string) (*models.UserAttribute, error)
	GetUsersByTag(ctx context.Context, tags []string) ([]*models.User, error)
	GetUserByChototID(ctx context.Context, chototID string) (*models.User, error)
	RemoveUserAttribute(ctx context.Context, userID primitive.ObjectID, key string) error

	GetPartnerAttributes(ctx context.Context, userID primitive.ObjectID) (*models.PartnerAttributesResponse, error)
	UpdatePartnerAttributes(ctx context.Context, userID primitive.ObjectID, req *models.PartnerAttributesRequest) error
	HasPartnerAttributes(ctx context.Context, userID primitive.ObjectID) (bool, error)
}

type userUsecase struct {
	userRepo          mongodb.UserRepository
	userAttributeRepo mongodb.UserAttributeRepository
	cryptoClient      crypto.Client
}

func NewUserUsecase(
	userRepo mongodb.UserRepository,
	userAttributeRepo mongodb.UserAttributeRepository,
	cryptoClient crypto.Client,
) UserUsecase {
	return &userUsecase{
		userRepo:          userRepo,
		userAttributeRepo: userAttributeRepo,
		cryptoClient:      cryptoClient,
	}
}

func (uc *userUsecase) CreateUser(ctx context.Context, name, email string) (*models.User, error) {
	// Check if user with email already exists
	existingUser, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, fmt.Errorf("user with email %s already exists", email)
	}

	user := &models.User{
		Name:  name,
		Email: email,
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (uc *userUsecase) GetUser(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	user, err := uc.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

func (uc *userUsecase) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return user, nil
}

func (uc *userUsecase) UpdateUser(ctx context.Context, user *models.User) error {
	if err := uc.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

func (uc *userUsecase) DeleteUser(ctx context.Context, id primitive.ObjectID) error {
	if err := uc.userRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

func (uc *userUsecase) SetUserAttribute(ctx context.Context, userID primitive.ObjectID, key, value string, tags []string) error {
	// Validate key format (alpha-numeric with underscores)
	if !isValidAttributeKey(key) {
		return fmt.Errorf("invalid attribute key format: %s (must be alpha-numeric with underscores)", key)
	}

	attr := &models.UserAttribute{
		UserID: userID,
		Key:    key,
		Value:  value,
		Tags:   tags,
	}

	if err := uc.userAttributeRepo.Upsert(ctx, attr); err != nil {
		return fmt.Errorf("failed to set user attribute: %w", err)
	}

	return nil
}

func (uc *userUsecase) GetUserAttributes(ctx context.Context, userID primitive.ObjectID) ([]*models.UserAttribute, error) {
	attrs, err := uc.userAttributeRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user attributes: %w", err)
	}
	return attrs, nil
}

func (uc *userUsecase) GetUserAttributeByKey(ctx context.Context, userID primitive.ObjectID, key string) (*models.UserAttribute, error) {
	attr, err := uc.userAttributeRepo.GetByUserIDAndKey(ctx, userID, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get user attribute: %w", err)
	}
	return attr, nil
}

func (uc *userUsecase) GetUsersByTag(ctx context.Context, tags []string) ([]*models.User, error) {
	// Get user attributes that have any of the specified tags
	attrs, err := uc.userAttributeRepo.GetByTags(ctx, tags)
	if err != nil {
		return nil, fmt.Errorf("failed to get user attributes by tags: %w", err)
	}

	// Extract unique user IDs
	userIDMap := make(map[primitive.ObjectID]bool)
	for _, attr := range attrs {
		userIDMap[attr.UserID] = true
	}

	// Get users by IDs
	var users []*models.User
	for userID := range userIDMap {
		user, err := uc.userRepo.GetByID(ctx, userID)
		if err != nil {
			continue // Skip users that can't be retrieved
		}
		users = append(users, user)
	}

	return users, nil
}

func (uc *userUsecase) GetUserByChototID(ctx context.Context, chototID string) (*models.User, error) {
	// Find user attribute with chotot_id key and matching value
	attrs, err := uc.userAttributeRepo.GetByKey(ctx, "chotot_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get chotot attributes: %w", err)
	}

	for _, attr := range attrs {
		if attr.Value == chototID {
			user, err := uc.userRepo.GetByID(ctx, attr.UserID)
			if err != nil {
				continue // Skip if user can't be retrieved
			}
			return user, nil
		}
	}

	return nil, fmt.Errorf("user with chotot ID %s not found", chototID)
}

func (uc *userUsecase) RemoveUserAttribute(ctx context.Context, userID primitive.ObjectID, key string) error {
	if err := uc.userAttributeRepo.DeleteByUserIDAndKey(ctx, userID, key); err != nil {
		return fmt.Errorf("failed to remove user attribute: %w", err)
	}
	return nil
}

func (uc *userUsecase) GetPartnerAttributes(ctx context.Context, userID primitive.ObjectID) (*models.PartnerAttributesResponse, error) {
	attrs, err := uc.userAttributeRepo.GetByUserIDAndTags(ctx, userID, []string{models.TagChotot, models.TagWhatsApp})
	if err != nil {
		return nil, fmt.Errorf("failed to get partner attributes: %w", err)
	}

	response := &models.PartnerAttributesResponse{}

	for _, attr := range attrs {
		// Skip sensitive attributes in response
		if slices.Contains(attr.Tags, models.TagSensitive) {
			continue
		}

		switch attr.Key {
		case models.PartnerAttrChototID:
			response.ChototID = attr.Value
		case models.PartnerAttrChototOID:
			response.ChototOID = attr.Value
		case models.PartnerAttrWhatsAppPhoneNumberID:
			response.WhatsAppPhoneNumberID = attr.Value
		}
	}

	return response, nil
}

func (uc *userUsecase) UpdatePartnerAttributes(ctx context.Context, userID primitive.ObjectID, req *models.PartnerAttributesRequest) error {
	// Update Chotot attributes
	if req.ChototID != "" {
		if err := uc.SetUserAttribute(ctx, userID, models.PartnerAttrChototID, req.ChototID, []string{models.TagChotot}); err != nil {
			return fmt.Errorf("failed to set chotot_id: %w", err)
		}
	}

	if req.ChototOID != "" {
		if err := uc.SetUserAttribute(ctx, userID, models.PartnerAttrChototOID, req.ChototOID, []string{models.TagChotot}); err != nil {
			return fmt.Errorf("failed to set chotot_oid: %w", err)
		}
	}

	// Update WhatsApp attributes
	if req.WhatsAppPhoneNumberID != "" {
		if err := uc.SetUserAttribute(ctx, userID, models.PartnerAttrWhatsAppPhoneNumberID, req.WhatsAppPhoneNumberID, []string{models.TagWhatsApp}); err != nil {
			return fmt.Errorf("failed to set whatsapp_phone_number_id: %w", err)
		}
	}

	if req.WhatsAppSystemToken != "" {
		// Encrypt sensitive token before storing
		encryptedToken, err := uc.cryptoClient.Encrypt(req.WhatsAppSystemToken)
		if err != nil {
			return fmt.Errorf("failed to encrypt whatsapp_system_token: %w", err)
		}

		if err := uc.SetUserAttribute(ctx, userID, models.PartnerAttrWhatsAppSystemToken, encryptedToken, []string{models.TagWhatsApp, models.TagSensitive}); err != nil {
			return fmt.Errorf("failed to set whatsapp_system_token: %w", err)
		}
	}

	return nil
}

func (uc *userUsecase) HasPartnerAttributes(ctx context.Context, userID primitive.ObjectID) (bool, error) {
	attrs, err := uc.userAttributeRepo.GetByUserIDAndTags(ctx, userID, []string{models.TagChotot, models.TagWhatsApp})
	if err != nil {
		return false, fmt.Errorf("failed to check partner attributes: %w", err)
	}

	return len(attrs) > 0, nil
}

// isValidAttributeKey validates that the key contains only alpha-numeric characters and underscores
func isValidAttributeKey(key string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]+$`, key)
	return matched
}
