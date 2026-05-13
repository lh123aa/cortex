package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/lh123aa/cortex/internal/chunker"
	"github.com/lh123aa/cortex/internal/embedding"
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

// 可通过 -ldflags 在构建时注入版本号
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// toolError 创建标准 MCP 错误响应
func toolError(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}

// Tool input schemas
// Note: jsonschema tag value is the description text (jsonschema-go v0.3.0 format).
// Required fields are inferred from JSON tags: fields without "omitempty" are required.
type SearchArgs struct {
	Query string `json:"query" jsonschema:"The exact search query to lookup"`
	TopK  int    `json:"top_k,omitempty" jsonschema:"Number of results to return"`
}

type ContextArgs struct {
	Query       string `json:"query" jsonschema:"The query to build context upon"`
	TokenBudget int    `json:"token_budget,omitempty" jsonschema:"Allowed max tokens"`
}

type MemoryWriteArgs struct {
	Content string   `json:"content" jsonschema:"The memory content to store"`
	Tags    []string `json:"tags,omitempty" jsonschema:"Optional tags for the memory"`
	Source  string   `json:"source,omitempty" jsonschema:"Optional source identifier"`
	Summary string   `json:"summary,omitempty" jsonschema:"Optional summary, auto-generated if empty"`
}

type MemorySearchArgs struct {
	Query string `json:"query" jsonschema:"The query to search memories"`
	TopK  int    `json:"top_k,omitempty" jsonschema:"Number of results to return"`
}

type MemoryDeleteArgs struct {
	ID string `json:"id" jsonschema:"The memory ID to delete"`
}

type MemoryDeleteBatchArgs struct {
	IDs []string `json:"ids" jsonschema:"List of memory IDs to delete"`
}

type SuggestArgs struct {
	Query       string `json:"query" jsonschema:"The partial query to get suggestions for"`
	CurrentFile string `json:"current_file,omitempty" jsonschema:"Current file path for context-aware suggestions"`
	TopK        int    `json:"top_k,omitempty" jsonschema:"Number of suggestions to return"`
}

type MCPServer struct {
	server  *mcp.Server
	search  *search.HybridSearchEngine
	rag     *rag.RAGBuilder
	storage storage.Storage
	memory  *MemoryHandler
	logger  *zap.Logger
	userID  string // 用户隔离：当前 MCP 会话的 userID

	startupTime time.Time // MCP 服务器启动时间，用于健康检测
}

// SetUserID 设置 MCP 服务器的用户上下文
// 注意：生产环境应通过 MCP 认证机制获取用户身份
func (s *MCPServer) SetUserID(userID string) {
	s.userID = userID
}

func NewMCPServer(se *search.HybridSearchEngine, st storage.Storage, em embedding.EmbeddingProvider, log *zap.Logger) *MCPServer {
	mh := NewMemoryHandler(st, se, em, log)
	s := &MCPServer{
		search:      se,
		rag:         rag.NewRAGBuilder(se),
		storage:     st,
		memory:      mh,
		logger:      log,
		startupTime: time.Now(),
	}

	// 实例化 MCP Server
	s.server = mcp.NewServer(&mcp.Implementation{
		Name:    ServerName,
		Version: Version,
	}, &mcp.ServerOptions{
		// 协议版本自动协商
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

	// cortex_memory_write: 写入记忆
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "cortex_memory_write",
		Description: "Write a memory entry to the cortex knowledge base for persistent storage",
	}, s.handleMemoryWriteTool)

	// cortex_memory_search: 搜索记忆
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "cortex_memory_search",
		Description: "Search memory entries in the cortex knowledge base",
	}, s.handleMemorySearchTool)

	// cortex_memory_delete: 删除记忆
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "cortex_memory_delete",
		Description: "Delete a memory entry from the cortex knowledge base by its ID",
	}, s.handleMemoryDeleteTool)

	// cortex_memory_delete_batch: 批量删除记忆
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "cortex_memory_delete_batch",
		Description: "Delete multiple memory entries from the cortex knowledge base by their IDs",
	}, s.handleMemoryDeleteBatchTool)

	// cortex_health: 健康检测
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "cortex_health",
		Description: "Check if the cortex server is healthy and responsive. Returns system status, document count, and uptime.",
	}, s.handleHealthTool)

	// cortex_suggest: 预联想搜索建议
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "cortex_suggest",
		Description: "Get smart search suggestions based on partial query and current context. Returns instant results from prefetched cache or fast search.",
	}, s.handleSuggestTool)
}

func (s *MCPServer) handleSearchTool(ctx context.Context, req *mcp.CallToolRequest, args SearchArgs) (*mcp.CallToolResult, any, error) {
	// 参数校验
	if args.Query == "" {
		return toolError("query is required"), nil, nil
	}
	if len(args.Query) > 5000 {
		return toolError("query too long (max 5000 characters)"), nil, nil
	}

	topK := args.TopK
	if topK <= 0 {
		topK = 10
	}
	if topK > 200 {
		topK = 200
	}

	opts := models.SearchOptions{TopK: topK, Mode: "hybrid"}
	results, err := s.search.Search(ctx, args.Query, opts)
	if err != nil {
		s.logger.Error("mcp tool execution failed on search", zap.Error(err))
		return toolError(fmt.Sprintf("search error: %v", err)), nil, nil
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
	// 参数校验
	if args.Query == "" {
		return toolError("query is required"), nil, nil
	}
	if len(args.Query) > 5000 {
		return toolError("query too long (max 5000 characters)"), nil, nil
	}

	budget := args.TokenBudget
	if budget <= 0 {
		budget = 1500
	}
	if budget > 100000 {
		budget = 100000
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

func (s *MCPServer) handleMemoryWriteTool(ctx context.Context, req *mcp.CallToolRequest, args MemoryWriteArgs) (*mcp.CallToolResult, any, error) {
	if args.Content == "" {
		return toolError("content is required"), nil, nil
	}

	userID := s.userID

	contentHash := sha256.Sum256([]byte(args.Content))
	memoryID := hex.EncodeToString(contentHash[:16])

	summary := args.Summary
	if summary == "" && len(args.Content) > 100 {
		summary = args.Content[:100] + "..."
	} else if summary == "" {
		summary = args.Content
	}

	tk, err := chunker.NewTextChunker(chunker.ChunkConfig{
		MinChars:  50,
		MaxTokens: 512,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create chunker: %w", err)
	}

	chunks, err := tk.Chunk(args.Content, fmt.Sprintf("memory://%s", memoryID))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to chunk memory: %w", err)
	}

	if len(chunks) > 0 {
		texts := make([]string, len(chunks))
		for i, chunk := range chunks {
			texts[i] = chunk.ContentRaw
		}
		if s.memory.em != nil {
			if embeddings, err := s.memory.em.EmbedBatch(texts); err == nil {
				for i, chunk := range chunks {
					chunk.Embedding = embeddings[i]
					if s.memory.em != nil {
						chunk.EmbeddingModel = s.memory.em.Name()
					}
				}
			}
		}
	}

	doc := &models.Document{
		ID:          memoryID,
		UserID:      userID,
		Path:        fmt.Sprintf("memory://%s", memoryID),
		FileType:    "memory",
		ContentHash: hex.EncodeToString(contentHash[:]),
		ChunkCount:  len(chunks),
		Status:      "indexed",
	}
	if err := s.storage.SaveDocument(doc); err != nil {
		return nil, nil, fmt.Errorf("failed to save memory: %w", err)
	}

	for _, chunk := range chunks {
		chunk.UserID = userID
		chunk.DocumentID = memoryID
	}
	if err := s.storage.SaveChunks(chunks); err != nil {
		return nil, nil, fmt.Errorf("failed to save memory chunks: %w", err)
	}

	result := fmt.Sprintf("Memory written successfully:\nID: %s\nSummary: %s\nTags: %v\nSource: %s",
		memoryID, summary, args.Tags, args.Source)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, nil, nil
}

func (s *MCPServer) handleMemorySearchTool(ctx context.Context, req *mcp.CallToolRequest, args MemorySearchArgs) (*mcp.CallToolResult, any, error) {
	if args.Query == "" {
		return toolError("query is required"), nil, nil
	}

	topK := args.TopK
	if topK <= 0 {
		topK = 10
	}

	opts := models.SearchOptions{
		TopK:   topK,
		Mode:   "hybrid",
		UserID: s.userID,
	}

	results, err := s.search.Search(ctx, args.Query, opts)
	if err != nil {
		return nil, nil, fmt.Errorf("memory search failed: %w", err)
	}

	var sb strings.Builder
	count := 0
	for _, r := range results {
		doc, _ := s.storage.GetDocumentByID(r.Chunk.DocumentID, s.userID)
		if doc == nil || doc.FileType != "memory" {
			continue
		}
		count++
		sb.WriteString(fmt.Sprintf("[%d] Score: %.3f\nID: %s\nContent: %s\n---\n",
			count, r.Score, doc.ID, truncateText(r.Chunk.ContentRaw, 300)))
	}

	if count == 0 {
		sb.WriteString("No memory entries found.")
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: sb.String()}},
	}, nil, nil
}

func (s *MCPServer) handleMemoryDeleteTool(ctx context.Context, req *mcp.CallToolRequest, args MemoryDeleteArgs) (*mcp.CallToolResult, any, error) {
	if args.ID == "" {
		return toolError("id is required"), nil, nil
	}

	if err := s.storage.DeleteDocumentByPath(fmt.Sprintf("memory://%s", args.ID), s.userID); err != nil {
		return nil, nil, fmt.Errorf("failed to delete memory: %w", err)
	}

	if err := s.storage.InvalidateSearchCache(); err != nil {
		s.logger.Warn("failed to invalidate search cache after memory deletion", zap.Error(err))
	}

	result := fmt.Sprintf("Memory deleted successfully:\nID: %s", args.ID)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, nil, nil
}

func (s *MCPServer) handleMemoryDeleteBatchTool(ctx context.Context, req *mcp.CallToolRequest, args MemoryDeleteBatchArgs) (*mcp.CallToolResult, any, error) {
	if len(args.IDs) == 0 {
		return toolError("ids is required"), nil, nil
	}

	var deleted []string
	var failed []string
	for _, id := range args.IDs {
		if err := s.storage.DeleteDocumentByPath(fmt.Sprintf("memory://%s", id), s.userID); err != nil {
			failed = append(failed, id)
			s.logger.Warn("failed to delete memory in batch", zap.String("id", id), zap.Error(err))
		} else {
			deleted = append(deleted, id)
		}
	}

	if err := s.storage.InvalidateSearchCache(); err != nil {
		s.logger.Warn("failed to invalidate search cache after batch deletion", zap.Error(err))
	}

	result := fmt.Sprintf("Batch delete complete:\n  Deleted: %v\n  Failed: %v", deleted, failed)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, nil, nil
}

func (s *MCPServer) handleHealthTool(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, any, error) {
	uptime := time.Since(s.startupTime).Round(time.Second).String()

	docCount, _ := s.storage.GetDocumentsCount("")

	result := fmt.Sprintf(`✅ Cortex MCP Server is healthy

Server:     %s v%s
Uptime:     %s
Status:     Running
Documents:  %d
Embedding:  %s (FTS5)
`,
		ServerName, Version, uptime,
		docCount,
		"none",
	)

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, nil, nil
}

func (s *MCPServer) handleSuggestTool(ctx context.Context, req *mcp.CallToolRequest, args SuggestArgs) (*mcp.CallToolResult, any, error) {
	if args.Query == "" {
		return toolError("query is required"), nil, nil
	}

	topK := args.TopK
	if topK <= 0 {
		topK = 5
	}
	if topK > 20 {
		topK = 20
	}

	pe := s.search.GetPrefetchEngine()
	if pe != nil {
		results := pe.Suggest(ctx, args.Query, args.CurrentFile, topK)
		if len(results) > 0 {
			var sb strings.Builder
			for i, r := range results {
				sb.WriteString(fmt.Sprintf("[%d] Score: %.3f\nSection: %s\n%s\n---\n",
					i+1, r.Score, r.Chunk.HeadingPath, truncateText(r.Chunk.ContentRaw, 200)))
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: sb.String()}},
			}, nil, nil
		}
	}

	opts := models.SearchOptions{TopK: topK, Mode: "hybrid"}
	results, err := s.search.Search(ctx, args.Query, opts)
	if err != nil {
		return toolError(fmt.Sprintf("search error: %v", err)), nil, nil
	}

	var sb strings.Builder
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("[%d] Score: %.3f\nSection: %s\n%s\n---\n",
			i+1, r.Score, r.Chunk.HeadingPath, truncateText(r.Chunk.ContentRaw, 200)))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: sb.String()}},
	}, nil, nil
}

func (s *MCPServer) Run() error {
	// mcp-go-sdk Server 底层自动借助 stdin/stdout 进行 JsonRPC 通讯交互
	return s.server.Run(context.Background(), &mcp.StdioTransport{})
}
