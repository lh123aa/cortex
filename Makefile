.PHONY: all build test clean format tools

BINARY_NAME=cortex

all: format test build

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags="-X github.com/lh123aa/cortex/internal/api.Version=$(VERSION) -X github.com/lh123aa/cortex/internal/api.Commit=$(COMMIT) -X github.com/lh123aa/cortex/internal/api.Date=$(DATE)"

build:
	@echo "Tidying dependencies..."
	@go mod tidy
	@echo "Building Cortex $(VERSION)..."
	@go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/cortex/main.go

test:
	@echo "Running tests..."
	@go test -v ./...

format:
	@echo "Formatting code..."
	@go fmt ./...

clean:
	@echo "Cleaning up..."
	@rm -rf bin/
	@rm -f cortex.db
	@rm -f cortex.db-shm
	@rm -f cortex.db-wal

run-mcp: build
	@echo "Running MCP server..."
	@./bin/$(BINARY_NAME) mcp
