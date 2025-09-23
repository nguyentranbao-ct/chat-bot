package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/mongodb"
)

type AuthUseCase struct {
	userRepo    mongodb.UserRepository
	tokenRepo   mongodb.AuthTokenRepository
	userUsecase UserUsecase
	jwtSecret   string
}

func NewAuthUseCase(userRepo mongodb.UserRepository, tokenRepo mongodb.AuthTokenRepository, userUsecase UserUsecase, jwtSecret string) *AuthUseCase {
	return &AuthUseCase{
		userRepo:    userRepo,
		tokenRepo:   tokenRepo,
		userUsecase: userUsecase,
		jwtSecret:   jwtSecret,
	}
}

func (uc *AuthUseCase) Login(ctx context.Context, req models.LoginRequest, userAgent, ipAddress string) (*models.LoginResponse, error) {
	// Find or create user by email
	user, err := uc.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			// Create new user if not exists
			user = &models.User{
				Email:      req.Email,
				IsActive:   true,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
				IsInternal: true,
			}
			if err := uc.userRepo.Create(ctx, user); err != nil {
				return nil, fmt.Errorf("failed to create user: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get user by email: %w", err)
		}
	}

	// Update last login time and ensure user is active
	now := time.Now()
	user.UpdatedAt = now
	user.IsActive = true // Ensure user is active on login
	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user login time: %w", err)
	}

	// Generate JWT token
	token, expiresAt, err := uc.generateJWT(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT: %w", err)
	}

	// Store token hash in database
	tokenHash := uc.hashToken(token)
	authToken := &models.AuthToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
		UserAgent: userAgent,
		IPAddress: ipAddress,
	}

	if err := uc.tokenRepo.Create(ctx, authToken); err != nil {
		return nil, fmt.Errorf("failed to store auth token: %w", err)
	}

	// Check if user has partner attributes
	hasPartnerAttributes, err := uc.userUsecase.HasPartnerAttributes(ctx, user.ID)
	if err != nil {
		// Log error but don't fail login
		hasPartnerAttributes = false
	}

	return &models.LoginResponse{
		Token:                token,
		User:                 *user,
		ExpiresAt:            expiresAt,
		HasPartnerAttributes: hasPartnerAttributes,
	}, nil
}

func (uc *AuthUseCase) ValidateToken(ctx context.Context, tokenString string) (*models.User, error) {
	// Parse and validate JWT
	claims, err := uc.parseJWT(tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Check if token exists and is not revoked
	tokenHash := uc.hashToken(tokenString)
	authToken, err := uc.tokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("token not found: %w", err)
	}

	if authToken.IsRevoked {
		return nil, errors.New("token has been revoked")
	}

	if authToken.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("token has expired")
	}

	// Get user
	userID, err := primitive.ObjectIDFromHex(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in token: %w", err)
	}

	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if !user.IsActive {
		return nil, errors.New("user account is deactivated")
	}

	return user, nil
}

func (uc *AuthUseCase) UpdateProfile(ctx context.Context, userID primitive.ObjectID, req models.ProfileUpdateRequest) (*models.User, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Update fields
	if req.Name != "" {
		user.Name = req.Name
	}

	user.UpdatedAt = time.Now()

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user profile: %w", err)
	}

	return user, nil
}

func (uc *AuthUseCase) RevokeToken(ctx context.Context, tokenString string) error {
	tokenHash := uc.hashToken(tokenString)
	return uc.tokenRepo.RevokeToken(ctx, tokenHash)
}

func (uc *AuthUseCase) CleanupExpiredTokens(ctx context.Context) error {
	return uc.tokenRepo.DeleteExpiredTokens(ctx)
}

func (uc *AuthUseCase) generateJWT(user *models.User) (string, time.Time, error) {
	expiresAt := time.Now().Add(24 * time.Hour) // 24 hour expiry

	claims := jwt.MapClaims{
		"user_id": user.ID.Hex(),
		"email":   user.Email,
		"exp":     expiresAt.Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(uc.jwtSecret))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

func (uc *AuthUseCase) parseJWT(tokenString string) (*models.JWTClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(uc.jwtSecret), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return &models.JWTClaims{
			UserID: claims["user_id"].(string),
			Email:  claims["email"].(string),
			Exp:    int64(claims["exp"].(float64)),
			Iat:    int64(claims["iat"].(float64)),
		}, nil
	}

	return nil, errors.New("invalid token claims")
}

func (uc *AuthUseCase) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
