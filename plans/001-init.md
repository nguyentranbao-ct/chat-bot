# Chat-Bot Service Implementation Plan

## Overview

Implement a Go-based chat-bot service that acts as an AI-powered seller in product chat channels, integrating with chat-api and using Firebase Genkit for Go for conversational AI.

## Phase 1: Project Foundation

1. **Initialize Go module and project structure**

   - Create `go.mod` with proper module name
   - Set up directory structure: `cmd/`, `internal/`, `pkg/`, `configs/`, `scripts/`
   - Create `Makefile` with standard Go commands (test, lint, build, mock)

2. **Core dependencies setup**
   - Firebase Genkit for Go (LLM framework)
   - Echo framework (HTTP server)
   - Uber FX (dependency injection)
   - Docker compose with MongoDB driver
   - Cobra for CLI commands
   - Configuration management (github.com/caarlos0/env/v11)

## Phase 2: Configuration & Database

3. **Configuration system**

   - Environment-based config structure
   - MongoDB connection settings
   - Chat-API endpoints and credentials
   - LLM provider configurations

4. **MongoDB integration**
   - Database models for: chat modes, sessions, activities
   - Repository pattern implementation
   - Connection pool and error handling

## Phase 3: Core API Structure

5. **HTTP server setup**

   - Echo server with middleware (logging, CORS, validation)
   - Route definitions for `/api/v1/messages`
   - Request validation and error handling
   - Header validation (`x-project-uuid`, `Service`)

6. **External API client**
   - Chat-API client for message and channel operations
   - HTTP client with retry logic and timeouts
   - Request/response models matching documented schemas

## Phase 4: LLM Integration

7. **Firebase Genkit setup**

   - Initialize Genkit with multiple LLM providers (OpenAI, Anthropic, Google AI)
   - Model configuration from MongoDB chat modes
   - Token limits and iteration controls

8. **Tool system implementation**
   - `PurchaseIntent`: Purchase intent logging
   - `ReplyMessage`: Send messages via chat-api
   - `FetchMessages`: Retrieve conversation history
   - `EndSession`: Terminate AI flow

## Phase 5: Business Logic

9. **Message processing usecase**

   - Validate incoming requests
   - Gather channel info and message history
   - Build dynamic prompts using Go templates
   - Execute AI flows with tool integration

10. **Prompt template system**
    - Go template engine for dynamic prompts
    - Template context building from channel data
    - Mode-specific prompt customization

## Phase 6: Testing & Quality

11. **Comprehensive testing**

    - Unit tests with testify and mocks (mockery)
    - Integration tests with real MongoDB
    - API endpoint testing with httptest
    - Mock external services (chat-api, LLM providers)

12. **Code quality tools**
    - Golangci-lint configuration
    - Pre-commit hooks
    - Coverage reporting
    - Documentation generation

## Phase 7: Deployment & Operations

13. **Containerization**

    - Dockerfile for production builds
    - Docker Compose for local development
    - Health check endpoints

14. **Monitoring & Logging**
    - Structured logging with request tracing
    - Metrics collection for AI operations
    - Error tracking and alerting

## Key Technical Decisions

- **Architecture**: Clean architecture with repository pattern
- **DI Framework**: Uber FX for dependency injection
- **HTTP Framework**: Echo for performance and middleware support
- **LLM Framework**: Firebase Genkit for Go for tool integration
- **Database**: MongoDB with official Go driver
- **Testing**: Testify framework with mockery for mocks
- **Validation**: Go validator with custom rules

## Success Criteria

- API accepts chat messages and triggers AI responses
- Tools execute successfully (send replies, log purchases)
- Dynamic prompts render with channel context
- Comprehensive test coverage (>80%)
- Performance meets requirements (sub-second response times)
- Error handling provides meaningful feedback

## Implementation Order

1. Project setup and dependencies
2. Configuration and MongoDB
3. Basic HTTP API structure
4. Chat-API client integration
5. Firebase Genkit and tools
6. Business logic and prompt templates
7. Testing and quality assurance
8. Deployment and monitoring

## Next Steps

Execute Phase 1 to establish the project foundation and core structure.
