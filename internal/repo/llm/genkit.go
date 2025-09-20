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
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/llm/tools"
	"github.com/nguyentranbao-ct/chat-bot/pkg/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Service interface {
	ProcessMessage(ctx context.Context, chatMode *models.ChatMode, data *PromptData) error
}

type genkitService struct {
	genkit         *genkit.Genkit
	toolsManager   *ToolsManager
	config         *config.Config
	tools          []ai.Tool
	currentSession tools.SessionContext
}

func NewGenkitService(cfg *config.Config, toolsManager *ToolsManager) (Service, error) {
	ctx := context.Background()

	// Initialize Google AI plugin
	googleAI := &googlegenai.GoogleAI{
		APIKey: cfg.LLM.GoogleAIAPIKey,
	}

	// Initialize Genkit with Google AI plugin
	g := genkit.Init(ctx, genkit.WithPlugins(googleAI))

	// Create Genkit service instance
	gs := &genkitService{
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

func (gs *genkitService) initializeTools(ctx context.Context) error {
	triggerBuyTool := gs.createTriggerBuyTool(ctx)
	replyMessageTool := gs.createReplyMessageTool(ctx)
	fetchMessagesTool := gs.createFetchMessagesTool(ctx)
	endSessionTool := gs.createEndSessionTool(ctx)

	gs.tools = []ai.Tool{triggerBuyTool, replyMessageTool, fetchMessagesTool, endSessionTool}
	return nil
}

func (gs *genkitService) createTriggerBuyTool(ctx context.Context) ai.Tool {
	return genkit.DefineTool(gs.genkit, "TriggerBuy", "Logs purchase intent when a user shows buying interest",
		func(toolCtx *ai.ToolContext, input TriggerBuyArgs) (string, error) {
			timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			if err := gs.toolsManager.triggerBuy(timeoutCtx, input, gs.currentSession); err != nil {
				return "", err
			}
			return "Purchase intent logged successfully", nil
		})
}

func (gs *genkitService) createReplyMessageTool(ctx context.Context) ai.Tool {
	return genkit.DefineTool(gs.genkit, "ReplyMessage", "Send a message to the chat channel via chat-api",
		func(toolCtx *ai.ToolContext, input ReplyMessageArgs) (string, error) {
			timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			if err := gs.toolsManager.replyMessage(timeoutCtx, input, gs.currentSession); err != nil {
				return "", err
			}
			return "Message sent successfully", nil
		})
}

func (gs *genkitService) createFetchMessagesTool(ctx context.Context) ai.Tool {
	return genkit.DefineTool(gs.genkit, "FetchMessages", "Fetch additional conversation history from the channel",
		func(toolCtx *ai.ToolContext, input FetchMessagesArgs) (*models.MessageHistory, error) {
			timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			return gs.toolsManager.fetchMessages(timeoutCtx, input, gs.currentSession)
		})
}

func (gs *genkitService) createEndSessionTool(ctx context.Context) ai.Tool {
	return genkit.DefineTool(gs.genkit, "EndSession", "Terminate the current AI conversation session",
		func(toolCtx *ai.ToolContext, input EndSessionArgs) (string, error) {
			timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			if err := gs.toolsManager.endSession(timeoutCtx, input, gs.currentSession); err != nil {
				return "", err
			}
			return "Session ended successfully", nil
		})
}

func (gs *genkitService) createSessionContext(data *PromptData) (tools.SessionContext, error) {
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

func (gs *genkitService) ProcessMessage(ctx context.Context, chatMode *models.ChatMode, data *PromptData) error {
	shouldProcess, err := gs.evaluateCondition(chatMode.Condition, data)
	if err != nil {
		return fmt.Errorf("failed to evaluate when condition: %w", err)
	}
	if !shouldProcess {
		gs.logConditionResult(ctx, chatMode, data, false)
		return nil
	}
	gs.logConditionResult(ctx, chatMode, data, true)

	prompt, err := gs.buildPrompt(chatMode.PromptTemplate, data)
	if err != nil {
		return fmt.Errorf("failed to build prompt: %w", err)
	}

	log.Infow(ctx, "Processing message", "chat_mode", chatMode.Name, "session_id", data.SessionID)

	session, err := gs.createSessionContext(data)
	if err != nil {
		return fmt.Errorf("failed to create session context: %w", err)
	}
	gs.currentSession = session

	availableTools := gs.getToolsForMode(chatMode.Tools)
	messages := gs.buildInitialMessages(prompt, data, session)

	if err := gs.runAgentLoop(ctx, chatMode, messages, availableTools, session); err != nil {
		return err
	}

	log.Infow(ctx, "Agent processing complete", "session_id", data.SessionID)
	return nil
}

func (gs *genkitService) logConditionResult(ctx context.Context, chatMode *models.ChatMode, data *PromptData, shouldProcess bool) {
	if shouldProcess {
		log.Infow(ctx, "When condition evaluated to true, proceeding with processing",
			"chat_mode", chatMode.Name,
			"sender_role", data.SenderRole)
	} else {
		log.Infow(ctx, "When condition evaluated to false, stopping processing",
			"chat_mode", chatMode.Name,
			"sender_role", data.SenderRole,
			"user_id", data.UserID)
	}
}

func (gs *genkitService) buildInitialMessages(prompt string, data *PromptData, session tools.SessionContext) []*ai.Message {
	messages := []*ai.Message{
		ai.NewSystemTextMessage(prompt),
	}

	if data.RecentMessages != nil && len(data.RecentMessages.Messages) > 0 {
		messages = gs.addRecentMessages(messages, data, session)
	}

	messages = append(messages, ai.NewUserTextMessage(data.Message))
	return messages
}

func (gs *genkitService) addRecentMessages(messages []*ai.Message, data *PromptData, session tools.SessionContext) []*ai.Message {
	for _, msg := range data.RecentMessages.Messages {
		if msg.SenderID == data.UserID {
			messages = append(messages, ai.NewUserTextMessage(msg.Message))
		} else {
			messages = append(messages, ai.NewModelTextMessage(msg.Message))
		}
	}

	oldestMessage := data.RecentMessages.Messages[len(data.RecentMessages.Messages)-1]
	session.SaveNextMessageTimestamp(oldestMessage.CreatedAt.UnixMilli())
	return messages
}

func (gs *genkitService) runAgentLoop(ctx context.Context, chatMode *models.ChatMode, messages []*ai.Message, availableTools []ai.Tool, session tools.SessionContext) error {
	for i := 0; i < chatMode.MaxIterations; i++ {
		log.Infow(ctx, "Agent iteration", "current", i+1, "max", chatMode.MaxIterations)

		response, err := gs.generateResponse(ctx, chatMode, messages, availableTools)
		if err != nil {
			return fmt.Errorf("failed to generate response: %w", err)
		}

		if response.Text() != "" {
			messages = append(messages, ai.NewModelTextMessage(response.Text()))
			log.Infow(ctx, "AI generated text response", "response", response.Text())
		}

		toolRequests := response.ToolRequests()
		if len(toolRequests) == 0 {
			log.Infow(ctx, "Ending conversation - AI did not use any tools", "ai_response", response.Text())
			break
		}

		log.Infow(ctx, "Processing tool requests", "count", len(toolRequests))

		toolResponseParts, err := gs.executeToolRequests(ctx, toolRequests, availableTools)
		if err != nil {
			return err
		}

		if len(toolResponseParts) > 0 {
			messages = append(messages, ai.NewMessage(ai.RoleTool, nil, toolResponseParts...))
		}

		if session.IsEnded() {
			log.Info(ctx, "Session has been terminated by tool execution, ending conversation")
			break
		}
	}
	return nil
}

func (gs *genkitService) generateResponse(ctx context.Context, chatMode *models.ChatMode, messages []*ai.Message, availableTools []ai.Tool) (*ai.ModelResponse, error) {
	var toolRefs []ai.ToolRef
	for _, tool := range availableTools {
		toolRefs = append(toolRefs, tool)
	}

	return genkit.Generate(ctx, gs.genkit,
		ai.WithMessages(messages...),
		ai.WithModelName(chatMode.Model),
		ai.WithTools(toolRefs...),
	)
}

func (gs *genkitService) executeToolRequests(ctx context.Context, toolRequests []*ai.ToolRequest, availableTools []ai.Tool) ([]*ai.Part, error) {
	var toolResponseParts []*ai.Part
	for _, req := range toolRequests {
		log.Infow(ctx, "Executing tool", "tool_name", req.Name)

		tool := gs.findToolByName(req.Name, availableTools)
		if tool == nil {
			log.Errorw(ctx, "Tool not found", "tool_name", req.Name)
			continue
		}

		output, err := tool.RunRaw(ctx, req.Input)
		if err != nil {
			log.Errorw(ctx, "Tool execution failed", "tool_name", req.Name, "error", err)
			continue
		}

		log.Infow(ctx, "Tool executed successfully", "tool_name", req.Name)

		toolResponseParts = append(toolResponseParts,
			ai.NewToolResponsePart(&ai.ToolResponse{
				Name:   req.Name,
				Ref:    req.Ref,
				Output: output,
			}))
	}
	return toolResponseParts, nil
}

func (gs *genkitService) findToolByName(name string, availableTools []ai.Tool) ai.Tool {
	for _, tool := range availableTools {
		if tool.Name() == name {
			return tool
		}
	}
	return nil
}

func (gs *genkitService) buildPrompt(templateStr string, data *PromptData) (string, error) {
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

func (gs *genkitService) evaluateCondition(whenTemplate string, data *PromptData) (bool, error) {
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

func (gs *genkitService) getToolsForMode(toolNames []string) []ai.Tool {
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
