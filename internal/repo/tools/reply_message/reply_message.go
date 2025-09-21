package reply_message

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/chatapi"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/mongodb"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/toolsmanager"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	ToolName        = "ReplyMessage"
	ToolDescription = "Send a message to the chat channel via chat-api"
)

// ReplyMessageArgs defines the arguments for the ReplyMessage tool
type ReplyMessageArgs struct {
	Message string `json:"message"`
}

type Tool interface {
	toolsmanager.Tool
}

// Tool implements the toolsmanager.Tool interface
type tool struct {
	chatAPIClient chatapi.Client
	activityRepo  mongodb.ChatActivityRepository
}

// NewTool creates a new ReplyMessage tool instance
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
	var replyArgs ReplyMessageArgs
	if err := t.parseArgs(args, &replyArgs); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Send the message
	outgoingMessage := &models.OutgoingMessage{
		ChannelID: session.GetChannelID(),
		SenderID:  session.GetSenderID(),
		Message:   replyArgs.Message,
	}

	if err := t.chatAPIClient.SendMessage(ctx, outgoingMessage); err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// Log activity
	if err := t.logActivity(ctx, replyArgs, session); err != nil {
		log.Errorf(ctx, "Failed to log ReplyMessage activity: %v", err)
	}

	log.Infof(ctx, "Message sent to channel %s: %s", session.GetChannelID(), replyArgs.Message)
	return "Message sent successfully", nil
}

// GetGenkitTool returns the Firebase Genkit tool definition for AI integration
func (t *tool) GetGenkitTool(session toolsmanager.SessionContext, g *genkit.Genkit) ai.Tool {
	return genkit.DefineTool(g, ToolName, ToolDescription,
		func(toolCtx *ai.ToolContext, input ReplyMessageArgs) (string, error) {
			timeoutCtx, cancel := context.WithTimeout(session.Context(), 30*time.Second)
			defer cancel()

			result, err := t.Execute(timeoutCtx, input, session)
			if err != nil {
				return "", err
			}

			if resultStr, ok := result.(string); ok {
				return resultStr, nil
			}
			return "Message sent successfully", nil
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
func (t *tool) logActivity(ctx context.Context, args ReplyMessageArgs, session toolsmanager.SessionContext) error {
	sessionID, err := primitive.ObjectIDFromHex(session.GetSessionID())
	if err != nil {
		return fmt.Errorf("invalid session ID: %w", err)
	}

	activity := &models.ChatActivity{
		SessionID: sessionID,
		ChannelID: session.GetChannelID(),
		Action:    models.ActivityReplyMessage,
		Data:      args,
	}

	return t.activityRepo.Create(ctx, activity)
}
