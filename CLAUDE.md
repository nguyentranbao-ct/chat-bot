# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

**Build and Run:**

- `make build` - Build the binary to ./bin/chat-bot
- `make run` - Build and run the application
- `make dev` - Start development server with hot reload (requires air)

**Testing:**

- `make test` - Run all tests with race detection and coverage
- `make test-coverage` - Generate HTML coverage report

**Code Quality:**

- `make lint` - Run golangci-lint
- `make fmt` - Format code with go fmt
- `make vet` - Run go vet
- `make mock` - Generate mocks using mockery (outputs to ./internal/mocks)

**Dependencies:**

- `make deps` - Download and tidy Go modules
- `make setup` - Install development tools and dependencies
- `make install-tools` - Install air, mockery, and golangci-lint

**Database (Development):**

- `make db-up` - Start MongoDB with Docker Compose
- `make db-down` - Stop MongoDB

## Architecture Overview

This is a Go-based chat-bot service that implements a conversational AI agent for product-selling channels. The service integrates with chat-api for data retrieval and uses Firebase Genkit for Go for LLM-powered conversations.

**Key Technologies:**

- Go with Uber FX for dependency injection
- Echo for HTTP server
- Firebase Genkit for Go (LLM framework)
- MongoDB 8.0 for data persistence
- Kafka for message event consumption
- Chat-API integration for external communication

**Architecture Layers:**

- **cmd/** - Application entry point with Cobra CLI
- **internal/app** - Application setup and DI configuration with Uber FX
- **internal/server** - HTTP endpoints, middleware, and server setup
- **internal/usecase** - Business logic layer
- **internal/repo** - Data access layer with external API clients and database implementations
  - **mongodb/** - MongoDB repository implementations
  - **chatapi/** - Chat-API client
  - **chotot/** - Chotot API client
  - **tools/** - LLM tool implementations (PurchaseIntent, ReplyMessage, FetchMessages, EndSession, ListProducts)
  - **toolsmanager/** - Tools management and session context
- **internal/kafka** - Kafka consumer for message processing
- **internal/models** - Data structures and DTOs
- **internal/config** - Configuration management
- **internal/types** - Shared type definitions
- **pkg/** - Shared packages (validator, ctxval, tmplx, util)

**Message Triggers:**

- `POST /api/v1/messages` - HTTP endpoint for receiving message events with required headers:
  - `x-project-uuid`: Project UUID
  - `Service: chat-bot`
- **Kafka Consumer** - Consumes messages from Kafka topic:
  - Brokers: `kafka-08.ct.dev:9200`
  - Topic: `chat.event.messages`
  - Channel whitelist configured in settings to control which channels trigger processing

**Core APIs:**

*User Management:*
- `POST /api/v1/users` - Create a new user
- `GET /api/v1/users/:id` - Get a user by ID
- `PUT /api/v1/users/:id` - Update a user
- `DELETE /api/v1/users/:id` - Delete a user
- `POST /api/v1/users/:id/attributes` - Set a user attribute
- `GET /api/v1/users/:id/attributes` - Get all attributes for a user
- `GET /api/v1/users/:id/attributes/:key` - Get a specific user attribute
- `DELETE /api/v1/users/:id/attributes/:key` - Remove a user attribute

*Authentication (JWT):*
- `POST /api/v1/auth/login` - Email-based login, returns JWT token
- `GET /api/v1/auth/me` - Get current user profile (requires auth)
- `PUT /api/v1/auth/profile` - Update user profile (requires auth)
- `POST /api/v1/auth/logout` - Revoke JWT token (requires auth)

*Chat Management (requires auth):*
- `GET /api/v1/chat/rooms` - Get user's chat rooms
- `GET /api/v1/chat/rooms/:id/members` - Get room members
- `POST /api/v1/chat/rooms/:id/messages` - Send message to room
- `GET /api/v1/chat/rooms/:id/messages` - Get room message history
- `GET /api/v1/chat/rooms/:id/events` - Get room activity events
- `POST /api/v1/chat/rooms/:id/read` - Mark room messages as read

*Profile Management (requires auth):*
- `GET /api/v1/profile/attributes` - Get partner-specific attributes
- `PUT /api/v1/profile/attributes` - Update partner integration settings

**External Services:**

- Chat-API endpoint: `chat-api.chat.gke1.ct.dev`

## Critical Architecture Concepts

### Partner Abstraction System

**Multi-platform support through unified partner interface:**
- **Current**: Chotot integration (chat-api + product services)
- **Framework Ready**: Facebook, WhatsApp, Telegram expansion
- **Partner Registry**: `internal/repo/partners/` - detects partner from message source and routes to appropriate implementation
- **Partner Interface**: Standardized methods (GetRoomInfo, FetchMessages, SendMessage, GetUserProducts)
- **Partner Detection**: Based on `source.name` field in incoming messages

```go
// Partner routing example
partner := partnerRegistry.GetPartner(message.Partner.Name) // "chotot", "facebook", etc.
roomInfo := partner.GetRoomInfo(ctx, message.Partner.RoomID)
```

### Denormalized Data Architecture

**Single RoomMember collection design for performance:**
- **Core Principle**: Avoid expensive joins by storing room + member data together
- **Collection**: `room_members` contains user info + complete room context + unread counts
- **Benefits**: Single query for complete chat context, optimal for read-heavy chat operations
- **Trade-off**: Room info updates require multi-document updates (handled in usecase layer)

```go
// Get user's rooms with full context in one query
roomMembers := repository.GetUserRooms(userID) // Returns room + member + unread data
rooms := convertToClientRooms(roomMembers)     // Simple conversion, no joins needed
```

### Loop Prevention

**Prevents infinite loops using logical user data:**
- **Internal User Check**: Looks up sender by partner-specific ID (e.g., `chotot_id` attribute)
- **Internal Flag**: Uses `user.is_internal` field to identify bot/system users
- **Logic**: Skip processing messages from internal users to prevent response loops
- **Efficiency**: Uses existing user data instead of separate tracking collection

### Authentication System

**JWT-based authentication with token management:**
- **Login**: Email-only login (no password), returns JWT token
- **Token Storage**: `auth_tokens` collection stores hashed tokens for revocation
- **Middleware**: `authMiddleware` validates JWT and sets user in context
- **Protected Routes**: Most `/api/v1/chat/*` and `/api/v1/profile/*` endpoints require auth
- **Context Access**: `user := c.Get("user").(*models.User)` in controllers

**Core Components:**

- **Chat Modes**: Configurable system prompts and LLM settings stored in MongoDB
- **Tools System**: AI tools for conversation capabilities (detailed below)
- **Session Management**: Tracks conversation sessions and activities
- **Firebase Genkit Integration**: Handles LLM calls with tool execution loops
- **User Management System**: CRUD operations for users and their attributes
- **Kafka Consumer**: Processes messages from Kafka topic as an alternative to HTTP endpoints

### LLM Tools System

**Tool-based AI architecture with 5 core tools:**

*Communication Tools:*
- **ReplyMessage**: Send responses to users via partner APIs (platform-specific formatting)
- **FetchMessages**: Retrieve conversation history for context (partner-specific implementation)

*Business Logic Tools:*
- **PurchaseIntent**: Log buy signals with confidence percentage (stores in `purchase_intents` collection)
- **ListProducts**: Search user's product listings via partner product APIs

*Session Management:*
- **EndSession**: Terminate conversation and mark session as ended

**Tool Execution Pattern:**
1. LLM analyzes conversation and determines required tools
2. Tools execute in parallel when possible
3. Tool results appended to conversation context
4. Process repeats until no tools needed or max iterations reached

**Data Flow:**

1. Receive message via HTTP API or Kafka topic consumption
2. Validate message and check channel whitelist (for Kafka events)
3. Gather channel info and message history from chat-api
4. Build prompt with system context and history
5. Run Genkit AI flow with LLM and tools
6. Execute tools (send replies, log activities, etc.)
7. Iterate until completion or max iterations

**MongoDB Collections:**

*Core Data (Denormalized):*
- `room_members` - **Main collection**: User + room + unread data in single document for performance
- `users` - User profiles and basic information
- `user_attributes` - Partner-specific user data (chotot_id, whatsapp_phone_number_id, etc.)

*Chat & AI:*
- `chat_modes` - LLM configuration templates and prompts (system + tools + model settings)
- `chat_sessions` - Conversation lifecycle tracking (active/ended/abandoned)
- `chat_activities` - Tool execution and action logs
- `chat_messages` - Message storage with metadata
- `purchase_intents` - Buy signal analytics with confidence percentages

*System:*
- `auth_tokens` - JWT token management for revocation

**Key Collection Patterns:**
- `room_members`: Denormalized design eliminates joins, contains room context + member data + unread counts
- `user_attributes`: Key-value pairs with tags for partner-specific settings (enables partner lookup and loop prevention)
- `users`: `is_internal` flag identifies bot users for loop prevention
- `chat_sessions`: Links to room_members and chat_modes for complete conversation context

The service uses FX for clean dependency injection and follows a layered architecture with clear separation between HTTP handling, business logic, and data access.

## Key Development Patterns

### Common Tasks & Patterns

**Adding a new Partner:**
1. Implement partner interface in `internal/repo/partners/`
2. Add partner registration in partner registry
3. Update partner detection logic for message routing
4. Add partner-specific user attributes if needed

**Working with Room Data:**
```go
// Always use denormalized room_members for chat operations
roomMembers := chatRepo.GetUserRooms(ctx, userID)  // Single query, complete data
// Avoid separate room + member queries - use the denormalized pattern
```

**Message Processing Development:**
1. Always check for internal users first: `isInternalUser(ctx, senderID, partnerType)` using `user_attributes` lookup
2. Skip processing messages from internal users (prevents response loops)
3. Use partner interface for external calls: `partner.GetRoomInfo()`, `partner.FetchMessages()`
4. Follow tool execution pattern: LLM → Parse Tools → Execute → Append Results → Repeat

**Authentication in Controllers:**
```go
func (c *controller) ProtectedEndpoint(ctx echo.Context) error {
    user := ctx.Get("user").(*models.User)  // Set by authMiddleware
    // user is guaranteed to be valid if middleware passed
}
```

**Partner Attribute Management:**
```go
// Use constants for attribute keys
models.PartnerAttrChototID           // "chotot_id"
models.PartnerAttrWhatsAppPhoneNumberID  // "whatsapp_phone_number_id"
// Store with appropriate tags: ["chotot", "primary"] or ["whatsapp", "sensitive"]
```

**Error Handling Patterns:**
- Validation errors: Return 400 with field details
- External API failures: Retry with exponential backoff (transient) or fail gracefully
- Database errors: Check for unique constraint violations, return appropriate HTTP status
- LLM errors: Fallback to backup provider or timeout handling

### File Organization Guidelines

**When adding new functionality:**
- **Controllers**: HTTP request/response handling only (`internal/server/`)
- **Usecases**: Business logic and orchestration (`internal/usecase/`)
- **Repositories**: Data access and external API calls (`internal/repo/`)
- **Models**: Data structures and validation (`internal/models/`)

**Testing Strategy:**
- Mock external dependencies (partners, databases)
- Test business logic in usecases with mocked repos
- Test HTTP endpoints with real Echo context
- Use `make mock` to regenerate mocks after interface changes

## Coding Principles

### Architecture & Import Hierarchy

The codebase follows a strict layered architecture with controlled import directions:

**Import Flow (from cmd to deepest):**

```
cmd/ → internal/app/ → internal/server/ → internal/usecase/ → internal/repo/ → internal/models/ → internal/config/ → internal/types/ → pkg/
     ↓              ↓                    ↓                    ↓              ↓              ↓              ↓              ↓
  main.go      app.go (FX DI)      controller.go        usecase.go     repositories    models.go     config.go     types.go     utilities
               providers.go       middleware/          llm_usecase.go mongodb/       errors.go      validators    shared       validator/
               (FX modules)        server.go            user_usecase.go chatapi/       events.go                   types        ctxval/
                                                                 tools/          message.go                               tmplx/
                                                                 toolsmanager/   user.go                                   util/
                                                                 chotot/         object_id.go
                                                                 kafka/
```

**Key Principles:**

- **Dependency Injection:** Uber FX manages all dependencies with clear provider/consumer relationships
- **Layer Separation:** Each layer has a single responsibility and imports only from lower layers
- **Repository Pattern:** Data access abstracted through repository interfaces
- **Clean Architecture:** Business logic isolated in usecase layer, independent of frameworks

### Code Structure Standards

- **Single Responsibility:** Each file and function has one clear purpose
- **File Size Limits:** Keep files under 500 lines, functions under 25 lines
- **DRY Principle:** Extract reusable logic, avoid code duplication
- **Configuration over Hard-coding:** All values configurable through environment or config files
- **Self-documenting Code:** Prefer clear code over comments, document public APIs

### Go-Specific Practices

- **Error Handling:** Explicit error returns, no panics in production code
- **Context Usage:** Pass context through call chains for cancellation and tracing
- **Interface Design:** Small, focused interfaces following Go idioms
- **Testing:** Unit tests with mocks, integration tests for critical paths
- **Linting:** Follow golangci-lint rules and go vet recommendations

## Detailed Documentation

For comprehensive information beyond this quick reference:

- **[docs/overview.md](./docs/overview.md)** - Complete system overview, architecture summary, and getting started guide
- **[docs/architecture.md](./docs/architecture.md)** - Detailed technical architecture, design patterns, and scalability considerations
- **[docs/erd.md](./docs/erd.md)** - Entity relationship diagrams, complete database schema, and data model details
- **[docs/dataflow.md](./docs/dataflow.md)** - Message processing flows with flowcharts and sequence diagrams
- **[docs/apis.md](./docs/apis.md)** - Complete API reference with request/response examples and authentication details

**Quick Context Guidelines:**
- Use this CLAUDE.md for day-to-day development decisions and architectural understanding
- Reference detailed docs for complete schemas, API specifications, and implementation examples
- When adding new features, follow the patterns established in existing code and document major changes
