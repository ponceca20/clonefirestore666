# Firestore Clone - Production Makefile
# Task 5: Integration and Production Setup

# Variables
BINARY_NAME=firestore-clone
BINARY_PATH=./bin/$(BINARY_NAME)
MAIN_PATH=./cmd/main.go
DOCKER_IMAGE=firestore-clone
DOCKER_TAG=latest

# Go related variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build flags for Windows
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(shell powershell -Command \"Get-Date -Format 'yyyy-MM-dd_HH:mm:ss'\")"

.PHONY: all build clean test coverage deps lint security docker run dev help

# Default target
all: clean deps test build

# Build the application
build:
	@echo "Building application..."
	$(GOBUILD) -o bin/$(BINARY_NAME) -v $(MAIN_PATH)

# Build for Linux
build-linux:
	@echo "Building for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o bin/$(BINARY_UNIX) -v $(MAIN_PATH)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f bin/$(BINARY_NAME)
	rm -f bin/$(BINARY_UNIX)

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -cover ./...
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -tags=integration -v ./...

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) verify

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

# Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	golangci-lint run

# Run the application
run:
	@echo "Running application..."
	$(GOCMD) run $(MAIN_PATH)

# Development mode with hot reload (requires air)
dev:
	@echo "Starting development server..."
	air

# Generate documentation
docs:
	@echo "Generating documentation..."
	$(GOCMD) doc -all ./...

# Security scan (requires gosec)
security:
	@echo "Running security scan..."
	gosec ./...

# Install development tools
install-tools:
	@echo "Installing development tools..."
	$(GOGET) github.com/cosmtrek/air@latest
	$(GOGET) github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.54.2

# Setup development environment
setup: deps install-tools
	@echo "Setting up development environment..."
	cp .env.example .env
	@echo "Please edit .env file with your configuration"

# Start MongoDB with Docker
mongo-up:
	@echo "Starting MongoDB with Docker..."
	docker run -d --name firestore-mongo \
		-p 27017:27017 \
		-e MONGO_INITDB_ROOT_USERNAME=admin \
		-e MONGO_INITDB_ROOT_PASSWORD=Ponceca120 \
		mongo:latest

# Stop MongoDB Docker container
mongo-down:
	@echo "Stopping MongoDB..."
	docker stop firestore-mongo
	docker rm firestore-mongo

# View MongoDB logs
mongo-logs:
	@echo "Viewing MongoDB logs..."
	docker logs -f firestore-mongo

# Database migration (placeholder)
migrate:
	@echo "Running database migrations..."
	# Add migration commands here

# Load test data (placeholder)
seed:
	@echo "Loading test data..."
	# Add data seeding commands here

# Create release build
release: clean test build-linux
	@echo "Creating release build..."
	tar -czf bin/$(BINARY_NAME)-linux-amd64.tar.gz -C bin $(BINARY_UNIX)

# Help command
help:
	@echo "Available commands:"
	@echo "  build          - Build the application"
	@echo "  build-linux    - Build for Linux"
	@echo "  clean          - Clean build artifacts"
	@echo "  test           - Run tests"
	@echo "  coverage       - Run tests with coverage"
	@echo "  test-integration - Run integration tests"
	@echo "  bench          - Run benchmarks"
	@echo "  deps           - Download dependencies"
	@echo "  tidy           - Tidy dependencies"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code"
	@echo "  run            - Run the application"
	@echo "  dev            - Start development server with hot reload"
	@echo "  docs           - Generate documentation"
	@echo "  security       - Run security scan"
	@echo "  install-tools  - Install development tools"
	@echo "  setup          - Setup development environment"
	@echo "  mongo-up       - Start MongoDB with Docker"
	@echo "  mongo-down     - Stop MongoDB Docker container"
	@echo "  mongo-logs     - View MongoDB logs"
	@echo "  migrate        - Run database migrations"
	@echo "  seed           - Load test data"
	@echo "  release        - Create release build"
	@echo "  help           - Show this help message"
