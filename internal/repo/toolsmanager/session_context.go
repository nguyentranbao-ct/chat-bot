package toolsmanager

import (
	"context"
	"fmt"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/firebase/genkit/go/genkit"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/mongodb"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// sessionContext implements the SessionContext interface
type sessionContext struct {
	ctx                  context.Context
	config               SessionContextConfig
	ended                bool
	nextMessageTimestamp *int64
	sessionRepo          mongodb.ChatSessionRepository
}

// SessionContextConfig holds configuration for creating a SessionContext
type SessionContextConfig struct {
	Genkit      *genkit.Genkit
	SessionID   primitive.ObjectID
	ChannelID   primitive.ObjectID
	BuyerID     primitive.ObjectID
	MerchantID  primitive.ObjectID
	SessionRepo mongodb.ChatSessionRepository
}

// NewSessionContext creates a new SessionContext instance
func NewSessionContext(ctx context.Context, config SessionContextConfig) SessionContext {
	return &sessionContext{
		ctx:         ctx,
		config:      config,
		ended:       false,
		sessionRepo: config.SessionRepo,
	}
}

func (s *sessionContext) Context() context.Context {
	return s.ctx
}

func (s *sessionContext) Genkit() *genkit.Genkit {
	return s.config.Genkit
}

// GetSessionID returns the session ID as a string
func (s *sessionContext) GetSessionID() primitive.ObjectID {
	return s.config.SessionID
}

// GetChannelID returns the channel ID
func (s *sessionContext) GetChannelID() primitive.ObjectID {
	return s.config.ChannelID
}

// GetBuyerID returns the user ID
func (s *sessionContext) GetBuyerID() primitive.ObjectID {
	return s.config.BuyerID
}

// GetMerchantID returns the sender ID
func (s *sessionContext) GetMerchantID() primitive.ObjectID {
	return s.config.MerchantID
}

// EndSession terminates the session
func (s *sessionContext) EndSession() error {
	if s.ended {
		return nil // Already ended
	}

	if err := s.sessionRepo.EndSession(s.ctx, s.config.SessionID); err != nil {
		return fmt.Errorf("failed to end session: %w", err)
	}

	s.ended = true
	log.Infof(s.ctx, "Session %s ended successfully", s.config.SessionID.Hex())
	return nil
}

// IsEnded returns whether the session has been terminated
func (s *sessionContext) IsEnded() bool {
	return s.ended
}

// GetNextMessageTimestamp returns the next message timestamp
func (s *sessionContext) GetNextMessageTimestamp() *int64 {
	return s.nextMessageTimestamp
}

// SaveNextMessageTimestamp saves the next message timestamp
func (s *sessionContext) SaveNextMessageTimestamp(timestamp int64) {
	s.nextMessageTimestamp = &timestamp
}
