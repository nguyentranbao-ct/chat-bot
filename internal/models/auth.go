package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AuthToken represents JWT tokens for user authentication
type AuthToken struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id" validate:"required"`
	TokenHash string             `bson:"token_hash" json:"-"` // hashed JWT token
	ExpiresAt time.Time          `bson:"expires_at" json:"expires_at"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	IsRevoked bool               `bson:"is_revoked" json:"is_revoked"`
	UserAgent string             `bson:"user_agent" json:"user_agent"`
	IPAddress string             `bson:"ip_address" json:"ip_address"`
}

// LoginRequest represents the email login request
type LoginRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// LoginResponse represents the login response with JWT token
type LoginResponse struct {
	Token     string    `json:"token"`
	User      User      `json:"user"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ProfileUpdateRequest represents user profile update request
type ProfileUpdateRequest struct {
	Name string `json:"name"`
}

// JWTClaims represents the JWT token claims
type JWTClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Exp    int64  `json:"exp"`
	Iat    int64  `json:"iat"`
}
