package llm

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"
	"time"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/nguyentranbao-ct/chat-bot/internal/config"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type GenkitService struct {
	genkit         *genkit.Genkit
	toolsManager   *ToolsManager
	config         *config.Config
	tools          []ai.Tool
	currentSession SessionContext // Current session context for tool execution
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

	// Define tools once during initialization
	if err := gs.initializeTools(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize tools: %w", err)
	}

	return gs, nil
}

type PromptData struct {
	ChannelInfo    *models.ChannelInfo
	SessionID      string
	UserID         string
	SenderRole     string
	Message        string
	RecentMessages *models.MessageHistory
}

func (gs *GenkitService) initializeTools(ctx context.Context) error {
	// Define TriggerBuy tool
	triggerBuyTool := genkit.DefineTool(gs.genkit, "TriggerBuy", "Logs purchase intent when a user shows buying interest",
		func(toolCtx *ai.ToolContext, input TriggerBuyArgs) (string, error) {
			timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			if err := gs.toolsManager.triggerBuy(timeoutCtx, input, gs.currentSession); err != nil {
				return "", err
			}
			return "Purchase intent logged successfully", nil
		})

	// Define ReplyMessage tool
	replyMessageTool := genkit.DefineTool(gs.genkit, "ReplyMessage", "Send a message to the chat channel via chat-api",
		func(toolCtx *ai.ToolContext, input ReplyMessageArgs) (string, error) {
			timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			if err := gs.toolsManager.replyMessage(timeoutCtx, input, gs.currentSession); err != nil {
				return "", err
			}
			return "Message sent successfully", nil
		})

	// Define FetchMessages tool
	fetchMessagesTool := genkit.DefineTool(gs.genkit, "FetchMessages", "Fetch additional conversation history from the channel",
		func(toolCtx *ai.ToolContext, input FetchMessagesArgs) (*models.MessageHistory, error) {
			timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			return gs.toolsManager.fetchMessages(timeoutCtx, input, gs.currentSession)
		})

	// Define EndSession tool
	endSessionTool := genkit.DefineTool(gs.genkit, "EndSession", "Terminate the current AI conversation session",
		func(toolCtx *ai.ToolContext, input EndSessionArgs) (string, error) {
			timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			if err := gs.toolsManager.endSession(timeoutCtx, input, gs.currentSession); err != nil {
				return "", err
			}
			return "Session ended successfully", nil
		})

	// Store tools for later use
	gs.tools = []ai.Tool{triggerBuyTool, replyMessageTool, fetchMessagesTool, endSessionTool}

	return nil
}

func (gs *GenkitService) createSessionContext(data *PromptData) (SessionContext, error) {
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

	return session, nil
}

func (gs *GenkitService) ProcessMessage(ctx context.Context, chatMode *models.ChatMode, data *PromptData) error {
	// Evaluate the  condition first
	shouldProcess, err := gs.evaluateCondition(chatMode.Condition, data)
	if err != nil {
		return fmt.Errorf("failed to evaluate when condition: %w", err)
	}
	if !shouldProcess {
		log.Infow(ctx, "When condition evaluated to false, stopping processing",
			"chat_mode", chatMode.Name,
			"sender_role", data.SenderRole,
			"user_id", data.UserID)
		return nil
	}
	log.Infow(ctx, "When condition evaluated to true, proceeding with processing",
		"chat_mode", chatMode.Name,
		"sender_role", data.SenderRole)

	prompt, err := gs.buildPrompt(chatMode.PromptTemplate, data)
	if err != nil {
		return fmt.Errorf("failed to build prompt: %w", err)
	}

	log.Infow(ctx, "Processing message", "chat_mode", chatMode.Name, "session_id", data.SessionID)

	// Create session context for this request and set it as current session
	session, err := gs.createSessionContext(data)
	if err != nil {
		return fmt.Errorf("failed to create session context: %w", err)
	}
	gs.currentSession = session

	// Get available tools for this chat mode
	availableTools := gs.getToolsForMode(chatMode.Tools)

	// Build initial messages
	messages := []*ai.Message{
		ai.NewSystemTextMessage(prompt),
	}

	// Add recent messages as conversation history
	if data.RecentMessages != nil && len(data.RecentMessages.Messages) > 0 {
		for _, msg := range data.RecentMessages.Messages {
			// Determine the role based on sender ID
			if msg.SenderID == data.UserID {
				messages = append(messages, ai.NewUserTextMessage(msg.Message))
			} else {
				messages = append(messages, ai.NewModelTextMessage(msg.Message))
			}
		}

		// Save the timestamp of the oldest message for pagination
		oldestMessage := data.RecentMessages.Messages[len(data.RecentMessages.Messages)-1]
		session.SaveNextMessageTimestamp(oldestMessage.CreatedAt.UnixMilli())
	}

	// Add the current message
	messages = append(messages, ai.NewUserTextMessage(data.Message))

	// Agent loop: iterate until no more tools are called or max iterations reached
	for i := 0; i < chatMode.MaxIterations; i++ {
		log.Infow(ctx, "Agent iteration", "current", i+1, "max", chatMode.MaxIterations)

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
		)
		if err != nil {
			return fmt.Errorf("failed to generate response: %w", err)
		}

		// Add assistant response to conversation
		if response.Text() != "" {
			messages = append(messages, ai.NewModelTextMessage(response.Text()))
			log.Infow(ctx, "AI generated text response", "response", response.Text())
		}

		// Check if there are tool requests in the response
		toolRequests := response.ToolRequests()
		if len(toolRequests) == 0 {
			log.Infow(ctx, "No tool requests found", "ai_response", response.Text())
			log.Info(ctx, "Ending conversation - AI did not use any tools")
			break
		}

		log.Infow(ctx, "Processing tool requests", "count", len(toolRequests))

		// Execute tools and add responses to conversation
		var toolResponseParts []*ai.Part
		for _, req := range toolRequests {
			log.Infow(ctx, "Executing tool", "tool_name", req.Name)

			// Find the tool by name
			var tool ai.Tool
			for _, t := range availableTools {
				if t.Name() == req.Name {
					tool = t
					break
				}
			}

			if tool == nil {
				log.Errorw(ctx, "Tool not found", "tool_name", req.Name)
				continue
			}

			// Execute the tool
			output, err := tool.RunRaw(ctx, req.Input)
			if err != nil {
				log.Errorw(ctx, "Tool execution failed", "tool_name", req.Name, "error", err)
				continue
			}

			log.Infow(ctx, "Tool executed successfully", "tool_name", req.Name)

			// Add tool response to the conversation
			toolResponseParts = append(toolResponseParts,
				ai.NewToolResponsePart(&ai.ToolResponse{
					Name:   req.Name,
					Ref:    req.Ref,
					Output: output,
				}))
		}

		// Add tool responses to the conversation
		if len(toolResponseParts) > 0 {
			messages = append(messages, ai.NewMessage(ai.RoleTool, nil, toolResponseParts...))
		}

		// Check if session has been ended by a tool
		if session.IsEnded() {
			log.Info(ctx, "Session has been terminated by tool execution, ending conversation")
			break
		}
	}

	log.Infow(ctx, "Agent processing complete", "session_id", data.SessionID)
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

func (gs *GenkitService) evaluateCondition(whenTemplate string, data *PromptData) (bool, error) {
	tmpl, err := template.New("when").Parse(whenTemplate)
	if err != nil {
		return false, fmt.Errorf("failed to parse when template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return false, fmt.Errorf("failed to execute when template: %w", err)
	}

	result := strings.TrimSpace(buf.String())
	return result == "true", nil
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
