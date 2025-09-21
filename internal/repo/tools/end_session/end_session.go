package end_session

import (
	"context"
	"encoding/json"
	"fmt"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/mongodb"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/toolsmanager"
	"github.com/nguyentranbao-ct/chat-bot/pkg/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	ToolName        = "EndSession"
	ToolDescription = "Terminate the current AI conversation session"
)

// EndSessionArgs defines the arguments for the EndSession tool
type EndSessionArgs struct {
	Reason string `json:"reason,omitempty"`
}

type Tool interface {
	toolsmanager.Tool
}

// Tool implements the toolsmanager.Tool interface
type tool struct {
	activityRepo mongodb.ChatActivityRepository
}

// NewTool creates a new EndSession tool instance
func NewTool(
	activityRepo mongodb.ChatActivityRepository,
) Tool {
	t := &tool{
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
	var endArgs EndSessionArgs
	if err := t.parseArgs(args, &endArgs); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	// End the session
	if err := session.EndSession(); err != nil {
		return nil, fmt.Errorf("failed to end session: %w", err)
	}

	// Log activity
	if err := t.logActivity(ctx, endArgs, session); err != nil {
		log.Errorf(ctx, "Failed to log EndSession activity: %v", err)
	}

	reason := endArgs.Reason
	if reason == "" {
		reason = "Session terminated by AI"
	}

	log.Infof(ctx, "Session %s ended: %s", session.GetSessionID(), reason)
	return "Session ended successfully", nil
}

// GetGenkitTool returns the Firebase Genkit tool definition for AI integration
func (t *tool) GetGenkitTool(session toolsmanager.SessionContext, g *genkit.Genkit) ai.Tool {
	return genkit.DefineTool(g, ToolName, ToolDescription,
		func(toolCtx *ai.ToolContext, input EndSessionArgs) (string, error) {
			result, err := t.Execute(session.Context(), input, session)
			if err != nil {
				return "", err
			}

			if resultStr, ok := result.(string); ok {
				return resultStr, nil
			}
			return "Session ended successfully", nil
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
func (t *tool) logActivity(ctx context.Context, args EndSessionArgs, session toolsmanager.SessionContext) error {
	sessionID, err := primitive.ObjectIDFromHex(session.GetSessionID())
	if err != nil {
		return fmt.Errorf("invalid session ID: %w", err)
	}

	activity := &models.ChatActivity{
		SessionID: sessionID,
		ChannelID: session.GetChannelID(),
		Action:    models.ActivityEndSession,
		Data:      args,
	}

	return t.activityRepo.Create(ctx, activity)
}
