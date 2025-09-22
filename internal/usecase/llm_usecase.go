package usecase

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/nguyentranbao-ct/chat-bot/internal/config"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/mongodb"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/tools/end_session"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/tools/fetch_messages"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/tools/list_products"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/tools/purchase_intent"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/tools/reply_message"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/toolsmanager"
	"github.com/nguyentranbao-ct/chat-bot/pkg/util"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// LLMUsecase defines the interface for LLM operations
type LLMUsecase interface {
	ProcessMessage(ctx context.Context, chatMode *models.ChatMode, data *PromptContext) error
}

// llmUsecase is the concrete implementation
type llmUsecase struct {
	toolsManager toolsmanager.ToolsManager
	sessionRepo  mongodb.ChatSessionRepository
	config       *config.Config
}

// NewLLMUsecase creates a new LLM usecase instance
func NewLLMUsecase(
	cfg *config.Config,
	toolsManager toolsmanager.ToolsManager,
	sessionRepo mongodb.ChatSessionRepository,
	endSessionTool end_session.Tool,
	fetchMessagesTool fetch_messages.Tool,
	replyMessageTool reply_message.Tool,
	purchaseIntentTool purchase_intent.Tool,
	listProductsTool list_products.Tool,
) (LLMUsecase, error) {
	util.PanicOnError(
		"register tools",
		toolsManager.AddTool(endSessionTool),
		toolsManager.AddTool(fetchMessagesTool),
		toolsManager.AddTool(replyMessageTool),
		toolsManager.AddTool(purchaseIntentTool),
		toolsManager.AddTool(listProductsTool),
	)

	return &llmUsecase{
		toolsManager: toolsManager,
		sessionRepo:  sessionRepo,
		config:       cfg,
	}, nil
}

// PromptContext holds the data needed for prompt generation
type PromptContext struct {
	SessionID  primitive.ObjectID
	MerchantID primitive.ObjectID
	BuyerID    primitive.ObjectID

	Channel        *models.Channel
	Message        string
	RecentMessages *models.MessageHistory
}

type PromptData struct {
	Channel *models.Channel
	Message string
}

// ProcessMessage processes a message with early validation and deferred expensive operations
func (l *llmUsecase) ProcessMessage(ctx context.Context, chatMode *models.ChatMode, data *PromptContext) error {
	// PHASE 1: Early validation - validate all inputs before any expensive operations
	if err := l.validateInputs(ctx, chatMode, data); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// PHASE 2: Evaluate conditions - check if processing should proceed
	if chatMode.Condition != "" {
		shouldProcess, err := l.evaluateCondition(chatMode.Condition, data)
		if err != nil {
			return fmt.Errorf("failed to evaluate when condition: %w", err)
		}
		if !shouldProcess {
			l.logConditionResult(ctx, chatMode, data, false)
			return nil // Early exit - no error, just don't process
		}
		l.logConditionResult(ctx, chatMode, data, true)
	}

	// PHASE 3: Build prompt - only after validation passes
	prompt, err := l.buildPrompt(chatMode.PromptTemplate, &PromptData{
		Channel: data.Channel,
		Message: data.Message,
	})
	if err != nil {
		return fmt.Errorf("failed to build prompt: %w", err)
	}

	log.Infow(ctx, "Processing message", "chat_mode", chatMode.Name, "session_id", data.SessionID)

	// PHASE 4: Create session context - deferred until after validation
	session, err := l.createSessionContext(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to create session context: %w", err)
	}

	// PHASE 5: Get available tools dynamically from tool manager
	availableTools, err := l.toolsManager.GetToolsForNames(session, chatMode.Tools)
	if err != nil {
		return fmt.Errorf("failed to get tools: %w", err)
	}

	// PHASE 6: Build initial messages
	messages := l.buildInitialMessages(prompt, data, session)

	// PHASE 7: Run AI agent loop
	if err := l.runAgentLoop(ctx, chatMode, messages, availableTools, session); err != nil {
		return err
	}

	log.Infow(ctx, "Agent processing complete", "session_id", data.SessionID)
	return nil
}

// validateInputs performs comprehensive validation before any processing
func (l *llmUsecase) validateInputs(ctx context.Context, chatMode *models.ChatMode, data *PromptContext) error {
	// Validate chat mode
	if chatMode == nil {
		return fmt.Errorf("chat mode is required")
	}
	if chatMode.Name == "" {
		return fmt.Errorf("chat mode name is required")
	}
	if chatMode.Model == "" {
		return fmt.Errorf("chat mode model is required")
	}
	if chatMode.PromptTemplate == "" {
		return fmt.Errorf("chat mode prompt template is required")
	}
	if chatMode.MaxIterations <= 0 {
		return fmt.Errorf("chat mode max iterations must be positive")
	}

	// Validate prompt data
	if data == nil {
		return fmt.Errorf("prompt data is required")
	}
	if data.Message == "" {
		return fmt.Errorf("message is required")
	}

	// Validate available tools exist
	for _, toolName := range chatMode.Tools {
		if !l.toolsManager.HasTool(toolName) {
			log.Warnw(ctx, "Requested tool not available", "tool_name", toolName, "chat_mode", chatMode.Name)
		}
	}

	log.Debugw(ctx, "Input validation passed", "chat_mode", chatMode.Name, "session_id", data.SessionID)
	return nil
}

// createSessionContext creates session context after validation passes
func (l *llmUsecase) createSessionContext(ctx context.Context, data *PromptContext) (toolsmanager.SessionContext, error) {
	gk := genkit.Init(ctx, genkit.WithPlugins(&googlegenai.GoogleAI{
		APIKey: l.config.LLM.GoogleAIAPIKey,
	}))

	// Create session context for tool operations
	session := toolsmanager.NewSessionContext(ctx, toolsmanager.SessionContextConfig{
		Genkit:      gk,
		SessionID:   data.SessionID,
		ChannelID:   data.Channel.ID,
		SessionRepo: l.sessionRepo,
		BuyerID:     data.BuyerID,
		MerchantID:  data.MerchantID,
	})

	return session, nil
}

// buildPrompt generates the prompt from template and data
func (l *llmUsecase) buildPrompt(templateStr string, data *PromptData) (string, error) {
	tmpl, err := template.New("prompt").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	log.Info(context.Background(), "Generated prompt:\n"+buf.String())

	return buf.String(), nil
}

// evaluateCondition evaluates the when condition template
func (l *llmUsecase) evaluateCondition(whenTemplate string, data *PromptContext) (bool, error) {
	if whenTemplate == "" {
		return true, nil // No condition means always process
	}

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

// buildInitialMessages creates the initial message array for the AI
func (l *llmUsecase) buildInitialMessages(prompt string, data *PromptContext, session toolsmanager.SessionContext) []*ai.Message {
	messages := []*ai.Message{
		ai.NewSystemTextMessage(prompt),
	}

	if data.RecentMessages != nil && len(data.RecentMessages.Messages) > 0 {
		messages = l.addRecentMessages(messages, data, session)
	}

	messages = append(messages, ai.NewUserTextMessage(data.Message))
	return messages
}

// addRecentMessages adds recent message history to the conversation
func (l *llmUsecase) addRecentMessages(messages []*ai.Message, data *PromptContext, session toolsmanager.SessionContext) []*ai.Message {
	for _, msg := range data.RecentMessages.Messages {
		if msg.Content == "" {
			continue
		}
		if msg.SenderID == data.BuyerID {
			messages = append(messages, ai.NewUserTextMessage(msg.Content))
		} else {
			messages = append(messages, ai.NewModelTextMessage(msg.Content))
		}
	}

	oldestMessage := data.RecentMessages.Messages[len(data.RecentMessages.Messages)-1]
	session.SaveNextMessageTimestamp(oldestMessage.CreatedAt.UnixMilli())
	return messages
}

// runAgentLoop executes the AI agent conversation loop
func (l *llmUsecase) runAgentLoop(ctx context.Context, chatMode *models.ChatMode, messages []*ai.Message, availableTools []ai.Tool, session toolsmanager.SessionContext) error {
	for i := 0; i < chatMode.MaxIterations; i++ {
		log.Infow(ctx, "Agent iteration", "current", i+1, "max", chatMode.MaxIterations)

		response, err := l.generateResponse(session, chatMode, messages, availableTools)
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

		toolResponseParts, err := l.executeToolRequests(ctx, toolRequests, availableTools, session)
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

// generateResponse generates AI response using Genkit
func (l *llmUsecase) generateResponse(session toolsmanager.SessionContext, chatMode *models.ChatMode, messages []*ai.Message, availableTools []ai.Tool) (*ai.ModelResponse, error) {
	var toolRefs []ai.ToolRef
	for _, tool := range availableTools {
		toolRefs = append(toolRefs, tool)
	}

	return genkit.Generate(session.Context(), session.Genkit(),
		ai.WithMessages(messages...),
		ai.WithModelName(chatMode.Model),
		ai.WithTools(toolRefs...),
	)
}

// executeToolRequests executes the requested tools
func (l *llmUsecase) executeToolRequests(ctx context.Context, toolRequests []*ai.ToolRequest, availableTools []ai.Tool, session toolsmanager.SessionContext) ([]*ai.Part, error) {
	var toolResponseParts []*ai.Part
	for _, req := range toolRequests {
		tool := l.findToolByName(req.Name, availableTools)
		if tool == nil {
			log.Errorw(ctx, "Tool not found", "tool_name", req.Name)
			continue
		}

		output, err := tool.RunRaw(ctx, req.Input)
		if err != nil {
			log.Errorw(ctx, "Tool execution failed", "tool_name", req.Name, "error", err)
			continue
		}
		toolResponseParts = append(toolResponseParts,
			ai.NewToolResponsePart(&ai.ToolResponse{
				Name:   req.Name,
				Ref:    req.Ref,
				Output: output,
			}))
	}
	return toolResponseParts, nil
}

// findToolByName finds a tool by name in the available tools
func (l *llmUsecase) findToolByName(name string, availableTools []ai.Tool) ai.Tool {
	for _, tool := range availableTools {
		if tool.Name() == name {
			return tool
		}
	}
	return nil
}

// logConditionResult logs the result of condition evaluation
func (l *llmUsecase) logConditionResult(ctx context.Context, chatMode *models.ChatMode, data *PromptContext, shouldProcess bool) {
	if shouldProcess {
		log.Infow(ctx, "When condition evaluated to true, proceeding with processing",
			"chat_mode", chatMode.Name)
	} else {
		log.Infow(ctx, "When condition evaluated to false, stopping processing",
			"chat_mode", chatMode)
	}
}
