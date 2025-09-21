package fetch_messages

import (
	"context"
	"encoding/json"
	"fmt"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/chatapi"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/mongodb"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/toolsmanager"
	"github.com/nguyentranbao-ct/chat-bot/pkg/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	ToolName        = "FetchMessages"
	ToolDescription = "Fetch additional conversation history from the channel"
)

// FetchMessagesArgs defines the arguments for the FetchMessages tool
type FetchMessagesArgs struct {
	Limit int `json:"limit,omitempty"`
}

type Tool interface {
	toolsmanager.Tool
}

// Tool implements the toolsmanager.Tool interface
type tool struct {
	chatAPIClient chatapi.Client
	activityRepo  mongodb.ChatActivityRepository
}

// NewTool creates a new FetchMessages tool instance
func NewTool(
	chatAPIClient chatapi.Client,
	activityRepo mongodb.ChatActivityRepository,
	toolsManager toolsmanager.ToolsManager,
) Tool {
	t := &tool{
		chatAPIClient: chatAPIClient,
		activityRepo:  activityRepo,
	}
	toolsManager.AddTool(t)
	return t
}

// Name returns the tool's unique identifier
func (t *tool) Name() string {
	return ToolName
}

// Description returns a human-readable description of what the tool does
func (t *tool) Description() string {
	return ToolDescription
}

// Execute runs the tool with the given arguments and session context
func (t *tool) Execute(ctx context.Context, args interface{}, session toolsmanager.SessionContext) (interface{}, error) {
	// Parse arguments
	var fetchArgs FetchMessagesArgs
	if err := t.parseArgs(args, &fetchArgs); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Set default limit if not provided
	if fetchArgs.Limit <= 0 {
		fetchArgs.Limit = 100
	}

	// Fetch messages from chat API
	messageHistory, err := t.chatAPIClient.GetMessageHistoryWithParams(ctx, chatapi.MessageHistoryRequest{
		ChannelID: session.GetChannelID(),
		UserID:    session.GetUserID(),
		Limit:     fetchArgs.Limit,
		BeforeTs:  session.GetNextMessageTimestamp(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	// Log activity
	if err := t.logActivity(ctx, fetchArgs, session); err != nil {
		log.Errorf(ctx, "Failed to log FetchMessages activity: %v", err)
	}

	log.Infof(ctx, "Fetched %d messages for channel %s", len(messageHistory.Messages), session.GetChannelID())
	return messageHistory, nil
}

// GetGenkitTool returns the Firebase Genkit tool definition for AI integration
func (t *tool) GetGenkitTool(session toolsmanager.SessionContext, g *genkit.Genkit) ai.Tool {
	return genkit.DefineTool(g, ToolName, ToolDescription,
		func(toolCtx *ai.ToolContext, input FetchMessagesArgs) (*models.MessageHistory, error) {
			result, err := t.Execute(session.Context(), input, session)
			if err != nil {
				return nil, err
			}

			if messageHistory, ok := result.(*models.MessageHistory); ok {
				return messageHistory, nil
			}
			return nil, fmt.Errorf("unexpected result type from FetchMessages")
		})
}

// parseArgs converts interface{} arguments to the expected type
func (t *tool) parseArgs(args interface{}, target interface{}) error {
	data, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("failed to marshal args: %w", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal args: %w", err)
	}
	return nil
}

// logActivity logs the tool execution activity
func (t *tool) logActivity(ctx context.Context, args FetchMessagesArgs, session toolsmanager.SessionContext) error {
	sessionID, err := primitive.ObjectIDFromHex(session.GetSessionID())
	if err != nil {
		return fmt.Errorf("invalid session ID: %w", err)
	}

	activity := &models.ChatActivity{
		SessionID: sessionID,
		ChannelID: session.GetChannelID(),
		Action:    models.ActivityFetchMessages,
		Data:      args,
	}

	return t.activityRepo.Create(ctx, activity)
}
