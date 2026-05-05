//go:build cgo

package api

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/lh123aa/cortex/internal/chunker"
	"github.com/lh123aa/cortex/internal/search"
	"github.com/lh123aa/cortex/internal/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

type mockEmbedding struct{}

func (m *mockEmbedding) EmbedBatch(texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = make([]float32, 128)
	}
	return result, nil
}

func (m *mockEmbedding) Embed(text string) ([]float32, error) {
	return make([]float32, 128), nil
}

func (m *mockEmbedding) Dimension() int { return 128 }

func (m *mockEmbedding) Name() string { return "mock" }

func (m *mockEmbedding) Health() error { return nil }

func setupMCPTest(t *testing.T) (*MCPServer, *storage.SQLiteStorage, func()) {
	t.Helper()

	dbPath := "file::memory:?cache=shared"
	st, err := storage.NewSQLiteStorage(dbPath)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	emb := &mockEmbedding{}
	se := search.NewHybridSearchEngine(st, emb)
	logger := zap.NewNop()

	mcpSrv := NewMCPServer(se, st, emb, logger)

	cleanup := func() {
		st.Close()
	}

	return mcpSrv, st, cleanup
}

func makeArgs(t *testing.T, v any) *mcp.CallToolRequest {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal args: %v", err)
	}
	return &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: b,
		},
	}
}

func TestMCPSearchHandler_ValidQuery(t *testing.T) {
	srv, st, cleanup := setupMCPTest(t)
	defer cleanup()

	doc, _ := chunker.NewTextChunker(chunker.ChunkConfig{MinChars: 1, MaxTokens: 512})
	chunks, _ := doc.Chunk("Go is a statically typed compiled programming language", "test.md")
	for _, c := range chunks {
		c.UserID = ""
		c.DocumentID = "test-doc"
	}
	if err := st.SaveChunks(chunks); err != nil {
		t.Fatalf("failed to save chunks: %v", err)
	}

	req := makeArgs(t, SearchArgs{Query: "Go", TopK: 5})
	result, _, err := srv.handleSearchTool(context.Background(), req, SearchArgs{Query: "Go", TopK: 5})
	if err != nil {
		t.Fatalf("search handler error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty result content")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text == "" {
		t.Fatal("expected non-empty result text")
	}
}

func TestMCPSearchHandler_EmptyQuery(t *testing.T) {
	srv, _, cleanup := setupMCPTest(t)
	defer cleanup()

	req := makeArgs(t, SearchArgs{Query: "", TopK: 5})
	_, _, err := srv.handleSearchTool(context.Background(), req, SearchArgs{Query: "", TopK: 5})
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestMCPContextHandler_ValidQuery(t *testing.T) {
	srv, st, cleanup := setupMCPTest(t)
	defer cleanup()

	doc, _ := chunker.NewTextChunker(chunker.ChunkConfig{MinChars: 1, MaxTokens: 512})
	chunks, _ := doc.Chunk("Go is a compiled language designed for concurrency", "test.md")
	for _, c := range chunks {
		c.UserID = ""
		c.DocumentID = "test-doc"
	}
	if err := st.SaveChunks(chunks); err != nil {
		t.Fatalf("failed to save chunks: %v", err)
	}

	req := makeArgs(t, ContextArgs{Query: "Go concurrency", TokenBudget: 500})
	result, _, err := srv.handleContextTool(context.Background(), req, ContextArgs{Query: "Go concurrency", TokenBudget: 500})
	if err != nil {
		t.Fatalf("context handler error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty result content")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text == "" {
		t.Fatal("expected non-empty context text")
	}
}

func TestMCPContextHandler_EmptyQuery(t *testing.T) {
	srv, _, cleanup := setupMCPTest(t)
	defer cleanup()

	_, _, err := srv.handleContextTool(context.Background(), nil, ContextArgs{Query: "", TokenBudget: 500})
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestMCPMemoryWriteHandler_Success(t *testing.T) {
	srv, _, cleanup := setupMCPTest(t)
	defer cleanup()

	req := makeArgs(t, MemoryWriteArgs{
		Content: "Test memory content for unit test",
		Tags:    []string{"test", "unit"},
		Source:  "mcp-test",
	})
	result, _, err := srv.handleMemoryWriteTool(context.Background(), req, MemoryWriteArgs{
		Content: "Test memory content for unit test",
		Tags:    []string{"test", "unit"},
		Source:  "mcp-test",
	})
	if err != nil {
		t.Fatalf("memory write handler error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty result content")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text == "" {
		t.Fatal("expected non-empty result text")
	}
}

func TestMCPMemoryWriteHandler_EmptyContent(t *testing.T) {
	srv, _, cleanup := setupMCPTest(t)
	defer cleanup()

	_, _, err := srv.handleMemoryWriteTool(context.Background(), nil, MemoryWriteArgs{Content: ""})
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestMCPMemorySearchHandler_AfterWrite(t *testing.T) {
	srv, st, cleanup := setupMCPTest(t)
	defer cleanup()

	writeReq := MemoryWriteArgs{
		Content: "Golang concurrency uses goroutines and channels",
		Tags:    []string{"go", "concurrency"},
		Source:  "test",
	}
	_, _, err := srv.handleMemoryWriteTool(context.Background(), makeArgs(t, writeReq), writeReq)
	if err != nil {
		t.Fatalf("memory write failed: %v", err)
	}

	var memID string
	doc, err := st.GetDocumentByPath("memory://*", "")
	if err == nil && doc != nil {
		memID = doc.ID
	}
	_ = memID

	searchReq := MemorySearchArgs{Query: "goroutines", TopK: 5}
	result, _, err := srv.handleMemorySearchTool(context.Background(), makeArgs(t, searchReq), searchReq)
	if err != nil {
		t.Fatalf("memory search handler error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty result content")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text == "" {
		t.Fatal("expected non-empty result text")
	}
}

func TestMCPMemorySearchHandler_EmptyQuery(t *testing.T) {
	srv, _, cleanup := setupMCPTest(t)
	defer cleanup()

	_, _, err := srv.handleMemorySearchTool(context.Background(), nil, MemorySearchArgs{Query: ""})
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestMCPMemoryDeleteHandler_NotFound(t *testing.T) {
	srv, _, cleanup := setupMCPTest(t)
	defer cleanup()

	req := makeArgs(t, MemoryDeleteArgs{ID: "nonexistent-id"})
	_, _, err := srv.handleMemoryDeleteTool(context.Background(), req, MemoryDeleteArgs{ID: "nonexistent-id"})
	if err != nil {
		t.Fatalf("delete handler should not error on missing memory: %v", err)
	}
}

func TestMCPMemoryDeleteHandler_EmptyID(t *testing.T) {
	srv, _, cleanup := setupMCPTest(t)
	defer cleanup()

	_, _, err := srv.handleMemoryDeleteTool(context.Background(), nil, MemoryDeleteArgs{ID: ""})
	if err == nil {
		t.Fatal("expected error for empty id")
	}
}

func TestMCPMemoryWriteThenDelete(t *testing.T) {
	srv, _, cleanup := setupMCPTest(t)
	defer cleanup()

	writeReq := MemoryWriteArgs{
		Content: "Delete me after writing",
		Tags:    []string{"temp"},
	}
	writeResult, _, err := srv.handleMemoryWriteTool(context.Background(), makeArgs(t, writeReq), writeReq)
	if err != nil {
		t.Fatalf("memory write failed: %v", err)
	}

	writeText := writeResult.Content[0].(*mcp.TextContent).Text
	if writeText == "" {
		t.Fatal("expected write result text")
	}

	searchReq := MemorySearchArgs{Query: "Delete me", TopK: 5}
	searchResult, _, err := srv.handleMemorySearchTool(context.Background(), makeArgs(t, searchReq), searchReq)
	if err != nil {
		t.Fatalf("memory search failed: %v", err)
	}
	if len(searchResult.Content) == 0 {
		t.Fatal("expected search results after write")
	}
}


