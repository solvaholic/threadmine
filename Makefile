# ThreadMine Makefile
# Automates building with FTS5 support and other common tasks

BINARY_NAME=mine
BUILD_DIR=.
GO?=go
GOFLAGS?=
TAGS=
LDFLAGS=-s -w
override REQUIRED_GOFLAGS = $(GOFLAGS) -tags=$(if $(TAGS),$(TAGS)$(comma))fts5 -mod=readonly
comma := ,
export CGO_ENABLED ?= 1
export GOTOOLCHAIN ?= local

.PHONY: all build test test-coverage clean install run fmt lint help check-go check-fmt verify-modules vet test-race build-all smoke check vuln

# Default target
all: build

# Build the binary with FTS5 support
build: check-go
	@echo "Building $(BINARY_NAME) with FTS5 support..."
	$(GO) build $(REQUIRED_GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/mine
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Run tests
test: check-go
	@echo "Running tests..."
	$(GO) test $(REQUIRED_GOFLAGS) -v ./...

# Run tests with coverage
test-coverage: check-go
	@echo "Running tests with coverage..."
	$(GO) test $(REQUIRED_GOFLAGS) -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BUILD_DIR)/$(BINARY_NAME)
	rm -f coverage.out coverage.html
	@echo "Clean complete"

# Install to $GOPATH/bin
install: check-go
	@echo "Installing $(BINARY_NAME) to $(GOPATH)/bin..."
	$(GO) install $(REQUIRED_GOFLAGS) -ldflags "$(LDFLAGS)" ./cmd/mine
	@echo "Install complete"

# Run with example command
run: build
	@echo "Running $(BINARY_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME) --help

# Format code
fmt: check-go
	@echo "Formatting code..."
	$(GO) fmt $(REQUIRED_GOFLAGS) ./...

check-go:
	@command -v "$(GO)" >/dev/null 2>&1 || { echo "Go executable not found: $(GO). Set GO to an installed Go 1.26.0+ executable." >&2; exit 1; }
	@test "$$($(GO) env CGO_ENABLED)" = 1 || { echo "CGO_ENABLED=1 is required for SQLite FTS5 and race tests." >&2; exit 1; }
	@compiler="$$($(GO) env CC)" && set -- $$compiler && command -v "$$1" >/dev/null 2>&1 || { echo "C compiler not found. Install a C toolchain or set CC." >&2; exit 1; }

check-fmt: check-go
	@root="$$($(GO) env GOROOT)" && sh scripts/check-fmt.sh "$$root/bin/gofmt"

verify-modules: check-go
	$(GO) mod verify

vet: check-go
	$(GO) vet $(REQUIRED_GOFLAGS) ./...

test-race: check-go
	$(GO) test $(REQUIRED_GOFLAGS) -race ./...

build-all: check-go
	$(GO) build $(REQUIRED_GOFLAGS) ./...

smoke: check-go
	@set -eu; tmp=$$(mktemp -d); trap 'rm -f "$$tmp/mine"; rmdir "$$tmp"' 0; \
		trap 'exit 1' 1 2 3 15; \
		$(GO) build $(REQUIRED_GOFLAGS) -o "$$tmp/mine" ./cmd/mine; \
		"$$tmp/mine" --help

# Recursive invocations keep the aggregate ordered, even with make -j.
check: check-go
	$(MAKE) check-fmt
	$(MAKE) verify-modules
	$(MAKE) vet
	$(MAKE) test-race
	$(MAKE) build-all
	$(MAKE) smoke

vuln: check-go
	GOFLAGS='$(REQUIRED_GOFLAGS)' $(GO) run golang.org/x/vuln/cmd/govulncheck@v1.9.0 -tags fts5 -show verbose ./cmd/mine

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
	@echo "  check          Check format, modules, vet, race tests, all packages and CLI help"
	@echo "  check-fmt      Check formatting without modifying Go sources"
	@echo "  verify-modules Verify cached module checksums"
	@echo "  vet            Run go vet on all packages"
	@echo "  test-race      Run tests with the race detector"
	@echo "  build-all      Build every package, including demos"
	@echo "  smoke          Build a temporary CLI and run --help"
	@echo "  vuln           Opt-in pinned govulncheck scan (requires network)"
	@echo "  clean          Remove build artifacts"
	@echo "  install        Install to \$$GOPATH/bin"
	@echo "  run            Build and run with --help"
	@echo "  fmt            Format code with go fmt"
	@echo "  lint           Run golangci-lint (requires golangci-lint)"
	@echo "  help           Show this help message"
	@echo ""
	@echo "Go commands require CGO, include fts5 and use -mod=readonly."
	@echo "Set GO for another executable, TAGS for extra tags, GOFLAGS for other flags."
	@echo "GOTOOLCHAIN defaults to local; Go 1.26.0+ and a C compiler are required."
