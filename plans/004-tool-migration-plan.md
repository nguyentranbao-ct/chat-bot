# Tool Migration Plan

## Current Issues

- Too many duplicate tool handling mechanisms (ExecuteTool method vs Genkit tools)
- Tool definitions scattered across multiple files
- Tool arguments handled inconsistently
- Heavy Firebase Genkit logic mixed with business logic in `genkit.go`

## Migration Strategy

### 1. Restructure Tool Architecture

**Target Structure:**

```
internal/
  usecase/
    llm_usecase.go     # Move genkit logic from repo/llm
  repo/
    toolsmanager/
      tool_manager.go  # Central tool registry and execution
      types.go         # Common interfaces
      session_context.go
    tools/
      purchaseIntent/
        purchase_intent.go   # Self-contained tool with name/description
      replyMessage/
        reply_message.go
      fetchMessages/
        fetch_messages.go
      endSession/
        end_session.go
```

### 2. Implement Generic MCP-Style Tool Manager

- Create a `ToolsManager` interface with `AddTool()` method
- **Generic design**: Tool manager should not know about specific tools (PurchaseIntent, ReplyMessage, etc.)
- Tools register themselves with name, description, and handler
- Single `ExecuteTool()` method using map lookup instead of switch statement
- Tools handle their own argument parsing (remove centralized type conversion)
- Tool manager provides only registry and execution infrastructure

### 3. Move Genkit Logic to Usecase Layer

- Move `genkit.go` logic to `internal/usecase/llm_usecase.go`
- **Generic design**: Genkit service should not hardcode specific tool definitions
- Extract Firebase Genkit orchestration from repository layer
- Separate AI flow logic from tool management
- Keep repository layer focused on data access only
- Genkit service receives available tools from tool manager dynamically

### 4. Tool Self-Registration Pattern

- Each tool defines its own name, description, and schema
- Tools auto-register with ToolsManager during initialization
- Remove hardcoded tool lists and switch statements
- Use dependency injection for tool dependencies

### 5. Add Validation and Precheck Requirements

- **Move all validation to the beginning**: Validate headers, metadata, and requirements before any setup
- **Defer expensive operations**: Don't create sessions or initialize LLM until all prechecks pass
- **Early exit pattern**: Return immediately if validation fails, whitelist checks fail, or conditions not met
- Validate chat mode exists and has required fields before proceeding
- Check user permissions and room access before session creation

### 6. Implementation Steps

1. Create new `toolsmanager` package with generic interfaces
2. Refactor existing tools to self-contained packages with self-registration
3. Create `llm_usecase.go` with moved Genkit logic (generic, no hardcoded tools)
4. Implement early validation and precheck pattern
5. Update dependency injection in `app.go`
6. Remove old tool handling code
7. Update tests and ensure compatibility

## Benefits

- Eliminates code duplication
- Makes adding new tools trivial
- Cleaner separation of concerns
- More maintainable and testable code
- Follows established patterns from sketch

## Current Code Analysis

### Existing Tool Structure

- Tools located in `internal/repo/llm/tools/`
- Each tool has its own file with Execute method
- ToolsManager in `internal/repo/llm/tools.go` handles coordination
- Genkit integration in `internal/repo/llm/genkit.go` (378 lines)

### Key Files to Migrate

1. `internal/repo/llm/genkit.go` → `internal/usecase/llm_usecase.go`
2. `internal/repo/llm/tools.go` → `internal/toolsmanager/tool_manager.go`
3. Individual tool files → Self-contained packages

### Dependencies to Consider

- Firebase Genkit integration
- MongoDB repositories
- Chat API client
- Session context management
- Uber FX dependency injection

## Migration Phases

### Phase 1: Create Foundation

- [ ] Create `internal/toolsmanager` package with generic interfaces
- [ ] Define ToolsManager interface (no hardcoded tool knowledge)
- [ ] Create tool registration mechanism
- [ ] Implement early validation and precheck patterns

### Phase 2: Refactor Tools

- [ ] Convert each tool to self-contained package
- [ ] Implement self-registration pattern
- [ ] Update tool argument handling

### Phase 3: Move Genkit Logic

- [ ] Create `internal/usecase/llm_usecase.go` (generic, no specific tool references)
- [ ] Move AI flow orchestration logic with early validation
- [ ] Implement deferred session creation pattern
- [ ] Update service interfaces

### Phase 4: Integration

- [ ] Update dependency injection in `app.go`
- [ ] Remove old tool handling code
- [ ] Update tests and documentation

### Phase 5: Cleanup

- [ ] Archive old files
- [ ] Update imports across codebase
- [ ] Verify all functionality works
