package llm

import (
	"context"
	"encoding/json"
	"fmt"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/nguyentranbao-ct/chat-bot/internal/client"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SessionContext interface defines methods that tools can call during execution
type SessionContext interface {
	// Getters for session information
	GetSessionID() primitive.ObjectID
	GetChannelID() string
	GetUserID() string
	GetSenderID() string

	// Session control methods
	EndSession() error
	IsEnded() bool

	// Message timestamp tracking for pagination
	GetNextMessageTimestamp() *int64
	SaveNextMessageTimestamp(timestamp int64)
}

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
	chatAPIClient      client.ChatAPIClient
	sessionRepo        repository.ChatSessionRepository
	activityRepo       repository.ChatActivityRepository
	purchaseIntentRepo repository.PurchaseIntentRepository
}

func NewToolsManager(
	chatAPIClient client.ChatAPIClient,
	sessionRepo repository.ChatSessionRepository,
	activityRepo repository.ChatActivityRepository,
	purchaseIntentRepo repository.PurchaseIntentRepository,
) *ToolsManager {
	return &ToolsManager{
		chatAPIClient:      chatAPIClient,
		sessionRepo:        sessionRepo,
		activityRepo:       activityRepo,
		purchaseIntentRepo: purchaseIntentRepo,
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

type TriggerBuyArgs struct {
	ItemName  string `json:"item_name"`
	ItemPrice string `json:"item_price"`
	Intent    string `json:"intent"`
	Message   string `json:"message,omitempty"` // Internal field for background processing
}

type ReplyMessageArgs struct {
	Message string `json:"message"`
}

type FetchMessagesArgs struct {
	Limit    int    `json:"limit"`
	BeforeTs *int64 `json:"before_ts,omitempty"`
}

type EndSessionArgs struct{}

func (tm *ToolsManager) triggerBuy(ctx context.Context, args TriggerBuyArgs, session SessionContext) error {
	intent := &models.PurchaseIntent{
		SessionID: session.GetSessionID(),
		ChannelID: session.GetChannelID(),
		UserID:    session.GetUserID(),
		ItemName:  args.ItemName,
		ItemPrice: args.ItemPrice,
		Intent:    args.Intent,
	}

	if err := tm.purchaseIntentRepo.Create(ctx, intent); err != nil {
		return fmt.Errorf("failed to create purchase intent: %w", err)
	}

	// Send message to channel if provided (background process, not exposed to LLM)
	if args.Message != "" {
		message := &models.OutgoingMessage{
			ChannelID: session.GetChannelID(),
			SenderID:  session.GetSenderID(),
			Message:   `[PURCHASE_INTENT] ` + args.Message,
		}

		if err := tm.chatAPIClient.SendMessage(ctx, message); err != nil {
			log.Errorf(ctx, "Failed to send message after TriggerBuy: %v", err)
			// Don't return error, just log it - we still want to log the activity
		} else {
			log.Infof(ctx, "Message sent to channel %s after TriggerBuy", session.GetChannelID())
		}
	}

	activity := &models.ChatActivity{
		SessionID: session.GetSessionID(),
		ChannelID: session.GetChannelID(),
		Action:    models.ActivityTriggerBuy,
		Data:      args,
	}

	if err := tm.activityRepo.Create(ctx, activity); err != nil {
		log.Errorf(ctx, "Failed to log TriggerBuy activity: %v", err)
	}

	log.Infof(ctx, "Purchase intent logged: %s wants to buy %s for %s", session.GetUserID(), args.ItemName, args.ItemPrice)
	return nil
}

func (tm *ToolsManager) replyMessage(ctx context.Context, args ReplyMessageArgs, session SessionContext) error {
	message := &models.OutgoingMessage{
		ChannelID: session.GetChannelID(),
		SenderID:  session.GetSenderID(),
		Message:   args.Message,
	}

	if err := tm.chatAPIClient.SendMessage(ctx, message); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	activity := &models.ChatActivity{
		SessionID: session.GetSessionID(),
		ChannelID: session.GetChannelID(),
		Action:    models.ActivityReplyMessage,
		Data:      args,
	}

	if err := tm.activityRepo.Create(ctx, activity); err != nil {
		log.Errorf(ctx, "Failed to log ReplyMessage activity: %v", err)
	}

	log.Infof(ctx, "Message sent to channel %s", session.GetChannelID())
	return nil
}

func minMax(current, minVal, maxVal int) int {
	if current < minVal {
		return minVal
	}
	if current > maxVal {
		return maxVal
	}
	return current
}

func (tm *ToolsManager) fetchMessages(ctx context.Context, args FetchMessagesArgs, session SessionContext) (*models.MessageHistory, error) {
	args.Limit = minMax(args.Limit, 20, 100)

	// Use BeforeTs from args if provided, otherwise use session's saved timestamp
	beforeTs := args.BeforeTs
	if beforeTs == nil {
		beforeTs = session.GetNextMessageTimestamp()
	}

	req := client.MessageHistoryRequest{
		UserID:    session.GetUserID(),
		ChannelID: session.GetChannelID(),
		Limit:     args.Limit,
		BeforeTs:  beforeTs,
	}

	history, err := tm.chatAPIClient.GetMessageHistoryWithParams(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	// Save the timestamp of the oldest message for next pagination
	if len(history.Messages) > 0 {
		oldestMessage := history.Messages[len(history.Messages)-1]
		session.SaveNextMessageTimestamp(oldestMessage.CreatedAt.UnixMilli())
	}

	activity := &models.ChatActivity{
		SessionID: session.GetSessionID(),
		ChannelID: session.GetChannelID(),
		Action:    models.ActivityFetchMessages,
		Data:      args,
	}

	if err := tm.activityRepo.Create(ctx, activity); err != nil {
		log.Errorf(ctx, "Failed to log FetchMessages activity: %v", err)
	}

	log.Infof(ctx, "Fetched %d messages from channel %s", len(history.Messages), session.GetChannelID())
	return history, nil
}

func (tm *ToolsManager) endSession(ctx context.Context, args EndSessionArgs, session SessionContext) error {
	log.Infof(ctx, "Ending session %s as requested by tool", session.GetSessionID().Hex())
	// Use the SessionContext's EndSession method
	if err := session.EndSession(); err != nil {
		return fmt.Errorf("failed to end session: %w", err)
	}

	activity := &models.ChatActivity{
		SessionID: session.GetSessionID(),
		ChannelID: session.GetChannelID(),
		Action:    models.ActivityEndSession,
		Data:      args,
	}

	if err := tm.activityRepo.Create(ctx, activity); err != nil {
		log.Errorf(ctx, "Failed to log EndSession activity: %v", err)
	}

	return nil
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
