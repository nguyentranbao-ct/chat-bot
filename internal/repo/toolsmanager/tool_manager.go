package toolsmanager

import (
	"context"
	"fmt"
	"sync"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/firebase/genkit/go/ai"
)

// toolsManager is the concrete implementation of ToolsManager
type toolsManager struct {
	tools map[string]Tool
	mutex sync.RWMutex
}

// NewToolsManager creates a new instance of ToolsManager
func NewToolsManager() ToolsManager {
	return &toolsManager{
		tools: make(map[string]Tool),
	}
}

// AddTool registers a new tool with the manager
func (tm *toolsManager) AddTool(tool Tool) error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	name := tool.Name()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	if _, exists := tm.tools[name]; exists {
		return fmt.Errorf("tool with name '%s' is already registered", name)
	}

	tm.tools[name] = tool
	log.Infof(context.Background(), "Tool registered: %s - %s", name, tool.Description())
	return nil
}

// ExecuteTool executes a tool by name with the given arguments
func (tm *toolsManager) ExecuteTool(ctx context.Context, toolName string, args interface{}, session SessionContext) (interface{}, error) {
	tm.mutex.RLock()
	tool, exists := tm.tools[toolName]
	tm.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("tool not found: %s", toolName)
	}

	log.Infow(ctx, "Executing tool", "tool_name", toolName)

	result, err := tool.Execute(ctx, args, session)
	if err != nil {
		log.Errorw(ctx, "Tool execution failed", "tool_name", toolName, "error", err)
		return nil, fmt.Errorf("tool execution failed: %w", err)
	}

	log.Infow(ctx, "Tool executed successfully", "tool_name", toolName)
	return result, nil
}

// GetAvailableTools returns a list of all registered tool names
func (tm *toolsManager) GetAvailableTools() []string {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	tools := make([]string, 0, len(tm.tools))
	for name := range tm.tools {
		tools = append(tools, name)
	}
	return tools
}

// GetToolsForNames returns Genkit tools for the specified tool names
func (tm *toolsManager) GetToolsForNames(s SessionContext, toolNames []string) ([]ai.Tool, error) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	var genkitTools []ai.Tool
	for _, toolName := range toolNames {
		tool, exists := tm.tools[toolName]
		if !exists {
			log.Warnw(s.Context(), "Requested tool not found", "tool_name", toolName)
			continue
		}

		genkitTool := tool.GetGenkitTool(s, s.Genkit())
		if genkitTool != nil {
			genkitTools = append(genkitTools, genkitTool)
		}
	}

	return genkitTools, nil
}

// HasTool checks if a tool with the given name is registered
func (tm *toolsManager) HasTool(toolName string) bool {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	_, exists := tm.tools[toolName]
	return exists
}
