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
	ServerName        = "cortex"
)

// Tool input schemas
type SearchArgs struct {
	Query string `json:"query" jsonschema:"description=The exact search query to lookup;required"`
	TopK  int    `json:"top_k,omitempty" jsonschema:"description=Number of results to return"`
}

type ContextArgs struct {
	Query       string `json:"query" jsonschema:"description=The query to build context upon;required"`
	TokenBudget int    `json:"token_budget,omitempty" jsonschema:"description=Allowed max tokens"`
}

type MCPServer struct {
	server  *mcp.Server
	search  *search.HybridSearchEngine
	rag     *rag.RAGBuilder
	storage storage.Storage
	logger  *zap.Logger
	userID  string  // 用户隔离：当前 MCP 会话的 userID
}

// SetUserID 设置 MCP 服务器的用户上下文
// 注意：生产环境应通过 MCP 认证机制获取用户身份
func (s *MCPServer) SetUserID(userID string) {
	s.userID = userID
}

func NewMCPServer(se *search.HybridSearchEngine, st storage.Storage, log *zap.Logger) *MCPServer {
	s := &MCPServer{
		search:  se,
		rag:     rag.NewRAGBuilder(se),
		storage: st,
		logger:  log,
	}

	// 实例化 MCP Server - v1.2.0 API
	s.server = mcp.NewServer(&mcp.Implementation{
		Name:    ServerName,
		Version: "v1.0.0",
	}, &mcp.ServerOptions{
		// v1.2.0: 没有 ProtocolVersion 字段，协议版本自动协商
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
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "cortex_search",
		Description: "Search the local knowledge base (cortex) using vector and fts and return relevant chunks",
	}, s.handleSearchTool)

	// cortex_context: 组装 RAG
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "cortex_context",
		Description: "Assemble relevant information within a specific token budget limit strictly",
	}, s.handleContextTool)
}

func (s *MCPServer) handleSearchTool(ctx context.Context, req *mcp.CallToolRequest, args SearchArgs) (*mcp.CallToolResult, any, error) {
	topK := args.TopK
	if topK <= 0 {
		topK = 10
	}

	opts := models.SearchOptions{TopK: topK, Mode: "hybrid"}
	results, err := s.search.Search(ctx, args.Query, opts)
	if err != nil {
		s.logger.Error("mcp tool execution failed on search", zap.Error(err))
		return nil, nil, fmt.Errorf("search operational error: %v", err)
	}

	var sb strings.Builder
	for i, r := range results {
		// P1-3: 使用用户隔离查询，而非空字符串
		docPath := r.Chunk.DocumentID // fallback
		if doc, err := s.storage.GetDocumentByID(r.Chunk.DocumentID, s.userID); err == nil && doc != nil {
			docPath = doc.Path
		}
		sb.WriteString(fmt.Sprintf("[%d] Score: %.3f\nPath: %s\nSection: %s\n\n%s\n---\n", i+1, r.Score, docPath, r.Chunk.HeadingPath, truncateText(r.Chunk.ContentRaw, 300)))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: sb.String()}},
	}, nil, nil
}

func (s *MCPServer) handleContextTool(ctx context.Context, req *mcp.CallToolRequest, args ContextArgs) (*mcp.CallToolResult, any, error) {
	budget := args.TokenBudget
	if budget <= 0 {
		budget = 1500
	}

	opts := models.SearchOptions{TopK: 50, Mode: "hybrid"}
	c, err := s.rag.BuildContext(ctx, args.Query, budget, opts)
	if err != nil {
		return nil, nil, err
	}

	ans := fmt.Sprintf("Context Built (%d / %d tokens)\n========\n%s", c.TokenCount, c.TokenBudget, c.Context)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: ans}},
	}, nil, nil
}

func (s *MCPServer) Run() error {
	// mcp-go-sdk Server 底层自动借助 stdin/stdout 进行 JsonRPC 通讯交互
	return s.server.Run(context.Background(), &mcp.StdioTransport{})
}
