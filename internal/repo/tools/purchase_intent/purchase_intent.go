package purchase_intent

import (
	"context"
	"encoding/json"
	"fmt"

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
	ToolName        = "PurchaseIntent"
	ToolDescription = `Logs purchase intent when a user shows buying interest with a percentage confidence level. Only trigger PurchaseIntent when the customer demonstrates CLEAR, EXPLICIT purchase intent with specific indicators:
- Direct purchase statements: "I want to buy this", "I'll take it", "How do I purchase this", "Add to cart", "I'm ready to buy"
- Payment-related questions: "What payment methods do you accept?", "How much is shipping?", "What's the total cost?"
- Commitment language: "I've decided to get this", "This is exactly what I need", "I'm convinced"
- Urgency indicators: "When can I get this?", "How fast can you ship this?"

When you do detect genuine purchase intent:
1. Call PurchaseIntent to log the purchase intent
2. IMMEDIATELY follow with ReplyMessage to acknowledge their decision and guide them to next steps
`
)

// PurchaseIntentArgs defines the arguments for the PurchaseIntent tool
type PurchaseIntentArgs struct {
	ItemName   string `json:"item_name"`
	ItemPrice  string `json:"item_price"`
	Intent     string `json:"intent"`
	Percentage int    `json:"percentage"`
	Message    string `json:"message,omitempty"`
}

type Tool interface {
	toolsmanager.Tool
}

// Tool implements the toolsmanager.Tool interface
type tool struct {
	chatAPIClient      chatapi.Client
	activityRepo       mongodb.ChatActivityRepository
	purchaseIntentRepo mongodb.PurchaseIntentRepository
}

// NewTool creates a new PurchaseIntent tool instance
func NewTool(
	chatAPIClient chatapi.Client,
	activityRepo mongodb.ChatActivityRepository,
	purchaseIntentRepo mongodb.PurchaseIntentRepository,
	toolsManager toolsmanager.ToolsManager,
) Tool {
	t := &tool{
		chatAPIClient:      chatAPIClient,
		activityRepo:       activityRepo,
		purchaseIntentRepo: purchaseIntentRepo,
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
	var purchaseIntentArgs PurchaseIntentArgs
	if err := t.parseArgs(args, &purchaseIntentArgs); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Convert session ID back to ObjectID for database operations
	sessionID, err := primitive.ObjectIDFromHex(session.GetSessionID())
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	// Create purchase intent
	intent := &models.PurchaseIntent{
		SessionID:  sessionID,
		ChannelID:  session.GetChannelID(),
		UserID:     session.GetUserID(),
		ItemName:   purchaseIntentArgs.ItemName,
		ItemPrice:  purchaseIntentArgs.ItemPrice,
		Intent:     purchaseIntentArgs.Intent,
		Percentage: purchaseIntentArgs.Percentage,
	}

	if err := t.purchaseIntentRepo.Create(ctx, intent); err != nil {
		return nil, fmt.Errorf("failed to create purchase intent: %w", err)
	}

	// Send message if provided
	if purchaseIntentArgs.Message != "" {
		if err := t.sendPurchaseMessage(session, purchaseIntentArgs); err != nil {
			log.Errorf(ctx, "Failed to send message after PurchaseIntent: %v", err)
		} else {
			log.Infof(ctx, "Message sent to channel %s after PurchaseIntent", session.GetChannelID())
		}
	}

	// Log activity
	if err := t.logActivity(ctx, purchaseIntentArgs, sessionID, session); err != nil {
		log.Errorf(ctx, "Failed to log PurchaseIntent activity: %v", err)
	}

	log.Infof(ctx, "Purchase intent logged: %s wants to buy %s for %s (%d%% confidence)", session.GetUserID(), purchaseIntentArgs.ItemName, purchaseIntentArgs.ItemPrice, purchaseIntentArgs.Percentage)
	return "Purchase intent logged successfully", nil
}

// GetGenkitTool returns the Firebase Genkit tool definition for AI integration
func (t *tool) GetGenkitTool(session toolsmanager.SessionContext, g *genkit.Genkit) ai.Tool {
	return genkit.DefineTool(g, ToolName, ToolDescription,
		func(toolCtx *ai.ToolContext, input PurchaseIntentArgs) (string, error) {
			// This is a placeholder - in practice, the session context will be provided
			// by the tool manager when the tool is executed
			result, err := t.Execute(session.Context(), input, session)
			if err != nil {
				return "", err
			}

			if resultStr, ok := result.(string); ok {
				return resultStr, nil
			}
			return "Purchase intent logged successfully", nil
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

// sendPurchaseMessage sends a message to the chat channel
func (t *tool) sendPurchaseMessage(session toolsmanager.SessionContext, args PurchaseIntentArgs) error {
	outgoingMessage := &models.OutgoingMessage{
		ChannelID: session.GetChannelID(),
		SenderID:  session.GetSenderID(),
		Message:   fmt.Sprintf(`[PURCHASE_INTENT %d%%] %s`, args.Percentage, args.Message),
	}

	return t.chatAPIClient.SendMessage(session.Context(), outgoingMessage)
}

// logActivity logs the tool execution activity
func (t *tool) logActivity(ctx context.Context, args PurchaseIntentArgs, sessionID primitive.ObjectID, session toolsmanager.SessionContext) error {
	activity := &models.ChatActivity{
		SessionID: sessionID,
		ChannelID: session.GetChannelID(),
		Action:    models.ActivityPurchaseIntent,
		Data:      args,
	}

	return t.activityRepo.Create(ctx, activity)
}
