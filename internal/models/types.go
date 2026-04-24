package models

import "time"

// Document 代表一个被索引的文档 (支持原始 Markdown, PDF 等)
type Document struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`       // 所有者用户 ID
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
	UserID         string    `json:"user_id"`        // 所有者用户 ID
	DocumentID     string    `json:"document_id"`
	HeadingPath    string    `json:"heading_path"`  // 例如: "基础知识 > Go语言 > 并发"
	HeadingLevel   int       `json:"heading_level"` // 0-6
	Content        string    `json:"content"`       // 带"Section: H1 > H2\n\n"前缀用于注入的上下文
	ContentRaw     string    `json:"content_raw"`   // 原始内容
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
	Mode        string   `json:"mode"`         // hybrid/vector/fts
	Filter      string   `json:"filter"`       // 路径过滤
	TokenBudget int      `json:"token_budget"` // RAG模式Token预算
	MinScore    float64  `json:"min_score"`    // 最小分数阈值
	Collections []string `json:"collections"`   // 限定集合
	UserID      string   `json:"-"`             // 用户隔离，不暴露给客户端
}

// SearchResult 表示一条召回的结构
type SearchResult struct {
	Chunk       *Chunk      `json:"chunk"`
	Score       float64     `json:"score"`        // 0-1标准化分数
	VectorScore float64     `json:"vector_score"` // 向量相似度
	FTSScore    float64     `json:"fts_score"`    // BM25分数
	Rank        int         `json:"rank"`         // 排名
	Highlights  []Highlight `json:"highlights,omitempty"`
}

type Highlight struct {
	Text  string `json:"text"`
	Start int    `json:"start"`
	End   int    `json:"end"`
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