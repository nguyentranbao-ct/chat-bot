package llm

import (
	"context"
	"encoding/json"
	"fmt"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/nguyentranbao-ct/chat-bot/internal/client"
	"github.com/nguyentranbao-ct/chat-bot/internal/llm/tools"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SessionContext type alias for the tools package interface
type SessionContext = tools.SessionContext

// sessionContext is the concrete implementation of SessionContext
type sessionContext struct {
	sessionID            primitive.ObjectID
	channelID            string
	userID               string
	senderID             string
	ended                bool
	nextMessageTimestamp *int64
	sessionRepo          repository.ChatSessionRepository
}

type ToolsManager struct {
	triggerBuyTool   *tools.TriggerBuyTool
	replyMessageTool *tools.ReplyMessageTool
	fetchMessageTool *tools.FetchMessagesTool
	endSessionTool   *tools.EndSessionTool
	sessionRepo      repository.ChatSessionRepository
}

func NewToolsManager(
	chatAPIClient client.ChatAPIClient,
	sessionRepo repository.ChatSessionRepository,
	activityRepo repository.ChatActivityRepository,
	purchaseIntentRepo repository.PurchaseIntentRepository,
) *ToolsManager {
	return &ToolsManager{
		triggerBuyTool:   tools.NewTriggerBuyTool(chatAPIClient, activityRepo, purchaseIntentRepo),
		replyMessageTool: tools.NewReplyMessageTool(chatAPIClient, activityRepo),
		fetchMessageTool: tools.NewFetchMessagesTool(chatAPIClient, activityRepo),
		endSessionTool:   tools.NewEndSessionTool(activityRepo),
		sessionRepo:      sessionRepo,
	}
}

// SessionContextConfig holds configuration for creating a SessionContext
type SessionContextConfig struct {
	SessionID   primitive.ObjectID
	ChannelID   string
	UserID      string
	SenderID    string
	SessionRepo repository.ChatSessionRepository
}

// NewSessionContext creates a new SessionContext instance
func NewSessionContext(config SessionContextConfig) SessionContext {
	return &sessionContext{
		sessionID:   config.SessionID,
		channelID:   config.ChannelID,
		userID:      config.UserID,
		senderID:    config.SenderID,
		ended:       false,
		sessionRepo: config.SessionRepo,
	}
}

// Getter methods
func (s *sessionContext) GetSessionID() primitive.ObjectID {
	return s.sessionID
}

func (s *sessionContext) GetChannelID() string {
	return s.channelID
}

func (s *sessionContext) GetUserID() string {
	return s.userID
}

func (s *sessionContext) GetSenderID() string {
	return s.senderID
}

// Session control methods
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

func (s *sessionContext) IsEnded() bool {
	return s.ended
}

// Message timestamp tracking methods
func (s *sessionContext) GetNextMessageTimestamp() *int64 {
	return s.nextMessageTimestamp
}

func (s *sessionContext) SaveNextMessageTimestamp(timestamp int64) {
	s.nextMessageTimestamp = &timestamp
}

// Type aliases for tool arguments
type TriggerBuyArgs = tools.TriggerBuyArgs
type ReplyMessageArgs = tools.ReplyMessageArgs
type FetchMessagesArgs = tools.FetchMessagesArgs
type EndSessionArgs = tools.EndSessionArgs

func (tm *ToolsManager) triggerBuy(ctx context.Context, args TriggerBuyArgs, session SessionContext) error {
	return tm.triggerBuyTool.Execute(ctx, args, session)
}

func (tm *ToolsManager) replyMessage(ctx context.Context, args ReplyMessageArgs, session SessionContext) error {
	return tm.replyMessageTool.Execute(ctx, args, session)
}

func (tm *ToolsManager) fetchMessages(ctx context.Context, args FetchMessagesArgs, session SessionContext) (*models.MessageHistory, error) {
	return tm.fetchMessageTool.Execute(ctx, args, session)
}

func (tm *ToolsManager) endSession(ctx context.Context, args EndSessionArgs, session SessionContext) error {
	return tm.endSessionTool.Execute(ctx, args, session)
}

type ToolCall struct {
	Name string      `json:"name"`
	Args interface{} `json:"args"`
}

type ToolResult struct {
	Success bool        `json:"success"`
	Result  interface{} `json:"result,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func (tm *ToolsManager) ExecuteTool(ctx context.Context, toolCall ToolCall, session SessionContext) (*ToolResult, error) {
	switch toolCall.Name {
	case "TriggerBuy":
		var args TriggerBuyArgs
		if err := convertArgs(toolCall.Args, &args); err != nil {
			return &ToolResult{Success: false, Error: err.Error()}, nil
		}
		if err := tm.triggerBuy(ctx, args, session); err != nil {
			return &ToolResult{Success: false, Error: err.Error()}, nil
		}
		return &ToolResult{Success: true, Result: "Purchase intent logged successfully"}, nil

	case "ReplyMessage":
		var args ReplyMessageArgs
		if err := convertArgs(toolCall.Args, &args); err != nil {
			return &ToolResult{Success: false, Error: err.Error()}, nil
		}
		if err := tm.replyMessage(ctx, args, session); err != nil {
			return &ToolResult{Success: false, Error: err.Error()}, nil
		}
		return &ToolResult{Success: true, Result: "Message sent successfully"}, nil

	case "FetchMessages":
		var args FetchMessagesArgs
		if err := convertArgs(toolCall.Args, &args); err != nil {
			return &ToolResult{Success: false, Error: err.Error()}, nil
		}
		history, err := tm.fetchMessages(ctx, args, session)
		if err != nil {
			return &ToolResult{Success: false, Error: err.Error()}, nil
		}
		return &ToolResult{Success: true, Result: history}, nil

	case "EndSession":
		var args EndSessionArgs
		if err := convertArgs(toolCall.Args, &args); err != nil {
			return &ToolResult{Success: false, Error: err.Error()}, nil
		}
		if err := tm.endSession(ctx, args, session); err != nil {
			return &ToolResult{Success: false, Error: err.Error()}, nil
		}
		return &ToolResult{Success: true, Result: "Session ended successfully"}, nil

	default:
		return &ToolResult{Success: false, Error: fmt.Sprintf("unknown tool: %s", toolCall.Name)}, nil
	}
}

func (tm *ToolsManager) GetAvailableTools() []string {
	return []string{"TriggerBuy", "ReplyMessage", "FetchMessages", "EndSession"}
}

func convertArgs(from interface{}, to interface{}) error {
	// Simple JSON marshal/unmarshal to convert between types
	data, err := json.Marshal(from)
	if err != nil {
		return fmt.Errorf("failed to marshal args: %w", err)
	}
	if err := json.Unmarshal(data, to); err != nil {
		return fmt.Errorf("failed to unmarshal args: %w", err)
	}
	return nil
}
