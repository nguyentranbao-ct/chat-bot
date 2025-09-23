# Run tests
test:
	go test -v -race -coverprofile=coverage.out ./...

# Run tests with coverage report
test-coverage: test
	go tool cover -html=coverage.out -o coverage.html

# Run linter
lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5 run --new-from-rev origin/main

# Generate mocks
mock:
	go run github.com/vektra/mockery/v3@v3.5

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

web-dev:
	cd web && npm start

socket-dev:
	cd socket && pnpm dev

# Database commands (for development)
db-up:
	docker compose up -d mongodb

db-down:
	docker compose down
