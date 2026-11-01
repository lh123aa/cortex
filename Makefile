.PHONY: all build test clean format tools vet lint check security run-mcp run-serve dev dist clean-dist

BINARY_NAME=cortex

# 版本号唯一来源：internal/version（构建时用 -ldflags 注入）。
# 发布 CI 会显式传入 VERSION=vX.Y.Z；本地默认取最近的 git tag。
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags="-s -w -X github.com/lh123aa/cortex/internal/version.Version=$(VERSION) -X github.com/lh123aa/cortex/internal/version.Commit=$(COMMIT) -X github.com/lh123aa/cortex/internal/version.Date=$(DATE)"

# dist 目标使用的 6 个发布平台（与 .github/workflows/build.yml 保持一致）。
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64
DIST_DIR := dist

all: format vet test build

build:
	@echo "Tidying dependencies..."
	@go mod tidy
	@echo "Building Cortex $(VERSION)..."
	@go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/cortex/main.go

# dist: 本地模拟发布产物 —— 交叉编译 6 个平台到 dist/，并生成 SHA256SUMS。
# 产物命名与 .github/workflows/build.yml 及 scripts/install.* 保持一致：
#   dist/cortex-<os>-<arch>[.exe]
dist:
	@echo "Building Cortex $(VERSION) for $(words $(PLATFORMS)) platforms..."
	@rm -rf $(DIST_DIR) && mkdir -p $(DIST_DIR)
	@set -e; \
	for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		out="$(DIST_DIR)/cortex-$$os-$$arch$$ext"; \
		echo "  -> $$out"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build $(LDFLAGS) -trimpath -o "$$out" ./cmd/cortex; \
	done
	@cd $(DIST_DIR) && rm -f SHA256SUMS && \
	if command -v sha256sum >/dev/null 2>&1; then \
		sha256sum cortex-* > SHA256SUMS; \
	else \
		shasum -a 256 cortex-* > SHA256SUMS; \
	fi
	@echo "SHA256SUMS:" && cat $(DIST_DIR)/SHA256SUMS

clean-dist:
	@rm -rf $(DIST_DIR)

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
	@if command -v powershell.exe > /dev/null 2>&1; then \
		powershell.exe -ExecutionPolicy Bypass -File scripts/ci-check.ps1; \
	else \
		echo "=== CI Check: go vet ==="; \
		go vet ./...; \
		echo ""; \
		echo "=== CI Check: go test (CGO_ENABLED=1) ==="; \
		CGO_ENABLED=1 go test -count=1 -timeout=300s ./...; \
		echo ""; \
		echo "=== CI Check: go test (CGO_ENABLED=0) ==="; \
		CGO_ENABLED=0 go test -count=1 -timeout=300s ./...; \
		echo ""; \
		echo "✅ All CI checks passed!"; \
	fi

clean:
	@echo "Cleaning up..."
	@rm -rf bin/
	@rm -rf $(DIST_DIR)
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
