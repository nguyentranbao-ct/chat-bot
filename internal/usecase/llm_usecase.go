package usecase

import (
	"context"
	"fmt"
	"strings"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/nguyentranbao-ct/chat-bot/internal/config"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/mongodb"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/toolsmanager"
	"github.com/nguyentranbao-ct/chat-bot/pkg/tmplx"
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
) (LLMUsecase, error) {
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

	RoomMember     *models.RoomMember
	Message        string
	RecentMessages *models.MessageHistory
}

type PromptData struct {
	RoomMember *models.RoomMember
	Message    string
}

// ProcessMessage processes a message with early validation and deferred expensive operations
func (l *llmUsecase) ProcessMessage(ctx context.Context, chatMode *models.ChatMode, data *PromptContext) error {
	log.Debugf(ctx, "Starting message processing")
	if err := l.validateInputs(ctx, chatMode, data); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	promptData := &PromptData{
		RoomMember: data.RoomMember,
		Message:    data.Message,
	}
	if chatMode.Condition != "" {
		shouldProcess, err := l.evaluateCondition(chatMode.Condition, promptData)
		if err != nil {
			return fmt.Errorf("failed to evaluate when condition: %w", err)
		}
		if !shouldProcess {
			l.logConditionResult(ctx, chatMode, data, false)
			return nil // Early exit - no error, just don't process
		}
		l.logConditionResult(ctx, chatMode, data, true)
	}

	prompt, err := l.buildPrompt(chatMode.PromptTemplate, promptData)
	if err != nil {
		return fmt.Errorf("failed to build prompt: %w", err)
	}

	session, err := l.createSessionContext(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to create session context: %w", err)
	}

	// PHASE 5: Get available tools dynamically from tool manager
	availableTools, err := l.toolsManager.GetToolsForNames(session, chatMode.Tools)
	if err != nil {
		return fmt.Errorf("failed to get tools: %w", err)
	}
	log.Debugw(ctx, "Available tools for this session", "count", len(availableTools), "tools", chatMode.Tools)

	// PHASE 6: Build initial messages
	messages := l.buildInitialMessages(prompt, data, session)

	// PHASE 7: Run AI agent loop
	if err := l.runAgentLoop(ctx, chatMode, messages, availableTools, session); err != nil {
		return fmt.Errorf("failed during agent iterations: %w", err)
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
		RoomID:      data.RoomMember.RoomID,
		SessionRepo: l.sessionRepo,
		BuyerID:     data.BuyerID,
		MerchantID:  data.MerchantID,
	})

	return session, nil
}

// buildPrompt generates the prompt from template and data
func (l *llmUsecase) buildPrompt(templateStr string, data *PromptData) (string, error) {
	tmpl, err := tmplx.Parse("prompt", templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	buf, err := tmpl.Render(data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	return buf.String(), nil
}

// evaluateCondition evaluates the when condition template
func (l *llmUsecase) evaluateCondition(whenTemplate string, data *PromptData) (bool, error) {
	if whenTemplate == "" {
		return true, nil // No condition means always process
	}
	tmpl, err := tmplx.Parse("when", whenTemplate)
	if err != nil {
		return false, fmt.Errorf("failed to parse when template: %w", err)
	}
	buf, err := tmpl.Render(data)
	if err != nil {
		return false, fmt.Errorf("failed to execute when template: %w", err)
	}
	result := strings.TrimSpace(buf.String()) == "true"
	return result, nil
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

// executeAgentIterations runs the actual iteration loop and returns whether any tools were used
func (l *llmUsecase) runAgentLoop(ctx context.Context, chatMode *models.ChatMode, messages []*ai.Message, availableTools []ai.Tool, session toolsmanager.SessionContext) error {
	response, err := l.generateResponse(session, chatMode, messages, availableTools)
	if err != nil {
		return fmt.Errorf("failed to generate response: %w", err)
	}

	if response.Text() != "" {
		messages = append(messages, ai.NewModelTextMessage(response.Text()))
		log.Debugw(ctx, "AI generated text response", "response", response.Text())
	}

	toolRequests := response.ToolRequests()
	if len(toolRequests) == 0 {
		log.Debugw(ctx, "AI did not use any tools in this iteration")
	}
	log.Debugw(ctx, "Processing tool requests", "count", len(toolRequests))

	// if session.IsEnded() {
	// 	log.Debugw(ctx, "Session has been terminated by tool execution, ending conversation")
	// }
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
		ai.WithMaxTurns(chatMode.MaxIterations),
	)
}

// logConditionResult logs the result of condition evaluation
func (l *llmUsecase) logConditionResult(ctx context.Context, chatMode *models.ChatMode, data *PromptContext, shouldProcess bool) {
	if shouldProcess {
		log.Debugw(ctx, "When condition evaluated to true, proceeding with processing",
			"chat_mode", chatMode.Name)
	} else {
		log.Debugw(ctx, "When condition evaluated to false, stopping processing",
			"chat_mode", chatMode)
	}
}
