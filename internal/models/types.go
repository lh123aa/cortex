package models

import (
	"strings"
	"time"
)

// Document 代表一个被索引的文档 (支持原始 Markdown, PDF 等)
type Document struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"` // 所有者用户 ID
	Path        string    `json:"path"`
	Title       string    `json:"title"`
	FileType    string    `json:"file_type"` // md/pdf/docx
	ContentHash string    `json:"content_hash"`
	FileSize    int64     `json:"file_size"`
	ChunkCount  int       `json:"chunk_count"`
	IndexedAt   time.Time `json:"indexed_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Status      string    `json:"status"` // indexed/processing/error
}

// Chunk 代表某个文档在被切分后的一块文本与向量
type Chunk struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"` // 所有者用户 ID
	DocumentID     string    `json:"document_id"`
	ContentHash    string    `json:"content_hash,omitempty"` // sha256 内容哈希，用于去重
	HeadingPath    string    `json:"heading_path"`           // 例如: "基础知识 > Go语言 > 并发"
	HeadingLevel   int       `json:"heading_level"`          // 0-6
	Content        string    `json:"content"`                // 带"Section: H1 > H2\n\n"前缀用于注入的上下文
	ContentRaw     string    `json:"content_raw"`            // 原始内容
	LineStart      int       `json:"line_start"`
	LineEnd        int       `json:"line_end"`
	CharStart      int       `json:"char_start"`
	CharEnd        int       `json:"char_end"`
	TokenCount     int       `json:"token_count"`
	Embedding      []float32 `json:"embedding,omitempty"`
	EmbeddingModel string    `json:"embedding_model,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// SearchOptions 控制搜索参数
type SearchOptions struct {
	TopK        int      `json:"top_k"`
	Offset      int      `json:"offset"`       // 分页偏移
	Mode        string   `json:"mode"`         // hybrid/vector/fts
	Filter      string   `json:"filter"`       // 路径过滤
	TokenBudget int      `json:"token_budget"` // RAG模式Token预算
	MinScore    float64  `json:"min_score"`    // 最小分数阈值
	Collections []string `json:"collections"`  // 限定集合
	UserID      string   `json:"-"`            // 用户隔离，不暴露给客户端
}

// SearchResult 表示一条召回的结构
type SearchResult struct {
	Chunk       *Chunk      `json:"chunk"`
	Score       float64     `json:"score"`               // 0-1标准化分数
	VectorScore float64     `json:"vector_score"`        // 向量相似度
	FTSScore    float64     `json:"fts_score"`           // BM25分数
	Rank        int         `json:"rank"`                // 排名
	FilePath    string      `json:"file_path,omitempty"` // 源文件路径（用于文件类型权重）
	Highlights  []Highlight `json:"highlights,omitempty"`
}

type Highlight struct {
	Text  string `json:"text"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

// DisplayText 返回适合展示给用户/模型的正文。
//
// 字段语义：
//   - Content    原文（可能带 "File: ... | section" 或 "Section: ..." 前缀，用于 RAG 上下文）
//   - ContentRaw 供 FTS5 检索的分词版本（中文会被拆成 "激 活" 这种单字词项）
//
// 展示时必须用原文。早期 CLI/REST/MCP 多处误用 ContentRaw，导致用户看到的是
// 「Go 标 准 库」这种被分词过的文本，既难读也污染了模型上下文。
func (c *Chunk) DisplayText() string {
	if c == nil {
		return ""
	}
	if v := strings.TrimSpace(c.Content); v != "" {
		return stripChunkBreadcrumb(v)
	}
	return strings.TrimSpace(c.ContentRaw)
}

// stripChunkBreadcrumb 去掉纯展示用的定位前缀行（"File: ... | section" / "Section: ..."）。
// 正文本身保留，避免把路径噪声塞进搜索结果里。
func stripChunkBreadcrumb(s string) string {
	for {
		idx := strings.Index(s, "\n\n")
		if idx < 0 || idx > 300 {
			return s
		}
		head := strings.TrimSpace(s[:idx])
		if head == "" {
			return s
		}
		lower := strings.ToLower(head)
		if strings.HasPrefix(lower, "file:") ||
			strings.HasPrefix(lower, "section:") ||
			strings.HasPrefix(lower, "path:") {
			s = s[idx+2:]
			continue
		}
		return s
	}
}

// FilterOptions 搜索过滤选项
type FilterOptions struct {
	UserID string
	Path   string
	Status string
}

// IndexRequest 索引请求
type IndexRequest struct {
	Path     string `json:"path"`
	UserID   string `json:"-"`
	SyncMode string `json:"sync_mode"` // full/incremental
}

// BatchIndexRequest 批量索引请求
type BatchIndexRequest struct {
	Paths  []string `json:"paths"`
	UserID string   `json:"-"`
}

// IndexResponse 索引响应
type IndexResponse struct {
	Total    int      `json:"total"`
	Indexed  int      `json:"indexed"`
	Skipped  int      `json:"skipped"`
	Failed   int      `json:"failed"`
	Duration int64    `json:"duration_ms"`
	Errors   []string `json:"errors,omitempty"`
}

// Memory 记忆条目（用于非文件内容的直接写入）
type Memory struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Content   string    `json:"content"`   // 记忆内容
	Summary   string    `json:"summary"`   // 自动生成的摘要
	Tags      []string  `json:"tags"`      // 标签
	Source    string    `json:"source"`    // 来源 (e.g., "conversation", "manual")
	SourceID  string    `json:"source_id"` // 来源ID (e.g., conversation_id)
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MemoryRequest 写入记忆请求
type MemoryRequest struct {
	Content string   `json:"content" binding:"required,min=1"`
	Summary string   `json:"summary"`
	Tags    []string `json:"tags"`
	Source  string   `json:"source"`
}

// MemoryResponse 记忆响应
type MemoryResponse struct {
	ID        string   `json:"id"`
	Content   string   `json:"content"`
	Summary   string   `json:"summary"`
	Tags      []string `json:"tags"`
	Source    string   `json:"source"`
	CreatedAt string   `json:"created_at"`
}

// MemorySearchResult 记忆搜索结果
type MemorySearchResult struct {
	ID      string  `json:"id"`
	Content string  `json:"content"`
	Summary string  `json:"summary"`
	Score   float64 `json:"score"`
	Source  string  `json:"source"`
}

// IndexProgressEvent 实时索引进度事件（通过 OnProgress 回调发送）
type IndexProgressEvent struct {
	Total       int           `json:"total"`
	Indexed     int           `json:"indexed"`
	Skipped     int           `json:"skipped"`
	Failed      int           `json:"failed"`
	CurrentFile string        `json:"current_file"`
	Speed       float64       `json:"speed"` // files/sec
	Elapsed     time.Duration `json:"elapsed_ms"`
	ETA         time.Duration `json:"eta_ms"` // 预计剩余时间
	Done        bool          `json:"done"`   // 是否已完成
}
