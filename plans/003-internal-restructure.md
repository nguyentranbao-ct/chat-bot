# Internal Directory Restructuring Plan

## Overview

Restructure the `internal/` directory to follow a clear top-to-bottom architectural flow with proper separation of concerns.

## Current Structure Analysis

```
internal/
├── app/           # FX dependency injection setup ✓
├── client/        # External API clients (chat-api) → Move to repo/
├── config/        # Configuration loading → Move to pkg/
├── handler/       # HTTP request handlers → Rename to server/
├── kafka/         # Kafka message handlers ✓
├── llm/           # LLM service and tools → Move to repo/
├── models/        # Data models → Move to pkg/
├── repository/    # Data access layer → Rename to repo/
├── service/       # Business services → Merge into usecase/
└── usecase/       # Domain logic ✓
```

## Proposed New Structure

### Top-to-Bottom Flow

```
cmd/                  # Commands and CLI tools (root level)
internal/
├── app/              # FX providers and dependency injection
├── config/           # Configuration structs and loading (internal only)
├── server/           # HTTP request handlers and routes
├── kafka/            # Kafka message handlers and consumers
├── usecase/          # Domain logic interfaces and implementations
└── repo/             # Data access and external integrations
    ├── mongodb/      # MongoDB repositories
    ├── chatapi/      # Chat-API client (external service calls)
    └── llm/          # LLM service and tools (external AI calls)
```

### Shared Packages (Move to pkg/)

```
pkg/
├── models/           # Data models and types
└── logger/           # Shared logging utilities (if needed)
```

## Detailed Migration Plan

### 1. Create New Directory Structure

```bash
# cmd already exists at root level
mkdir -p internal/server
mkdir -p internal/repo/mongodb
mkdir -p internal/repo/chatapi
mkdir -p internal/repo/llm
mkdir -p pkg/models
```

### 2. File Migration Mapping

#### Keep as-is (no changes)
- `internal/app/` → `internal/app/` (FX providers and DI setup)
- `internal/config/` → `internal/config/` (stays internal, not shared)
- `internal/kafka/` → `internal/kafka/` (Kafka handlers)
- `internal/usecase/` → `internal/usecase/` (Domain logic)

#### Rename and move
- `internal/handler/` → `internal/server/`
  - `message_handler.go` → `server/message_handler.go`
  - `routes.go` → `server/routes.go`
  - `middleware.go` → `server/middleware.go`

- `internal/repository/` → `internal/repo/mongodb/`
  - `interfaces.go` → Delete (interfaces go at top of implementation files)
  - `mongodb/` → `repo/mongodb/` (MongoDB implementations with interfaces at top)

- `internal/client/` → `internal/repo/chatapi/`
  - All chat-api client code moves to `repo/chatapi/`

- `internal/llm/` → `internal/repo/llm/`
  - All LLM service and tools move to `repo/llm/`

#### Move to shared packages
- `internal/models/` → `pkg/models/`

#### Merge into existing
- `internal/service/` → Merge into `internal/usecase/`
  - `whitelist.go` → `usecase/whitelist_usecase.go`
  - `chat_mode_initializer.go` → `usecase/chat_mode_usecase.go`

### 3. Import Path Updates

#### Before
```go
"github.com/nguyentranbao-ct/chat-bot/internal/models"
"github.com/nguyentranbao-ct/chat-bot/internal/config"
"github.com/nguyentranbao-ct/chat-bot/internal/handler"
"github.com/nguyentranbao-ct/chat-bot/internal/repository"
"github.com/nguyentranbao-ct/chat-bot/internal/client"
"github.com/nguyentranbao-ct/chat-bot/internal/llm"
```

#### After
```go
"github.com/nguyentranbao-ct/chat-bot/pkg/models"
"github.com/nguyentranbao-ct/chat-bot/internal/config"
"github.com/nguyentranbao-ct/chat-bot/internal/server"
"github.com/nguyentranbao-ct/chat-bot/internal/repo"
"github.com/nguyentranbao-ct/chat-bot/internal/repo/chatapi"
"github.com/nguyentranbao-ct/chat-bot/internal/repo/llm"
```

### 4. Interface Exposure and Method Argument Patterns

#### Interface Exposure Pattern
All handlers must expose interfaces at the top of implementation files (no separate interfaces.go):

```go
// server/message_handler.go
type Handler interface {
    ProcessMessage(ctx context.Context, message *models.IncomingMessage) error
    Health(ctx context.Context) error
}

type messageHandler struct {  // private struct
    usecase usecase.MessageUsecase
}

func NewHandler(usecase usecase.MessageUsecase) Handler {  // returns interface
    return &messageHandler{usecase: usecase}
}
```

#### Method Argument Pattern (Max 3 args)
For methods with >3 arguments, use struct:

```go
// Before (too many args)
func (u *messageUsecase) ProcessMessage(
    ctx context.Context,
    message *models.IncomingMessage,
    channelID string,
    userID string,
    timestamp time.Time,
) error

// After (struct for >3 args)
type ProcessMessageArgs struct {
    Message   *models.IncomingMessage
    ChannelID string
    UserID    string
    Timestamp time.Time
}

func (u *messageUsecase) ProcessMessage(ctx context.Context, args ProcessMessageArgs) error
```

#### Constructor Pattern (OK to have many args)
```go
// Constructor functions can have many arguments
func NewMessageUsecase(
    chatModeRepo repo.ChatModeRepository,
    sessionRepo repo.ChatSessionRepository,
    activityRepo repo.ChatActivityRepository,
    chatAPIClient chatapi.Client,
    llmService llm.Service,
    whitelistRepo repo.WhitelistRepository,
) usecase.MessageUsecase {
    return &messageUsecase{
        chatModeRepo:  chatModeRepo,
        sessionRepo:   sessionRepo,
        activityRepo:  activityRepo,
        chatAPIClient: chatAPIClient,
        llmService:    llmService,
        whitelistRepo: whitelistRepo,
    }
}
```

### 5. App Provider Updates

Update `internal/app/app.go` to reflect new import paths and interface usage:

```go
// Update imports
import (
    "github.com/nguyentranbao-ct/chat-bot/pkg/models"
    "github.com/nguyentranbao-ct/chat-bot/internal/config"
    "github.com/nguyentranbao-ct/chat-bot/internal/server"
    "github.com/nguyentranbao-ct/chat-bot/internal/usecase"
    "github.com/nguyentranbao-ct/chat-bot/internal/repo"
    "github.com/nguyentranbao-ct/chat-bot/internal/repo/mongodb"
    "github.com/nguyentranbao-ct/chat-bot/internal/repo/chatapi"
    "github.com/nguyentranbao-ct/chat-bot/internal/repo/llm"
)

// All providers return interfaces
func NewMessageUsecase(repos *Repositories, chatAPIClient chatapi.Client, llmService llm.Service) usecase.MessageUsecase {
    return usecase.NewMessageUsecase(
        repos.ChatMode,
        repos.Session,
        repos.Activity,
        chatAPIClient,
        llmService,
        repos.Whitelist,
    )
}

func NewMessageHandler(messageUsecase usecase.MessageUsecase) server.Handler {
    return server.NewHandler(messageUsecase)
}
```

## Benefits of New Structure

### 1. Clear Architectural Flow
- **cmd** → **app** → **server/kafka** → **usecase** → **repo**
- No circular dependencies
- Clear separation of concerns

### 2. External Dependencies Grouped
- All external integrations in `repo/` (MongoDB, Chat-API, LLM)
- Clear boundary between domain logic and external systems

### 3. Shared Code in pkg/
- Models accessible by all layers
- Config stays internal (not shared)
- Follows Go project layout standards

### 4. Interface Pattern Benefits
- Interfaces defined at top of implementation files
- No separate interfaces.go files to maintain
- Clear contract definition with implementation
- Easy mocking for tests

### 5. Consistent Function Signatures
- Max 3 arguments for methods (struct if more needed)
- Constructor functions can have many arguments
- Clear documentation of required dependencies

### 6. Better Testability
- Clear dependency injection points
- Easy to mock entire dependency groups
- Isolated unit testing per layer

## Implementation Steps

1. **Create new directories** and move files
2. **Update import paths** across all files
3. **Refactor function signatures** to use struct arguments
4. **Update app providers** for new structure
5. **Run tests** to ensure no regressions
6. **Update documentation** and README

## Validation

After restructuring:
- [ ] All tests pass
- [ ] No circular import dependencies
- [ ] Clear top-to-bottom flow maintained
- [ ] Function arguments follow struct pattern
- [ ] External dependencies isolated in repo/