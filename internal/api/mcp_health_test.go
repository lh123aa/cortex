package api

import (
	"context"
	"strings"
	"testing"

	"github.com/lh123aa/cortex/internal/search"
	"github.com/lh123aa/cortex/internal/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// mockEmbedding is defined in mcp_test.go

func TestMCPHealthTool_WithEmbedding(t *testing.T) {
	dbPath := "file::memory:?cache=shared"
	st, err := storage.NewSQLiteStorage(dbPath)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer st.Close()

	emb := &mockEmbedding{}
	se := search.NewHybridSearchEngine(st, emb)
	logger := zap.NewNop()

	mcpSrv := NewMCPServer(se, st, emb, logger)

	result, _, err := mcpSrv.handleHealthTool(context.Background(), nil, struct{}{})
	if err != nil {
		t.Fatalf("handleHealthTool error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty content")
	}
	text := result.Content[0].(*mcp.TextContent).Text

	if !strings.Contains(text, "Embedding:  mock") {
		t.Fatalf("expected 'Embedding:  mock' in health output, got:\n%s", text)
	}
	if strings.Contains(text, "disabled") {
		t.Fatal("should not contain 'disabled' when embedding is set")
	}
}

func TestMCPHealthTool_WithoutEmbedding(t *testing.T) {
	dbPath := "file::memory:?cache=shared"
	st, err := storage.NewSQLiteStorage(dbPath)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer st.Close()

	se := search.NewHybridSearchEngine(st, nil)
	logger := zap.NewNop()

	mcpSrv := NewMCPServer(se, st, nil, logger)

	result, _, err := mcpSrv.handleHealthTool(context.Background(), nil, struct{}{})
	if err != nil {
		t.Fatalf("handleHealthTool error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty content")
	}
	text := result.Content[0].(*mcp.TextContent).Text

	if !strings.Contains(text, "disabled (FTS5-only)") {
		t.Fatalf("expected 'disabled (FTS5-only)' in health output, got:\n%s", text)
	}
}

func TestMCPHealthTool_ContainsServerInfo(t *testing.T) {
	dbPath := "file::memory:?cache=shared"
	st, err := storage.NewSQLiteStorage(dbPath)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer st.Close()

	se := search.NewHybridSearchEngine(st, nil)
	logger := zap.NewNop()

	mcpSrv := NewMCPServer(se, st, nil, logger)

	result, _, err := mcpSrv.handleHealthTool(context.Background(), nil, struct{}{})
	if err != nil {
		t.Fatalf("handleHealthTool error: %v", err)
	}
	text := result.Content[0].(*mcp.TextContent).Text

	checks := []string{"Cortex MCP Server", "Uptime", "Status", "Running", "Documents"}
	for _, c := range checks {
		if !strings.Contains(text, c) {
			t.Errorf("expected '%s' in health output, got:\n%s", c, text)
		}
	}
}
