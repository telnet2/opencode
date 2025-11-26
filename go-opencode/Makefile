# OpenCode Server Makefile

.PHONY: build run test clean fmt lint deps

# Build variables
VERSION ?= 0.1.0
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Binary output
BIN_DIR := bin
BINARY := $(BIN_DIR)/opencode

# Default target
all: build

# Build the server
build:
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BINARY) ./cmd/opencode-server

# Run the server
run: build
	$(BINARY) --port 8080

# Run with a specific directory
run-dir: build
	$(BINARY) --port 8080 --directory $(DIR)

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Install dependencies
deps:
	go mod download
	go mod tidy

# Clean build artifacts
clean:
	rm -rf $(BIN_DIR)
	rm -f coverage.out coverage.html

# Development mode - rebuild and run on file changes
dev:
	@which air > /dev/null || go install github.com/cosmtrek/air@latest
	air

# Generate mocks for testing
mocks:
	@which mockgen > /dev/null || go install github.com/golang/mock/mockgen@latest
	go generate ./...

# Print version
version:
	@echo $(VERSION)

# Help
help:
	@echo "OpenCode Server Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build        - Build the server binary"
	@echo "  make run          - Build and run the server"
	@echo "  make test         - Run tests"
	@echo "  make test-coverage- Run tests with coverage report"
	@echo "  make fmt          - Format code"
	@echo "  make lint         - Lint code"
	@echo "  make deps         - Install/update dependencies"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make dev          - Run in development mode with hot reload"
	@echo "  make help         - Show this help"
