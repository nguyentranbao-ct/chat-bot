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
	golangci-lint run

# Generate mocks
mock:
	mockery --all --output=./internal/mocks --case=underscore

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Download dependencies
tidy:
	go mod tidy

# Run the application
dev:
	go run .

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/vektra/mockery/v3@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

# Database commands (for development)
db-up:
	@echo "Starting MongoDB with Docker Compose..."
	docker compose up -d mongodb

db-down:
	@echo "Stopping MongoDB..."
	docker compose down

# Full development setup
setup: tidy install-tools
	@echo "Development environment setup complete!"
