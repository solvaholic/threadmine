# ThreadMine Makefile
# Automates building with FTS5 support and other common tasks

BINARY_NAME=mine
BUILD_DIR=.
GO=go
GOFLAGS=-tags "fts5"
LDFLAGS=-s -w

.PHONY: all build test clean install help

# Default target
all: build

# Build the binary with FTS5 support
build:
	@echo "Building $(BINARY_NAME) with FTS5 support..."
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/mine
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Run tests
test:
	@echo "Running tests..."
	$(GO) test $(GOFLAGS) -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test $(GOFLAGS) -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BUILD_DIR)/$(BINARY_NAME)
	rm -f coverage.out coverage.html
	@echo "Clean complete"

# Install to $GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME) to $(GOPATH)/bin..."
	$(GO) install $(GOFLAGS) -ldflags "$(LDFLAGS)" ./cmd/mine
	@echo "Install complete"

# Run with example command
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME) --help

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	golangci-lint run

# Show help
help:
	@echo "ThreadMine Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build          Build the binary with FTS5 support (default)"
	@echo "  test           Run tests"
	@echo "  test-coverage  Run tests with coverage report"
	@echo "  clean          Remove build artifacts"
	@echo "  install        Install to \$$GOPATH/bin"
	@echo "  run            Build and run with --help"
	@echo "  fmt            Format code with go fmt"
	@echo "  lint           Run golangci-lint (requires golangci-lint)"
	@echo "  help           Show this help message"
	@echo ""
	@echo "The build always includes -tags \"fts5\" for full-text search support"
