package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/lh123aa/cortex/internal/models"
	"github.com/lh123aa/cortex/internal/rag"
	"github.com/lh123aa/cortex/internal/search"
	"github.com/lh123aa/cortex/internal/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

const (
	MCPProtocolVersion = "2025-06-18"
	ServerName         = "cortex"
)

type MCPServer struct {
	server  *mcp.Server
	search  *search.HybridSearchEngine
	rag     *rag.RAGBuilder
	storage storage.Storage
	logger  *zap.Logger
}

func NewMCPServer(se *search.HybridSearchEngine, st storage.Storage, log *zap.Logger) *MCPServer {
	s := &MCPServer{
		search:  se,
		rag:     rag.NewRAGBuilder(se),
		storage: st,
		logger:  log,
	}

	// 实例化 MCP Server
	s.server = mcp.NewServer(&mcp.Implementation{
		Name:    ServerName,
		Version: "v1.0.0",
	}, &mcp.ServerOptions{
		ProtocolVersion: MCPProtocolVersion,
	})

	s.registerTools()
	return s
}

// truncateText 截断显示，防止控制台文本爆炸
func truncateText(text string, n int) string {
	if len(text) > n {
		return text[:n] + "..."
	}
	return text
}

func (s *MCPServer) registerTools() {
	// cortex_search: 提供语义搜索
	s.server.AddTool(mcp.Tool{
		Name:        "cortex_search",
		Description: "Search the local knowledge base (cortex) using vector and fts and return relevant chunks",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"query": map[string]any{
					Type:        "string",
					Description: "The exact search query to lookup",
				},
				"top_k": map[string]any{
					Type:        "integer",
					Description: "Number of results to return",
					Default:     10,
				},
			},
			Required: []string{"query"},
		},
	}, s.handleSearch)

	// cortex_context: 组装 RAG
	s.server.AddTool(mcp.Tool{
		Name:        "cortex_context",
		Description: "Assemble relevant information within a specific token budget limit strictly",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"query": map[string]any{
					Type:        "string",
					Description: "The query to build context upon",
				},
				"token_budget": map[string]any{
					Type:        "integer",
					Description: "Allowed max tokens",
					Default:     1500,
				},
			},
			Required: []string{"query"},
		},
	}, s.handleContext)

	// P3-5: 注册 tools/list 处理器（Agent 发现可用工具）
	s.server.AddHandler(mcp.Handler{
		Method: "tools/list",
		Handler: func(ctx context.Context, req map[string]any) (any, error) {
			return map[string]any{
				"tools": []map[string]any{
					{
						"name":        "cortex_search",
						"description": "Search the local knowledge base using hybrid vector and full-text search",
					},
					{
						"name":        "cortex_context",
						"description": "Assemble RAG context within a token budget",
					},
				},
			}, nil
		},
	})
}

func (s *MCPServer) handleSearch(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("cortex error: query is strictly required for tool")
	}

	topK := 10
	if tk, ok := args["top_k"].(float64); ok {
		topK = int(tk)
	}

	opts := models.SearchOptions{TopK: topK, Mode: "hybrid"}
	results, err := s.search.Search(ctx, query, opts)
	if err != nil {
		s.logger.Error("mcp tool execution failed on search", zap.Error(err))
		return nil, fmt.Errorf("search operational error: %v", err)
	}

	var sb strings.Builder
	for i, r := range results {
		// P0-2: 追溯实际文件路径，而非仅 DocumentID
		docPath := r.Chunk.DocumentID // fallback
		if doc, err := s.storage.GetDocumentByID(r.Chunk.DocumentID); err == nil && doc != nil {
			docPath = doc.Path
		}
		sb.WriteString(fmt.Sprintf("[%d] Score: %.3f\nPath: %s\nSection: %s\n\n%s\n---\n", i+1, r.Score, docPath, r.Chunk.HeadingPath, truncateText(r.Chunk.ContentRaw, 300)))
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func (s *MCPServer) handleContext(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("cortex error: query requried")
	}

	budget := 1500
	if b, ok := args["token_budget"].(float64); ok {
		budget = int(b)
	}

	opts := models.SearchOptions{TopK: 50, Mode: "hybrid"}
	c, err := s.rag.BuildContext(ctx, query, budget, opts)
	if err != nil {
		return nil, err
	}

	ans := fmt.Sprintf("Context Built (%d / %d tokens)\n========\n%s", c.TokenCount, c.TokenBudget, c.Context)
	return mcp.NewToolResultText(ans), nil
}

func (s *MCPServer) Run() error {
	// mcp-go-sdk Server 底层自动借助 stdin/stdout 进行 JsonRPC 通讯交互
	return s.server.Run()
}
