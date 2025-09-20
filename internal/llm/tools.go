package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/nguyentranbao-ct/chat-bot/internal/client"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

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

type TriggerBuyArgs struct {
	ItemName  string  `json:"item_name"`
	ItemPrice float64 `json:"item_price"`
	Intent    string  `json:"intent"`
}

type ReplyMessageArgs struct {
	Message string `json:"message"`
}

type FetchMessagesArgs struct {
	Limit int `json:"limit"`
}

type EndSessionArgs struct{}

func (tm *ToolsManager) triggerBuy(ctx context.Context, args TriggerBuyArgs, sessionID primitive.ObjectID, channelID, userID string) error {
	intent := &models.PurchaseIntent{
		SessionID: sessionID,
		ChannelID: channelID,
		UserID:    userID,
		ItemName:  args.ItemName,
		ItemPrice: args.ItemPrice,
		Intent:    args.Intent,
	}

	if err := tm.purchaseIntentRepo.Create(ctx, intent); err != nil {
		return fmt.Errorf("failed to create purchase intent: %w", err)
	}

	activity := &models.ChatActivity{
		SessionID: sessionID,
		ChannelID: channelID,
		Action:    models.ActivityTriggerBuy,
		Data:      args,
	}

	if err := tm.activityRepo.Create(ctx, activity); err != nil {
		log.Printf("Failed to log TriggerBuy activity: %v", err)
	}

	log.Printf("Purchase intent logged: %s wants to buy %s for $%.2f", userID, args.ItemName, args.ItemPrice)
	return nil
}

func (tm *ToolsManager) replyMessage(ctx context.Context, args ReplyMessageArgs, sessionID primitive.ObjectID, channelID, senderID string) error {
	message := &models.OutgoingMessage{
		ChannelID: channelID,
		SenderID:  senderID,
		Message:   args.Message,
	}

	if err := tm.chatAPIClient.SendMessage(ctx, message); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	activity := &models.ChatActivity{
		SessionID: sessionID,
		ChannelID: channelID,
		Action:    models.ActivityReplyMessage,
		Data:      args,
	}

	if err := tm.activityRepo.Create(ctx, activity); err != nil {
		log.Printf("Failed to log ReplyMessage activity: %v", err)
	}

	log.Printf("Message sent to channel %s", channelID)
	return nil
}

func (tm *ToolsManager) fetchMessages(ctx context.Context, args FetchMessagesArgs, sessionID primitive.ObjectID, channelID, userID string) (*models.MessageHistory, error) {
	if args.Limit == 0 {
		args.Limit = 100
	}

	history, err := tm.chatAPIClient.GetMessageHistory(ctx, userID, channelID, args.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	activity := &models.ChatActivity{
		SessionID: sessionID,
		ChannelID: channelID,
		Action:    models.ActivityFetchMessages,
		Data:      args,
	}

	if err := tm.activityRepo.Create(ctx, activity); err != nil {
		log.Printf("Failed to log FetchMessages activity: %v", err)
	}

	log.Printf("Fetched %d messages from channel %s", len(history.Messages), channelID)
	return history, nil
}

func (tm *ToolsManager) endSession(ctx context.Context, args EndSessionArgs, sessionID primitive.ObjectID, channelID string) error {
	if err := tm.sessionRepo.EndSession(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to end session: %w", err)
	}

	activity := &models.ChatActivity{
		SessionID: sessionID,
		ChannelID: channelID,
		Action:    models.ActivityEndSession,
		Data:      args,
	}

	if err := tm.activityRepo.Create(ctx, activity); err != nil {
		log.Printf("Failed to log EndSession activity: %v", err)
	}

	log.Printf("Session %s ended successfully", sessionID.Hex())
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

func (tm *ToolsManager) ExecuteTool(ctx context.Context, toolCall ToolCall, sessionID primitive.ObjectID, channelID, userID, senderID string) (*ToolResult, error) {
	switch toolCall.Name {
	case "TriggerBuy":
		var args TriggerBuyArgs
		if err := convertArgs(toolCall.Args, &args); err != nil {
			return &ToolResult{Success: false, Error: err.Error()}, nil
		}
		if err := tm.triggerBuy(ctx, args, sessionID, channelID, userID); err != nil {
			return &ToolResult{Success: false, Error: err.Error()}, nil
		}
		return &ToolResult{Success: true, Result: "Purchase intent logged successfully"}, nil

	case "ReplyMessage":
		var args ReplyMessageArgs
		if err := convertArgs(toolCall.Args, &args); err != nil {
			return &ToolResult{Success: false, Error: err.Error()}, nil
		}
		if err := tm.replyMessage(ctx, args, sessionID, channelID, senderID); err != nil {
			return &ToolResult{Success: false, Error: err.Error()}, nil
		}
		return &ToolResult{Success: true, Result: "Message sent successfully"}, nil

	case "FetchMessages":
		var args FetchMessagesArgs
		if err := convertArgs(toolCall.Args, &args); err != nil {
			return &ToolResult{Success: false, Error: err.Error()}, nil
		}
		history, err := tm.fetchMessages(ctx, args, sessionID, channelID, userID)
		if err != nil {
			return &ToolResult{Success: false, Error: err.Error()}, nil
		}
		return &ToolResult{Success: true, Result: history}, nil

	case "EndSession":
		var args EndSessionArgs
		if err := convertArgs(toolCall.Args, &args); err != nil {
			return &ToolResult{Success: false, Error: err.Error()}, nil
		}
		if err := tm.endSession(ctx, args, sessionID, channelID); err != nil {
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
