package llm

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"text/template"
	"time"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/nguyentranbao-ct/chat-bot/internal/config"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type GenkitService struct {
	genkit       *genkit.Genkit
	toolsManager *ToolsManager
	config       *config.Config
	tools        []ai.Tool
}

func NewGenkitService(cfg *config.Config, toolsManager *ToolsManager) (*GenkitService, error) {
	ctx := context.Background()

	// Initialize Google AI plugin
	googleAI := &googlegenai.GoogleAI{
		APIKey: cfg.LLM.GoogleAIAPIKey,
	}

	// Initialize Genkit with Google AI plugin
	g := genkit.Init(ctx, genkit.WithPlugins(googleAI))

	// Create Genkit service instance
	gs := &GenkitService{
		genkit:       g,
		toolsManager: toolsManager,
		config:       cfg,
	}

	return gs, nil
}

type PromptData struct {
	ChannelInfo *models.ChannelInfo
	SessionID   string
	UserID      string
	Message     string
}

func (gs *GenkitService) defineTools(ctx context.Context, data *PromptData) (SessionContext, error) {
	sessionID, err := primitive.ObjectIDFromHex(data.SessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	channelID := getChannelID(data)
	userID := data.UserID

	// Find the other user ID (not the current user) for reply messages
	otherUserID := findOtherUserID(data.ChannelInfo, userID)
	if otherUserID == "" {
		otherUserID = "chat-bot" // Default fallback
	}

	// Create session context for tool operations
	session := NewSessionContext(SessionContextConfig{
		SessionID:   sessionID,
		ChannelID:   channelID,
		UserID:      userID,
		SenderID:    otherUserID,
		SessionRepo: gs.toolsManager.sessionRepo,
	})

	// Define TriggerBuy tool
	triggerBuyTool := genkit.DefineTool(gs.genkit, "TriggerBuy", "Logs purchase intent and notifies sellers when a user shows buying interest",
		func(toolCtx *ai.ToolContext, input TriggerBuyArgs) (string, error) {
			timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			if err := gs.toolsManager.triggerBuy(timeoutCtx, input, session); err != nil {
				return "", err
			}
			result := "Purchase intent logged successfully"
			if input.Message != "" {
				result += " and message sent to channel"
			}
			return result, nil
		})

	// Define ReplyMessage tool
	replyMessageTool := genkit.DefineTool(gs.genkit, "ReplyMessage", "Send a message to the chat channel via chat-api",
		func(toolCtx *ai.ToolContext, input ReplyMessageArgs) (string, error) {
			timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			if err := gs.toolsManager.replyMessage(timeoutCtx, input, session); err != nil {
				return "", err
			}
			return "Message sent successfully", nil
		})

	// Define FetchMessages tool
	fetchMessagesTool := genkit.DefineTool(gs.genkit, "FetchMessages", "Fetch additional conversation history from the channel",
		func(toolCtx *ai.ToolContext, input FetchMessagesArgs) (*models.MessageHistory, error) {
			timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			return gs.toolsManager.fetchMessages(timeoutCtx, input, session)
		})

	// Define EndSession tool
	endSessionTool := genkit.DefineTool(gs.genkit, "EndSession", "Terminate the current AI conversation session",
		func(toolCtx *ai.ToolContext, input EndSessionArgs) (string, error) {
			timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			if err := gs.toolsManager.endSession(timeoutCtx, input, session); err != nil {
				return "", err
			}
			return "Session ended successfully", nil
		})

	// Store tools for later use
	gs.tools = []ai.Tool{triggerBuyTool, replyMessageTool, fetchMessagesTool, endSessionTool}

	return session, nil
}

func (gs *GenkitService) ProcessMessage(ctx context.Context, chatMode *models.ChatMode, data *PromptData) error {
	prompt, err := gs.buildPrompt(chatMode.PromptTemplate, data)
	if err != nil {
		return fmt.Errorf("failed to build prompt: %w", err)
	}

	log.Printf("Processing with chat mode: %s", chatMode.Name)

	// Define tools with context data for this request
	session, err := gs.defineTools(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to define tools: %w", err)
	}

	// Get available tools for this chat mode
	availableTools := gs.getToolsForMode(chatMode.Tools)

	// Build initial messages
	messages := []*ai.Message{
		ai.NewSystemTextMessage(prompt),
		ai.NewUserTextMessage(data.Message),
	}

	// Agent loop: iterate until no more tools are called or max iterations reached
	for i := 0; i < chatMode.MaxIterations; i++ {
		log.Printf("Agent iteration %d/%d", i+1, chatMode.MaxIterations)

		// Convert tools to ToolRef
		var toolRefs []ai.ToolRef
		for _, tool := range availableTools {
			toolRefs = append(toolRefs, tool)
		}

		// Generate response using real Genkit API
		response, err := genkit.Generate(ctx, gs.genkit,
			ai.WithMessages(messages...),
			ai.WithModelName(chatMode.Model),
			ai.WithTools(toolRefs...),
			ai.WithConfig(&ai.GenerationCommonConfig{
				MaxOutputTokens: chatMode.MaxResponseTokens,
				Temperature:     0.7,
			}),
		)
		if err != nil {
			return fmt.Errorf("failed to generate response: %w", err)
		}

		// Add assistant response to conversation
		if response.Text() != "" {
			messages = append(messages, ai.NewModelTextMessage(response.Text()))
			log.Printf("AI Response: %s", response.Text())
		}

		// Check if there are tool requests in the response
		toolRequests := response.ToolRequests()
		if len(toolRequests) == 0 {
			log.Printf("No tool requests found, ending conversation")
			break
		}

		// Tool execution is handled automatically by Genkit when tools are properly defined
		// The tool responses will be included in the next iteration automatically

		// Check if session has been ended by a tool
		if session.IsEnded() {
			log.Printf("Session has been terminated by tool execution, ending conversation")
			break
		}
	}

	log.Printf("Agent processing complete for session %s", data.SessionID)
	return nil
}

func (gs *GenkitService) buildPrompt(templateStr string, data *PromptData) (string, error) {
	tmpl, err := template.New("prompt").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

func (gs *GenkitService) getToolsForMode(toolNames []string) []ai.Tool {
	var availableTools []ai.Tool
	for _, toolName := range toolNames {
		for _, tool := range gs.tools {
			// Check if this tool matches the requested tool name
			if tool.Name() == toolName {
				availableTools = append(availableTools, tool)
				break
			}
		}
	}
	return availableTools
}

func getChannelID(data *PromptData) string {
	if data.ChannelInfo != nil && data.ChannelInfo.ID != "" {
		return data.ChannelInfo.ID
	}
	return "unknown-channel"
}

// findOtherUserID finds the user ID that is not the current user (for reply messages)
func findOtherUserID(channelInfo *models.ChannelInfo, currentUserID string) string {
	if channelInfo == nil || len(channelInfo.Participants) == 0 {
		return ""
	}

	for _, participant := range channelInfo.Participants {
		if participant.UserID != currentUserID {
			return participant.UserID
		}
	}
	return ""
}
