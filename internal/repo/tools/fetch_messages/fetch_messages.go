package fetch_messages

import (
	"context"
	"encoding/json"
	"fmt"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/mongodb"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/toolsmanager"
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
	messagesRepo mongodb.ChatMessageRepository
	activityRepo mongodb.ChatActivityRepository
}

// NewTool creates a new FetchMessages tool instance
func NewTool(
	messagesRepo mongodb.ChatMessageRepository,
	activityRepo mongodb.ChatActivityRepository,
	toolsManager toolsmanager.ToolsManager,
) Tool {
	t := &tool{
		messagesRepo: messagesRepo,
		activityRepo: activityRepo,
	}
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

	// Fetch messages from database
	messages, err := t.messagesRepo.GetChannelMessages(ctx, session.GetChannelID(), fetchArgs.Limit, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	// Log activity
	if err := t.logActivity(ctx, fetchArgs, session); err != nil {
		log.Errorf(ctx, "Failed to log FetchMessages activity: %v", err)
	}

	log.Infof(ctx, "Fetched %d messages for channel %s", len(messages), session.GetChannelID())
	return messages, nil
}

// GetGenkitTool returns the Firebase Genkit tool definition for AI integration
func (t *tool) GetGenkitTool(session toolsmanager.SessionContext, g *genkit.Genkit) ai.Tool {
	return genkit.DefineTool(g, ToolName, ToolDescription,
		func(toolCtx *ai.ToolContext, input FetchMessagesArgs) (*models.MessageHistory, error) {
			result, err := t.Execute(session.Context(), input, session)
			if err != nil {
				return nil, err
			}
			// Convert []*models.ChatMessage to *models.MessageHistory
			messages := result.([]*models.ChatMessage)
			history := &models.MessageHistory{
				Messages: make([]models.HistoryMessage, len(messages)),
				HasMore:  len(messages) == input.Limit,
			}
			for i, msg := range messages {
				history.Messages[i] = models.HistoryMessage{
					ID:        msg.ID.Hex(),
					ChannelID: msg.ChannelID,
					SenderID:  msg.SenderID,
					Content:   msg.Content,
					CreatedAt: msg.CreatedAt,
				}
			}
			return history, nil
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
	activity := &models.ChatActivity{
		SessionID: session.GetSessionID(),
		ChannelID: session.GetChannelID(),
		Action:    models.ActivityFetchMessages,
		Data:      args,
	}

	return t.activityRepo.Create(ctx, activity)
}
