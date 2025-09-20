# Go Chat-Bot Service Makefile
.PHONY: help build test lint mock clean run dev fmt vet deps kafka-enable kafka-disable kafka-status

# Variables
BINARY_NAME=chat-bot
MAIN_PATH=./cmd/chat-bot
BUILD_DIR=./bin

# Default target
help:
	@echo "Available commands:"
	@echo "  build     - Build the binary"
	@echo "  test      - Run all tests"
	@echo "  lint      - Run golangci-lint"
	@echo "  mock      - Generate mocks using mockery"
	@echo "  fmt       - Format code"
	@echo "  vet       - Run go vet"
	@echo "  deps      - Download dependencies"
	@echo "  run       - Run the application"
	@echo "  dev       - Run in development mode with hot reload"
	@echo "  clean     - Clean build artifacts"
	@echo "  kafka-enable  - Enable Kafka consumer"
	@echo "  kafka-disable - Disable Kafka consumer"
	@echo "  kafka-status  - Show Kafka configuration status"

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

# Run tests with coverage report
test-coverage: test
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html

# Run linter
lint:
	@echo "Running golangci-lint..."
	golangci-lint run

# Generate mocks
mock:
	@echo "Generating mocks..."
	mockery --all --output=./internal/mocks --case=underscore

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME)

# Development mode (requires air for hot reload)
dev:
	@echo "Starting development server..."
	air

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/cosmtrek/air@latest
	go install github.com/vektra/mockery/v2@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Docker commands
docker-build:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME) .

docker-run:
	@echo "Running Docker container..."
	docker run --rm -p 8080:8080 $(BINARY_NAME)

# Database commands (for development)
db-up:
	@echo "Starting MongoDB with Docker Compose..."
	docker-compose up -d mongodb

db-down:
	@echo "Stopping MongoDB..."
	docker-compose down

# Full development setup
setup: deps install-tools
	@echo "Development environment setup complete!"

# Kafka commands
kafka-enable:
	@echo "Enabling Kafka consumer..."
	@echo "Set KAFKA_ENABLED=true in your environment"
	@echo "Example: export KAFKA_ENABLED=true"

kafka-disable:
	@echo "Disabling Kafka consumer..."
	@echo "Set KAFKA_ENABLED=false in your environment"
	@echo "Example: export KAFKA_ENABLED=false"

kafka-status:
	@echo "Kafka Configuration Status:"
	@echo "KAFKA_ENABLED: $${KAFKA_ENABLED:-false}"
	@echo "KAFKA_BROKERS: $${KAFKA_BROKERS:-kafka-08.ct.dev:9200}"
	@echo "KAFKA_TOPIC: $${KAFKA_TOPIC:-chat.event.messages}"
	@echo "KAFKA_CONSUMER_GROUP: $${KAFKA_CONSUMER_GROUP:-chat-bot-consumers}"
	@echo "KAFKA_CHANNEL_WHITELIST: $${KAFKA_CHANNEL_WHITELIST:-<empty>}"