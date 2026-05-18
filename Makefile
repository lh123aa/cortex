.PHONY: all build test clean format tools vet lint check security run-mcp run-serve dev

BINARY_NAME=cortex

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags="-X github.com/lh123aa/cortex/internal/api.Version=$(VERSION) -X github.com/lh123aa/cortex/internal/api.Commit=$(COMMIT) -X github.com/lh123aa/cortex/internal/api.Date=$(DATE)"

all: format vet test build

build:
	@echo "Tidying dependencies..."
	@go mod tidy
	@echo "Building Cortex $(VERSION)..."
	@go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/cortex/main.go

test:
	@echo "Running tests..."
	@go test -v -count=1 ./...

test-short:
	@echo "Running tests (short mode)..."
	@go test -count=1 ./...

test-race:
	@echo "Running tests with race detector..."
	@go test -race -count=1 ./...

test-cover:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -5

format:
	@echo "Formatting code..."
	@go fmt ./...

vet:
	@echo "Running go vet..."
	@go vet ./...

lint:
	@echo "Running staticcheck (if installed)..."
	@which staticcheck 2>/dev/null && staticcheck ./... || echo "staticcheck not installed, skipping"

security:
	@echo "Running govulncheck (if installed)..."
	@which govulncheck 2>/dev/null && govulncheck ./... || echo "govulncheck not installed, skipping"

check: vet lint security
	@echo "All checks passed!"

# ci-check: 模拟 GitHub CI quality job 环境，推送前必跑
ci-check:
	@echo "=== CI Check: go vet ==="
	go vet ./...
	@echo ""
	@echo "=== CI Check: go test (CGO_ENABLED=1) ==="
	CGO_ENABLED=1 go test -count=1 -timeout=300s ./...
	@echo ""
	@echo "=== CI Check: go test (CGO_ENABLED=0) ==="
	CGO_ENABLED=0 go test -count=1 -timeout=300s ./...
	@echo ""
	@echo "✅ All CI checks passed!"

clean:
	@echo "Cleaning up..."
	@rm -rf bin/
	@rm -f cortex.db
	@rm -f cortex.db-shm
	@rm -f cortex.db-wal
	@rm -f coverage.out

run-mcp:
	@go run $(LDFLAGS) ./cmd/cortex/main.go mcp

run-serve:
	@go run $(LDFLAGS) ./cmd/cortex/main.go serve

dev: format vet
	@echo "Starting MCP server in dev mode..."
	@go run $(LDFLAGS) ./cmd/cortex/main.go mcp
