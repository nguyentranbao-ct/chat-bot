package toolsmanager

import (
	"context"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Tool represents a generic tool that can be executed
type Tool interface {
	// Name returns the tool's unique identifier
	Name() string
	// Description returns a human-readable description of what the tool does
	Description() string
	// GetGenkitTool returns the Firebase Genkit tool definition for AI integration
	GetGenkitTool(session SessionContext, g *genkit.Genkit) ai.Tool
}

// ToolsManager manages tool registration and execution
type ToolsManager interface {
	// AddTool registers a new tool with the manager
	AddTool(tool Tool) error
	// GetAvailableTools returns a list of all registered tool names
	GetAvailableTools() []string
	// GetToolsForNames returns Genkit tools for the specified tool names
	GetToolsForNames(session SessionContext, toolNames []string) ([]ai.Tool, error)
	// HasTool checks if a tool with the given name is registered
	HasTool(toolName string) bool
}

// SessionContext provides access to session-related data and operations
type SessionContext interface {
	Context() context.Context
	Genkit() *genkit.Genkit

	// Session identification
	GetSessionID() primitive.ObjectID
	GetChannelID() primitive.ObjectID
	GetBuyerID() primitive.ObjectID
	GetMerchantID() primitive.ObjectID

	// Session control
	EndSession() error
	IsEnded() bool

	// Message tracking
	GetNextMessageTimestamp() *int64
	SaveNextMessageTimestamp(timestamp int64)
}

// ToolExecutionResult represents the result of tool execution
type ToolExecutionResult struct {
	Success bool        `json:"success"`
	Result  interface{} `json:"result,omitempty"`
	Error   string      `json:"error,omitempty"`
}
