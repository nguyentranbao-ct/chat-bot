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
	genkit               *genkit.Genkit
	sessionID            primitive.ObjectID
	channelID            string
	userID               string
	senderID             string
	ended                bool
	nextMessageTimestamp *int64
	sessionRepo          mongodb.ChatSessionRepository
}

// SessionContextConfig holds configuration for creating a SessionContext
type SessionContextConfig struct {
	Genkit      *genkit.Genkit
	SessionID   primitive.ObjectID
	ChannelID   string
	UserID      string
	SenderID    string
	SessionRepo mongodb.ChatSessionRepository
}

// NewSessionContext creates a new SessionContext instance
func NewSessionContext(ctx context.Context, config SessionContextConfig) SessionContext {
	return &sessionContext{
		ctx:         ctx,
		genkit:      config.Genkit,
		sessionID:   config.SessionID,
		channelID:   config.ChannelID,
		userID:      config.UserID,
		senderID:    config.SenderID,
		ended:       false,
		sessionRepo: config.SessionRepo,
	}
}

func (s *sessionContext) Context() context.Context {
	return s.ctx
}

func (s *sessionContext) Genkit() *genkit.Genkit {
	return s.genkit
}

// GetSessionID returns the session ID as a string
func (s *sessionContext) GetSessionID() string {
	return s.sessionID.Hex()
}

// GetChannelID returns the channel ID
func (s *sessionContext) GetChannelID() string {
	return s.channelID
}

// GetUserID returns the user ID
func (s *sessionContext) GetUserID() string {
	return s.userID
}

// GetSenderID returns the sender ID
func (s *sessionContext) GetSenderID() string {
	return s.senderID
}

// EndSession terminates the session
func (s *sessionContext) EndSession() error {
	if s.ended {
		return nil // Already ended
	}

	if err := s.sessionRepo.EndSession(context.Background(), s.sessionID); err != nil {
		return fmt.Errorf("failed to end session: %w", err)
	}

	s.ended = true
	log.Infof(context.Background(), "Session %s ended successfully", s.sessionID.Hex())
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
