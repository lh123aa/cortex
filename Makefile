.PHONY: all build test clean format tools

BINARY_NAME=cortex

all: format test build

build:
	@echo "Tidying dependencies..."
	@go mod tidy
	@echo "Building Cortex..."
	@go build -o bin/$(BINARY_NAME) ./cmd/cortex/main.go

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
