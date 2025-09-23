# Chat-Bot System Overview

## Table of Contents

- [System Introduction](#system-introduction)
- [Architecture Summary](#architecture-summary)
- [Key Features](#key-features)
- [Technology Stack](#technology-stack)
- [Documentation Structure](#documentation-structure)
- [Getting Started](#getting-started)

## System Introduction

The Chat-Bot Service is a sophisticated conversational AI system designed to act as an intelligent merchant assistant across multiple messaging platforms. Built with a **partner abstraction layer**, the system currently supports Chotot's chat-api and is architected for seamless expansion to Facebook Messenger, WhatsApp, Telegram, and other platforms.

### Core Capabilities

- **Multi-Platform Integration**: Unified interface supporting current Chotot integration with ready framework for Facebook, WhatsApp, and other partners
- **AI-Powered Conversations**: Firebase Genkit for Go with tool-based AI flows for intelligent customer interactions
- **Dual Message Processing**: Synchronous HTTP API and asynchronous Kafka topic consumption with built-in deduplication
- **Denormalized Performance**: Single RoomMember collection architecture for optimal read performance
- **Real-time Communication**: WebSocket support through Node.js Socket.IO server and React frontend

## Architecture Summary

The system implements a clean layered architecture with strict separation of concerns:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Chat Clients  │    │  Partner APIs   │    │  External LLM   │
│  (Chotot, etc.) │    │ (Product APIs)  │    │   (Genkit)      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Chat-Bot Service                             │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐ │
│  │   HTTP API  │  │   Kafka     │  │    Partner Registry     │ │
│  │  Messages   │  │  Consumer   │  │   (Chotot, Future)      │ │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘ │
│         │                 │                      │             │
│         ▼                 ▼                      ▼             │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │              Business Logic Layer                          │ │
│  │     (Message Processing, LLM Orchestration)                │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                              │                                 │
│                              ▼                                 │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                   Data Layer                               │ │
│  │        (MongoDB, External APIs, Tools)                     │ │
│  └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────┐
│    MongoDB      │
│  (Collections)  │
└─────────────────┘
```

### Key Architectural Decisions

1. **Partner Abstraction Layer**: Enables multi-platform support through unified interfaces
2. **Denormalized RoomMember Design**: Single collection eliminates expensive joins for chat operations
3. **Message Deduplication**: Prevents infinite loops using external message ID tracking
4. **Tool-Based AI Architecture**: Modular LLM tools for extensible conversation capabilities

## Key Features

### Multi-Platform Partner Support
- **Current**: Full Chotot integration (chat-api + product services)
- **Framework Ready**: Facebook Messenger, WhatsApp, Telegram expansion
- **Unified Interface**: Partner detection and routing with consistent API

### Advanced Chat Management
- **Denormalized Room Architecture**: Single RoomMember collection for optimal performance
- **Real-time Unread Tracking**: Efficient message state management
- **Session Management**: Conversation tracking with activity logging
- **Message History**: Complete conversation context for AI processing

### AI-Powered Conversations
- **Firebase Genkit Integration**: Enterprise-grade LLM orchestration
- **Tool-Based Architecture**: Modular conversation capabilities:
  - `PurchaseIntent`: Intent logging and analytics
  - `ReplyMessage`: Platform-specific message sending
  - `FetchMessages`: Conversation history retrieval
  - `EndSession`: Conversation termination
  - `ListProducts`: Product search and recommendations
- **Configurable Chat Modes**: MongoDB-based prompt templates and model settings

### Scalable Message Processing
- **Dual Input Channels**: HTTP API and Kafka consumer support
- **Message Deduplication**: Loop prevention with external ID tracking
- **Async Processing**: Non-blocking Kafka-based message handling
- **Partner Filtering**: Whitelist-based channel control for Kafka events

## Technology Stack

### Backend Services
- **Go 1.25**: Primary backend language with modern features
- **Uber FX**: Dependency injection and application lifecycle management
- **Echo Framework**: High-performance HTTP server and routing
- **Firebase Genkit for Go**: LLM framework with tool orchestration
- **MongoDB 8.0**: Document database with optimized collections
- **Kafka**: Event-driven message processing and pub-sub messaging

### Frontend & Real-time
- **React with TypeScript**: Modern frontend application
- **Node.js Socket.IO**: Real-time WebSocket server
- **Web Frontend**: Interactive chat interface for testing and management

### External Integrations
- **Chotot Chat-API**: Room management and message handling
- **Chotot Product API**: Product search and user inventory
- **LLM Providers**: Multi-provider support through Firebase Genkit

### Development & DevOps
- **Docker Compose**: Development environment orchestration
- **Golang Templates**: Prompt template system for AI configurations
- **golangci-lint**: Code quality and style enforcement
- **Air**: Hot reload development server

## Documentation Structure

This documentation suite provides comprehensive coverage of the Chat-Bot system:

### Core Documentation
- **[overview.md](./overview.md)** *(This file)*: System introduction and high-level architecture
- **[architecture.md](./architecture.md)**: Detailed technical architecture, patterns, and design decisions

### Data & System Design
- **[erd.md](./erd.md)**: Entity Relationship Diagrams showing data model relationships
- **[dataflow.md](./dataflow.md)**: Message flow diagrams with flowcharts and sequence diagrams
- **[apis.md](./apis.md)**: Complete API reference with endpoints, request/response formats

### Additional References
- **[CLAUDE.md](../CLAUDE.md)**: Development commands, coding standards, and project conventions
- **[Makefile](../Makefile)**: Build commands and development workflows

## Getting Started

### Prerequisites
- Go 1.25 or higher
- MongoDB 8.0
- Docker and Docker Compose
- Node.js (for Socket.IO server)

### Quick Start

1. **Setup Development Environment**
   ```bash
   make setup          # Install development tools
   make db-up          # Start MongoDB with Docker Compose
   ```

2. **Build and Run**
   ```bash
   make build          # Build the chat-bot binary
   make run            # Run the application
   # OR
   make dev            # Start with hot reload (requires air)
   ```

3. **Verify Installation**
   ```bash
   curl http://localhost:8080/health
   # Expected: {"status":"healthy","service":"chat-bot"}
   ```

### Development Workflow

1. **Code Quality**
   ```bash
   make fmt            # Format code
   make lint           # Run linting
   make vet            # Run go vet
   ```

2. **Testing**
   ```bash
   make test           # Run all tests with coverage
   make test-coverage  # Generate HTML coverage report
   ```

3. **Mocking**
   ```bash
   make mock           # Generate mocks for interfaces
   ```

### Configuration

Key configuration areas:
- **Environment Variables**: Database connections, API keys, partner settings
- **Chat Modes**: MongoDB-based prompt templates and LLM configurations
- **Partner Settings**: Chotot API endpoints and authentication
- **Kafka Settings**: Broker connections and topic configurations

### Next Steps

1. **Explore the API**: Review [apis.md](./apis.md) for complete endpoint documentation
2. **Understand Data Flow**: Check [dataflow.md](./dataflow.md) for message processing flows
3. **Review Architecture**: See [architecture.md](./architecture.md) for detailed technical design
4. **Database Schema**: Examine [erd.md](./erd.md) for data model relationships

### Support & Development

- **Build Commands**: Defined in `Makefile` with comprehensive development workflow
- **Code Standards**: Follows Go best practices with strict import hierarchy
- **Testing Strategy**: Unit tests with mocks, integration tests for critical flows
- **Documentation**: Self-documenting code with API documentation for public interfaces

The Chat-Bot service represents a production-ready, scalable foundation for multi-platform conversational AI with room for extensive customization and platform expansion.