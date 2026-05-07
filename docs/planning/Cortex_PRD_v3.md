# Cortex - Agent Knowledge Base

> **版本**：v3.1（完整版 - 已补充细节）  
> **技术栈**：Go + SQLite + 可插拔Embedding  
> **参考**：sqmd架构设计 + qmd (Go版)  
> **定位**：5分钟上手的Agent知识库  
> **MCP协议版本**：2025-06-18 (Stable)

---

## 一、产品概述

### 1.1 核心定位

**Cortex** 是一个专为AI Agent设计的知识管理系统——Agent的大脑皮层。

**核心理念**：
- **Agent原生**：API设计优先，人类界面其次
- **5分钟上手**：下载二进制、一行命令索引、一行命令搜索
- **零配置启动**：SQLite嵌入式存储，无需安装数据库
- **隐私保护**：数据本地存储，可选云端Embedding

### 1.2 目标用户

| 用户类型 | 使用方式 | 核心诉求 |
|----------|----------|----------|
| AI Agent (Coze/Dify/LangChain) | MCP协议 / REST API | 高效知识检索、结构化存储 |
| 开发者 | API集成 / CLI | 轻量级RAG方案，无运维负担 |
| 个人用户 | CLI / Web UI | 本地知识管理，隐私可控 |

### 1.3 核心价值主张

```
传统方案的问题：
┌─────────────────────────────────────────────────┐
│ LlamaIndex/LangChain: 重型框架，配置复杂         │
│ Obsidian + AI插件: 人类优先，Agent集成弱         │
│ 云端RAG服务: 数据上传，隐私风险，成本高          │
└─────────────────────────────────────────────────┘

Cortex的解法：
┌─────────────────────────────────────────────────┐
│ 1. 下载单二进制 → 无需Python/Node环境            │
│ 2. cortex index ~/notes → 一行命令索引          │
│ 3. cortex search "xxx" → 混合搜索出结果         │
│ 4. SQLite存储 → 无需安装数据库                  │
│ 5. MCP原生 → Agent直接调用                      │
└─────────────────────────────────────────────────┘
```

### 1.4 差异化优势

| 维度 | LlamaIndex | LangChain | Cortex |
|------|------------|-----------|--------|
| 部署复杂度 | 高（Python环境+依赖） | 高（Python环境+依赖） | **低（单二进制）** |
| 配置成本 | 高（需配置向量库） | 高（需配置向量库） | **零（SQLite嵌入式）** |
| Agent支持 | 中（需封装） | 中（需封装） | **高（MCP原生）** |
| 搜索精度 | 语义搜索 | 语义搜索 | **混合搜索（向量+BM25）** |
| 上手时间 | 1-2小时 | 1-2小时 | **5分钟** |

**量化性能指标（目标）**：

| 指标 | Cortex目标 | LlamaIndex典型值 | 对比说明 |
|------|------------|------------------|----------|
| 搜索召回率 | **≥90%** | 85-95% | 混合搜索提升召回 |
| 搜索响应时间 | **≤100ms** | 200-500ms | SQLite本地无网络延迟 |
| 索引速度 | **≥100 docs/min** | 50-100 docs/min | 增量索引+并行处理 |
| 部署时间 | **≤5分钟** | 30-60分钟 | 单二进制vs Python环境 |
| 内存占用 | **≤200MB** | 500MB-1GB | SQLite vs 向量数据库 |
| 单节点最大分块数 | **≥100万** | 无限制 | SQLite WAL模式支撑 |

---

## 二、核心功能

### 2.1 功能架构

```
┌─────────────────────────────────────────────────────────────┐
│                        用户界面层                            │
│   CLI (cobra)    │    REST API (gin)    │    MCP Server     │
└────────┬─────────┴──────────┬───────────┴─────────┬─────────┘
         │                    │                     │
         ▼                    ▼                     ▼
┌─────────────────────────────────────────────────────────────┐
│                        业务逻辑层                            │
│  索引管理  │  搜索引擎  │  RAG组装  │  配置管理  │  监控     │
└────────┬───┴─────┬──────┴─────┬────┴─────┬──────┴─────┬─────┘
         │         │            │          │            │
         ▼         ▼            ▼          ▼            ▼
┌─────────────────────────────────────────────────────────────┐
│                        核心算法层                            │
│  层级分块  │  向量搜索  │  BM25搜索  │  RRF融合  │ Token计算 │
└────────┬───┴─────┬──────┴─────┬───────┴────┬─────┴──────────┘
         │         │            │            │
         ▼         ▼            ▼            ▼
┌─────────────────────────────────────────────────────────────┐
│                        基础设施层                            │
│  存储抽象  │  Embedding抽象  │  日志  │  监控  │  缓存     │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 MVP功能清单

| 功能模块 | 子功能 | 优先级 | 状态 |
|----------|--------|--------|------|
| **文档索引** | Markdown解析 | P0 | 待开发 |
| | PDF解析 | P1 | 待开发 |
| | Word解析 | P2 | 待开发 |
| | 增量索引（Hash检测） | P0 | 待开发 |
| | 文件监听自动索引 | P1 | 待开发 |
| **智能分块** | Markdown层级分块 | P0 | 待开发 |
| | Token预算控制 | P0 | 待开发 |
| | 分块预览（Web UI） | P1 | 待开发 |
| **混合搜索** | 向量语义搜索 | P0 | 待开发 |
| | BM25关键词搜索 | P0 | 待开发 |
| | RRF融合排序 | P0 | 待开发 |
| | 搜索结果缓存 | P0 | 待开发 |
| **RAG输出** | Token预算上下文组装 | P0 | 待开发 |
| | 源归属标注 | P0 | 待开发 |
| | 上下文截断策略 | P1 | 待开发 |
| **Embedding** | Ollama本地嵌入 | P0 | 待开发 |
| | ONNX本地嵌入 | P1 | 待开发 |
| | OpenAI云端嵌入（fallback） | P0 | 待开发 |
| | 批量嵌入优化 | P0 | 待开发 |
| **接口层** | CLI命令 | P0 | 待开发 |
| | REST API | P0 | 待开发 |
| | MCP Server | P0 | 待开发 |
| | LangChain适配器 | P1 | 待开发 |
| | LlamaIndex适配器 | P1 | 待开发 |
| **可视化** | 索引状态查看 | P1 | 待开发 |
| | 分块预览 | P1 | 待开发 |
| | 搜索结果对比 | P2 | 待开发 |
| **工程化** | 日志系统（zap） | P0 | 待开发 |
| | 监控指标（prometheus） | P0 | 待开发 |
| | 单元测试 | P0 | 待开发 |
| | 集成测试 | P0 | 待开发 |

---

## 三、技术架构

### 3.1 技术选型

| 组件 | 技术选型 | 版本 | 选型理由 |
|------|----------|------|----------|
| 语言 | Go | 1.21+ | 单二进制部署，跨平台编译，高性能 |
| 存储 | SQLite | 3.40+ | 零配置，嵌入式，单文件，WAL模式支持并发 |
| 向量存储 | **自建向量索引** | - | 无需sqlite-vss扩展，保持零配置 |
| Embedding | Ollama / ONNX / OpenAI | - | 可插拔多方案，降低用户门槛 |
| Token计算 | tiktoken-go | - | 专业Token计算，精度高 |
| 日志 | zap | 1.27+ | 高性能结构化日志 |
| 监控 | prometheus | - | 业界标准监控方案 |
| 缓存 | go-cache | - | 内存缓存，简单高效 |
| Web框架 | gin | 1.9+ | 高性能，中间件丰富 |
| CLI框架 | cobra | 1.8+ | 标准CLI框架 |
| 配置管理 | viper | 1.18+ | 支持多格式、环境变量 |
| Markdown解析 | goldmark | 1.7+ | 高性能，可扩展 |
| PDF解析 | **pdfcpu** | 0.7+ | **开源免费**，MIT协议 |
| Word解析 | **unidoc-unioffice** | 1.x | 开源免费，或用goversion-docx |
| 文件监听 | fsnotify | 1.7+ | 跨平台文件系统监听 |
| 并发控制 | ants | 2.x | goroutine池化 |

**关键选型说明**：

1. **自建向量索引 vs sqlite-vss**：
   - sqlite-vss需要编译SQLite扩展，与"零配置"矛盾
   - 自建方案：向量存为BLOB，在内存中计算余弦相似度
   - 性能：10万向量内搜索<50ms，满足MVP需求
   - 后续可扩展为HNSW索引（性能优化）

2. **PDF解析选型**：
   - pdfcpu（MIT协议）：开源免费，支持文本提取
   - unipdf需商用授权，不适合开源项目
   - pdfcpu不支持OCR，表格提取精度一般，但满足基础需求

3. **Word解析选型**：
   - unioffice：开源免费（社区版），功能完整
   - goversion-docx：纯Go实现，更轻量
   - 推荐unioffice，功能更全

### 3.2 架构分层设计

```go
// 存储层抽象 - 解耦具体实现
type Storage interface {
    // 文档操作
    SaveDocument(doc *Document) error
    GetDocument(id string) (*Document, error)
    DeleteDocument(id string) error
    ListDocuments(filter DocumentFilter) ([]*Document, error)
    
    // 分块操作
    SaveChunks(chunks []*Chunk) error
    GetChunk(id string) (*Chunk, error)
    DeleteChunksByDocument(docID string) error
    
    // 搜索操作
    VectorSearch(vector []float32, topK int) ([]*SearchResult, error)
    FTSSearch(query string, topK int) ([]*SearchResult, error)
    
    // 元数据操作
    GetMetadata(key string) (string, error)
    SetMetadata(key, value string) error
}

// Embedding接口 - 可插拔
type EmbeddingProvider interface {
    // 批量嵌入（性能优化）
    EmbedBatch(texts []string) ([][]float32, error)
    // 单个嵌入
    Embed(text string) ([]float32, error)
    // 向量维度
    Dimension() int
    // 提供商名称
    Name() string
    // 健康检查
    Health() error
}

// 分块器接口 - 可插拔
type Chunker interface {
    Chunk(content string, path string) ([]*Chunk, error)
    Name() string
}

// 搜索引擎接口
type SearchEngine interface {
    Search(query string, opts SearchOptions) ([]*SearchResult, error)
    HybridSearch(query string, opts SearchOptions) ([]*SearchResult, error)
}
```

### 3.3 并发控制设计

```go
// 全局goroutine池
var workerPool *ants.Pool

func init() {
    var err error
    workerPool, err = ants.NewPool(100, ants.WithPreAlloc(true))
    if err != nil {
        panic(err)
    }
}

// 搜索带超时控制
func (s *SearchEngine) SearchWithTimeout(
    ctx context.Context, 
    query string, 
    timeout time.Duration,
) ([]*SearchResult, error) {
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()
    
    resultCh := make(chan searchResult, 1)
    
    err := workerPool.Submit(func() {
        results, err := s.Search(ctx, query)
        resultCh <- searchResult{results: results, err: err}
    })
    if err != nil {
        return nil, err
    }
    
    select {
    case res := <-resultCh:
        return res.results, res.err
    case <-ctx.Done():
        return nil, fmt.Errorf("search timeout after %v", timeout)
    }
}
```

---

## 四、数据结构设计

### 4.1 SQLite Schema（优化版）

```sql
-- 启用WAL模式（解决并发问题）
PRAGMA journal_mode=WAL;
PRAGMA synchronous=NORMAL;
PRAGMA cache_size=10000;
PRAGMA busy_timeout=5000;

-- 文档表
CREATE TABLE IF NOT EXISTS documents (
    id TEXT PRIMARY KEY,              -- 文档ID (SHA-256前16位)
    path TEXT UNIQUE NOT NULL,        -- 文件路径
    title TEXT,                       -- 文档标题
    file_type TEXT DEFAULT 'md',      -- 文件类型: md/pdf/docx
    content_hash TEXT NOT NULL,       -- 内容SHA-256
    file_size INTEGER,                -- 文件大小(字节)
    chunk_count INTEGER DEFAULT 0,    -- 分块数量
    indexed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status TEXT DEFAULT 'indexed'     -- indexed/processing/error
);

CREATE INDEX idx_documents_path ON documents(path);
CREATE INDEX idx_documents_status ON documents(status);
CREATE INDEX idx_documents_indexed_at ON documents(indexed_at);

-- 分块表
CREATE TABLE IF NOT EXISTS chunks (
    id TEXT PRIMARY KEY,              -- 分块ID
    document_id TEXT NOT NULL,        -- 所属文档
    heading_path TEXT,                -- 标题路径 "H1 > H2 > H3"
    heading_level INTEGER,            -- 标题层级 0-6
    content TEXT NOT NULL,            -- 分块内容（含标题前缀）
    content_raw TEXT NOT NULL,        -- 原始内容（不含标题前缀）
    line_start INTEGER,               -- 起始行号
    line_end INTEGER,                 -- 结束行号
    char_start INTEGER,               -- 起始字符位置
    char_end INTEGER,                 -- 结束字符位置
    token_count INTEGER,              -- Token数量（tiktoken计算）
    embedding BLOB,                   -- 向量嵌入（JSON数组）
    embedding_model TEXT,             -- 使用的嵌入模型
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);

CREATE INDEX idx_chunks_document_id ON chunks(document_id);
CREATE INDEX idx_chunks_heading_level ON chunks(heading_level);

-- 全文搜索索引
CREATE VIRTUAL TABLE IF NOT EXISTS chunks_fts USING fts5(
    content_raw,
    heading_path,
    document_id,
    content='chunks',
    content_rowid='rowid',
    tokenize='unicode61'
);

-- 向量存储表（如果不用sqlite-vss）
CREATE TABLE IF NOT EXISTS vectors (
    chunk_id TEXT PRIMARY KEY,
    embedding BLOB NOT NULL,          -- 序列化的向量
    dimension INTEGER NOT NULL,
    model TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (chunk_id) REFERENCES chunks(id) ON DELETE CASCADE
);

-- 元数据表（键值存储）
CREATE TABLE IF NOT EXISTS metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 索引任务队列表（支持增量索引）
CREATE TABLE IF NOT EXISTS index_tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT NOT NULL,
    action TEXT NOT NULL,             -- add/update/delete
    status TEXT DEFAULT 'pending',    -- pending/processing/completed/failed
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP
);

CREATE INDEX idx_index_tasks_status ON index_tasks(status);

-- 搜索缓存表（可选，持久化缓存）
CREATE TABLE IF NOT EXISTS search_cache (
    query_hash TEXT PRIMARY KEY,      -- query的SHA-256
    query TEXT NOT NULL,
    mode TEXT NOT NULL,
    results TEXT NOT NULL,            -- JSON序列化的结果
    hit_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_search_cache_expires ON search_cache(expires_at);
```

### 4.2 Go结构体定义（完整版）

```go
package models

import "time"

// 文档
type Document struct {
    ID           string    `json:"id"`
    Path         string    `json:"path"`
    Title        string    `json:"title"`
    FileType     string    `json:"file_type"`     // md/pdf/docx
    ContentHash  string    `json:"content_hash"`
    FileSize     int64     `json:"file_size"`
    ChunkCount   int       `json:"chunk_count"`
    IndexedAt    time.Time `json:"indexed_at"`
    UpdatedAt    time.Time `json:"updated_at"`
    Status       string    `json:"status"`        // indexed/processing/error
}

// 分块
type Chunk struct {
    ID            string    `json:"id"`
    DocumentID    string    `json:"document_id"`
    HeadingPath   string    `json:"heading_path"`   // "基础知识 > Go语言 > 并发"
    HeadingLevel  int       `json:"heading_level"`  // 0-6
    Content       string    `json:"content"`        // 带"Section: H1 > H2\n\n"前缀
    ContentRaw    string    `json:"content_raw"`    // 原始内容
    LineStart     int       `json:"line_start"`
    LineEnd       int       `json:"line_end"`
    CharStart     int       `json:"char_start"`
    CharEnd       int       `json:"char_end"`
    TokenCount    int       `json:"token_count"`
    Embedding     []float32 `json:"embedding,omitempty"`
    EmbeddingModel string   `json:"embedding_model,omitempty"`
    CreatedAt     time.Time `json:"created_at"`
    UpdatedAt     time.Time `json:"updated_at"`
}

// 搜索结果
type SearchResult struct {
    Chunk       *Chunk  `json:"chunk"`
    Score       float64 `json:"score"`        // 0-1标准化分数
    VectorScore float64 `json:"vector_score"` // 向量相似度
    FTSScore    float64 `json:"fts_score"`    // BM25分数
    Rank        int     `json:"rank"`         // 排名
    Highlights  []Highlight `json:"highlights,omitempty"` // 高亮片段
}

// 高亮片段
type Highlight struct {
    Text   string `json:"text"`
    Start  int    `json:"start"`
    End    int    `json:"end"`
}

// RAG上下文输出
type RAGContext struct {
    Context     string   `json:"context"`      // 组装好的上下文文本
    TokenCount  int      `json:"token_count"`  // 实际Token数
    TokenBudget int      `json:"token_budget"` // Token预算
    Sources     []Source `json:"sources"`      // 来源列表
    Truncated   bool     `json:"truncated"`    // 是否被截断
}

// 来源
type Source struct {
    Path        string  `json:"path"`
    HeadingPath string  `json:"heading_path"`
    LineStart   int     `json:"line_start"`
    LineEnd     int     `json:"line_end"`
    Score       float64 `json:"score"`
}

// 搜索选项
type SearchOptions struct {
    TopK        int      `json:"top_k"`
    Mode        string   `json:"mode"`        // hybrid/vector/fts
    Filter      string   `json:"filter"`      // 路径过滤
    TokenBudget int      `json:"token_budget"` // RAG模式Token预算
    MinScore    float64  `json:"min_score"`   // 最小分数阈值
    Collections []string `json:"collections"` // 限定集合
}

// 索引结果
type IndexResult struct {
    Total     int      `json:"total"`
    Indexed   int      `json:"indexed"`
    Skipped   int      `json:"skipped"`
    Failed    int      `json:"failed"`
    Errors    []string `json:"errors,omitempty"`
    Duration  int64    `json:"duration_ms"`
}

// 系统状态
type SystemStatus struct {
    Version       string    `json:"version"`
    Documents     int       `json:"documents"`
    Chunks        int       `json:"chunks"`
    StorageSize   int64     `json:"storage_size"`
    LastIndexed   time.Time `json:"last_indexed"`
    EmbeddingProvider string `json:"embedding_provider"`
    Uptime        int64     `json:"uptime_seconds"`
}
```

### 4.3 自建向量索引实现

**设计目标**：无需sqlite-vss扩展，保持"零配置"卖点，支持10万级向量高效搜索。

```go
package storage

import (
    "database/sql"
    "encoding/json"
    "math"
    "sort"
)

// 向量索引器
type VectorIndex struct {
    db        *sql.DB
    dimension int
    cache     map[string][]float32 // 内存缓存热点向量
}

// 存储向量
func (v *VectorIndex) Store(chunkID string, embedding []float32, model string) error {
    // 序列化为JSON
    data, err := json.Marshal(embedding)
    if err != nil {
        return err
    }
    
    _, err = v.db.Exec(`
        INSERT OR REPLACE INTO vectors (chunk_id, embedding, dimension, model)
        VALUES (?, ?, ?, ?)
    `, chunkID, data, len(embedding), model)
    
    return err
}

// 批量存储
func (v *VectorIndex) StoreBatch(chunkIDs []string, embeddings [][]float32, model string) error {
    tx, err := v.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    stmt, err := tx.Prepare(`
        INSERT OR REPLACE INTO vectors (chunk_id, embedding, dimension, model)
        VALUES (?, ?, ?, ?)
    `)
    if err != nil {
        return err
    }
    defer stmt.Close()
    
    for i, chunkID := range chunkIDs {
        data, _ := json.Marshal(embeddings[i])
        stmt.Exec(chunkID, data, len(embeddings[i]), model)
    }
    
    return tx.Commit()
}

// 向量搜索（余弦相似度）
func (v *VectorIndex) Search(queryVector []float32, topK int) ([]*VectorSearchResult, error) {
    // 1. 加载所有向量到内存（优化：可分批加载或预加载）
    rows, err := v.db.Query(`
        SELECT v.chunk_id, v.embedding, c.content_raw, c.heading_path, d.path
        FROM vectors v
        JOIN chunks c ON v.chunk_id = c.id
        JOIN documents d ON c.document_id = d.id
    `)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    // 2. 计算相似度
    results := []*VectorSearchResult{}
    for rows.Next() {
        var chunkID string
        var embeddingData []byte
        var contentRaw, headingPath, docPath string
        
        rows.Scan(&chunkID, &embeddingData, &contentRaw, &headingPath, &docPath)
        
        var embedding []float32
        json.Unmarshal(embeddingData, &embedding)
        
        // 余弦相似度
        similarity := cosineSimilarity(queryVector, embedding)
        
        results = append(results, &VectorSearchResult{
            ChunkID:     chunkID,
            Score:       similarity,
            ContentRaw:  contentRaw,
            HeadingPath: headingPath,
            DocPath:     docPath,
        })
    }
    
    // 3. 排序取TopK
    sort.Slice(results, func(i, j int) bool {
        return results[i].Score > results[j].Score
    })
    
    if len(results) > topK {
        results = results[:topK]
    }
    
    return results, nil
}

// 余弦相似度计算
func cosineSimilarity(a, b []float32) float64 {
    if len(a) != len(b) {
        return 0
    }
    
    var dotProduct, normA, normB float64
    for i := range a {
        dotProduct += float64(a[i]) * float64(b[i])
        normA += float64(a[i]) * float64(a[i])
        normB += float64(b[i]) * float64(b[i])
    }
    
    if normA == 0 || normB == 0 {
        return 0
    }
    
    return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

type VectorSearchResult struct {
    ChunkID     string
    Score       float64
    ContentRaw  string
    HeadingPath string
    DocPath     string
}
```

**性能优化路径**：
- MVP：全量加载内存搜索（10万向量<100MB内存，搜索<50ms）
- V1.1：HNSW索引（百万级向量，搜索<10ms）
- V1.2：量化压缩（减少内存占用50%）

### 4.4 缓存策略设计

**目标**：提升搜索性能，保证缓存一致性。

```go
package cache

import (
    "time"
    "github.com/patrickmn/go-cache"
)

type SearchCache struct {
    memoryCache  *cache.Cache
    ttl          time.Duration
    cleanupInt   time.Duration
}

func NewSearchCache(ttl, cleanup time.Duration) *SearchCache {
    return &SearchCache{
        memoryCache: cache.New(ttl, cleanup),
        ttl:        ttl,
        cleanupInt: cleanup,
    }
}

// 缓存策略配置
type CacheConfig struct {
    DefaultTTL      time.Duration `yaml:"default_ttl"`       // 默认过期时间，5分钟
    MaxItems        int           `yaml:"max_items"`         // 最大缓存条目，10000
    CleanupInterval time.Duration `yaml:"cleanup_interval"`  // 清理间隔，10分钟
}

// 缓存失效策略
type InvalidationPolicy struct {
    OnIndexUpdate   bool `yaml:"on_index_update"`   // 索引更新时失效，默认true
    OnEmbedUpdate   bool `yaml:"on_embed_update"`   // 向量更新时失效，默认true
    OnConfigChange  bool `yaml:"on_config_change"`  // 配置变更时失效，默认true
}

// 获取缓存
func (c *SearchCache) Get(query string, opts SearchOptions) ([]*SearchResult, bool) {
    key := c.generateKey(query, opts)
    if result, found := c.memoryCache.Get(key); found {
        metrics.CacheHits.Inc()
        return result.([]*SearchResult), true
    }
    metrics.CacheMisses.Inc()
    return nil, false
}

// 设置缓存
func (c *SearchCache) Set(query string, opts SearchOptions, results []*SearchResult) {
    key := c.generateKey(query, opts)
    c.memoryCache.Set(key, results, c.ttl)
}

// 失效相关查询的缓存
func (c *SearchCache) InvalidateForDocument(docID string) {
    // 遍历缓存，删除与该文档相关的所有查询
    for key, item := range c.memoryCache.Items() {
        results := item.Object.([]*SearchResult)
        for _, r := range results {
            if r.Chunk.DocumentID == docID {
                c.memoryCache.Delete(key)
                break
            }
        }
    }
}

// 失效所有缓存
func (c *SearchCache) InvalidateAll() {
    c.memoryCache.Flush()
}

// 生成缓存key
func (c *SearchCache) generateKey(query string, opts SearchOptions) string {
    // 基于查询+选项生成唯一key
    data := map[string]interface{}{
        "query":  query,
        "top_k":  opts.TopK,
        "mode":   opts.Mode,
        "filter": opts.Filter,
    }
    bytes, _ := json.Marshal(data)
    return fmt.Sprintf("%x", sha256.Sum256(bytes))
}
```

**缓存一致性规则**：

| 触发事件 | 缓存操作 | 说明 |
|----------|----------|------|
| 文档索引/更新 | 失效该文档相关查询 | 精准失效，不影响其他缓存 |
| 文档删除 | 失效该文档相关查询 | 同上 |
| 向量重新生成 | 失效所有缓存 | 保守策略，确保一致性 |
| 配置变更（分块参数等） | 失效所有缓存 | 保守策略 |
| 缓存达到上限 | LRU淘汰 | go-cache默认策略 |
| 缓存过期 | 自动清理 | 定期清理过期条目 |

### 4.5 SQLite备份与恢复

```go
package storage

import (
    "io"
    "os"
    "path/filepath"
    "time"
    
    "go.uber.org/zap"
)

type BackupManager struct {
    dbPath      string
    backupDir   string
    maxBackups  int
    logger      *zap.Logger
}

// 创建备份
func (b *BackupManager) CreateBackup() (string, error) {
    timestamp := time.Now().Format("20060102-150405")
    backupPath := filepath.Join(b.backupDir, fmt.Sprintf("cortex-%s.db", timestamp))
    
    // 复制数据库文件
    if err := copyFile(b.dbPath, backupPath); err != nil {
        return "", err
    }
    
    // 清理旧备份
    b.cleanOldBackups()
    
    b.logger.Info("backup created", zap.String("path", backupPath))
    return backupPath, nil
}

// 恢复备份
func (b *BackupManager) RestoreBackup(backupPath string) error {
    // 验证备份文件
    if _, err := os.Stat(backupPath); os.IsNotExist(err) {
        return fmt.Errorf("backup file not found: %s", backupPath)
    }
    
    // 备份当前数据库
    currentBackup := b.dbPath + ".restore-backup"
    if err := copyFile(b.dbPath, currentBackup); err != nil {
        return fmt.Errorf("failed to backup current db: %w", err)
    }
    
    // 恢复备份
    if err := copyFile(backupPath, b.dbPath); err != nil {
        // 恢复失败，还原
        copyFile(currentBackup, b.dbPath)
        return fmt.Errorf("failed to restore: %w", err)
    }
    
    // 清理临时备份
    os.Remove(currentBackup)
    
    b.logger.Info("backup restored", zap.String("path", backupPath))
    return nil
}

// 列出所有备份
func (b *BackupManager) ListBackups() ([]BackupInfo, error) {
    files, err := os.ReadDir(b.backupDir)
    if err != nil {
        return nil, err
    }
    
    backups := []BackupInfo{}
    for _, f := range files {
        if filepath.Ext(f.Name()) == ".db" {
            info, _ := f.Info()
            backups = append(backups, BackupInfo{
                Path:    filepath.Join(b.backupDir, f.Name()),
                Size:    info.Size(),
                ModTime: info.ModTime(),
            })
        }
    }
    
    // 按时间倒序
    sort.Slice(backups, func(i, j int) bool {
        return backups[i].ModTime.After(backups[j].ModTime)
    })
    
    return backups, nil
}

// 自动备份（定时任务）
func (b *BackupManager) StartAutoBackup(interval time.Duration) {
    ticker := time.NewTicker(interval)
    go func() {
        for range ticker.C {
            b.CreateBackup()
        }
    }()
}

// 清理旧备份
func (b *BackupManager) cleanOldBackups() {
    backups, _ := b.ListBackups()
    if len(backups) > b.maxBackups {
        for i := b.maxBackups; i < len(backups); i++ {
            os.Remove(backups[i].Path)
            b.logger.Info("old backup removed", zap.String("path", backups[i].Path))
        }
    }
}

func copyFile(src, dst string) error {
    sourceFile, err := os.Open(src)
    if err != nil {
        return err
    }
    defer sourceFile.Close()
    
    destFile, err := os.Create(dst)
    if err != nil {
        return err
    }
    defer destFile.Close()
    
    _, err = io.Copy(destFile, sourceFile)
    return err
}

type BackupInfo struct {
    Path    string
    Size    int64
    ModTime time.Time
}
```

**备份配置**：

```yaml
backup:
  enabled: true
  dir: "~/.cortex/backups"
  max_backups: 10           # 保留最近10个备份
  auto_backup: true
  interval: 24h             # 每天自动备份
```

**CLI命令**：

```bash
cortex backup create              # 创建备份
cortex backup list                # 列出所有备份
cortex backup restore <path>      # 恢复备份
cortex backup restore --latest    # 恢复最新备份
```

---

### 5.1 层级分块算法（完整版）

```go
package chunker

import (
    "fmt"
    "strings"
    "github.com/yuin/goldmark"
    "github.com/yuin/goldmark/ast"
    "github.com/yuin/goldmark/text"
    "github.com/pkoukk/tiktoken-go"
)

// 分块配置
type ChunkConfig struct {
    MaxTokens        int  `yaml:"max_tokens"`         // 单块最大Token，默认512
    OverlapTokens    int  `yaml:"overlap_tokens"`     // 重叠Token，默认64
    MinChars         int  `yaml:"min_chars"`          // 最小字符数，默认50
    IncludeBreadcrumb bool `yaml:"include_breadcrumb"` // 是否注入标题路径，默认true
    Tokenizer        string `yaml:"tokenizer"`        // tiktoken模型，默认cl100k_base
}

// Markdown分块器
type MarkdownChunker struct {
    config    ChunkConfig
    tokenizer *tiktoken.Tiktoken
    md        goldmark.Markdown
}

func NewMarkdownChunker(config ChunkConfig) (*MarkdownChunker, error) {
    // 初始化tiktoken
    tokenizer, err := tiktoken.GetEncoding(config.Tokenizer)
    if err != nil {
        return nil, fmt.Errorf("failed to init tiktoken: %w", err)
    }
    
    return &MarkdownChunker{
        config:    config,
        tokenizer: tokenizer,
        md:        goldmark.New(),
    }, nil
}

// 分块入口
func (c *MarkdownChunker) Chunk(content string, path string) ([]*Chunk, error) {
    // 1. 解析Markdown AST
    reader := text.NewReader([]byte(content))
    doc := ast.NewDocument()
    if err := c.md.Parser().Parse(reader, doc); err != nil {
        return nil, fmt.Errorf("failed to parse markdown: %w", err)
    }
    
    // 2. 按标题层级提取Section
    sections := c.extractSections(doc, content, path)
    
    // 3. 处理每个Section
    chunks := []*Chunk{}
    for _, section := range sections {
        sectionChunks := c.processSection(section)
        chunks = append(chunks, sectionChunks...)
    }
    
    // 4. 过滤短块
    chunks = c.filterShortChunks(chunks)
    
    // 5. 添加重叠
    chunks = c.addOverlap(chunks)
    
    // 6. 计算Token（使用tiktoken精确计算）
    for _, chunk := range chunks {
        chunk.TokenCount = len(c.tokenizer.Encode(chunk.ContentRaw, nil, nil))
    }
    
    return chunks, nil
}

// Section结构
type Section struct {
    Path        string
    HeadingPath string
    HeadingLevel int
    Content     string
    ContentRaw  string
    LineStart   int
    LineEnd     int
    CharStart   int
    CharEnd     int
}

// 提取Sections
func (c *MarkdownChunker) extractSections(doc ast.Node, content string, path string) []*Section {
    sections := []*Section{}
    headingStack := []string{} // 标题栈，追踪当前标题路径
    
    source := []byte(content)
    
    ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
        if !entering {
            return ast.WalkContinue, nil
        }
        
        switch n.Kind() {
        case ast.KindHeading:
            heading := n.(*ast.Heading)
            headingText := string(heading.Text(source))
            
            // 更新标题栈
            if heading.Level <= len(headingStack) {
                headingStack = headingStack[:heading.Level-1]
            }
            headingStack = append(headingStack, headingText)
            
        case ast.KindParagraph, ast.KindCodeBlock, ast.KindFencedCodeBlock,
             ast.KindList, ast.KindBlockquote, ast.KindTable:
            // 收集内容块
            section := c.createSection(n, source, path, headingStack)
            if section != nil {
                sections = append(sections, section)
            }
        }
        
        return ast.WalkContinue, nil
    })
    
    return sections
}

// 处理单个Section
func (c *MarkdownChunker) processSection(section *Section) []*Chunk {
    // 精确计算Token
    tokens := len(c.tokenizer.Encode(section.ContentRaw, nil, nil))
    
    if tokens <= c.config.MaxTokens {
        // 直接成块
        chunk := c.createChunk(section)
        return []*Chunk{chunk}
    }
    
    // 超长，按段落边界再分
    return c.splitByParagraph(section)
}

// 按段落分割
func (c *MarkdownChunker) splitByParagraph(section *Section) []*Chunk {
    paragraphs := strings.Split(section.ContentRaw, "\n\n")
    chunks := []*Chunk{}
    
    currentContent := ""
    currentTokens := 0
    
    for _, para := range paragraphs {
        paraTokens := len(c.tokenizer.Encode(para, nil, nil))
        
        if currentTokens + paraTokens > c.config.MaxTokens {
            // 保存当前块
            if currentContent != "" {
                chunk := c.createChunkFromContent(section, currentContent)
                chunks = append(chunks, chunk)
            }
            currentContent = para
            currentTokens = paraTokens
        } else {
            if currentContent != "" {
                currentContent += "\n\n" + para
            } else {
                currentContent = para
            }
            currentTokens += paraTokens
        }
    }
    
    // 保存最后一块
    if currentContent != "" {
        chunk := c.createChunkFromContent(section, currentContent)
        chunks = append(chunks, chunk)
    }
    
    return chunks
}

// 创建分块
func (c *MarkdownChunker) createChunk(section *Section) *Chunk {
    content := section.ContentRaw
    if c.config.IncludeBreadcrumb && section.HeadingLevel > 0 {
        breadcrumb := fmt.Sprintf("Section: %s\n\n", section.HeadingPath)
        content = breadcrumb + section.ContentRaw
    }
    
    return &Chunk{
        ID:           generateChunkID(section.Path, section.CharStart),
        HeadingPath:  section.HeadingPath,
        HeadingLevel: section.HeadingLevel,
        Content:      content,
        ContentRaw:   section.ContentRaw,
        LineStart:    section.LineStart,
        LineEnd:      section.LineEnd,
        CharStart:    section.CharStart,
        CharEnd:      section.CharEnd,
    }
}

// 过滤短块
func (c *MarkdownChunker) filterShortChunks(chunks []*Chunk) []*Chunk {
    filtered := []*Chunk{}
    for _, chunk := range chunks {
        if len(chunk.ContentRaw) >= c.config.MinChars {
            filtered = append(filtered, chunk)
        }
    }
    return filtered
}

// 添加重叠
func (c *MarkdownChunker) addOverlap(chunks []*Chunk) []*Chunk {
    if len(chunks) <= 1 || c.config.OverlapTokens == 0 {
        return chunks
    }
    
    for i := 1; i < len(chunks); i++ {
        // 从前一个块末尾取overlap部分
        prevChunk := chunks[i-1]
        tokens := c.tokenizer.Encode(prevChunk.ContentRaw, nil, nil)
        
        if len(tokens) > c.config.OverlapTokens {
            overlapTokens := tokens[len(tokens)-c.config.OverlapTokens:]
            overlapText := c.tokenizer.Decode(overlapTokens)
            
            // 添加到当前块开头
            chunks[i].ContentRaw = overlapText + "\n\n" + chunks[i].ContentRaw
        }
    }
    
    return chunks
}
```

### 5.2 PDF分块器（pdfcpu实现）

**选型说明**：
- 使用pdfcpu（MIT协议）替代unipdf
- pdfcpu开源免费，无商用授权问题
- 缺点：不支持OCR，表格提取精度一般
- 满足基础PDF文本提取需求

```go
package chunker

import (
    "bytes"
    "fmt"
    "strings"
    
    "github.com/pdfcpu/pdfcpu/pkg/api"
    "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

type PDFChunker struct {
    config    ChunkConfig
    tokenizer *tiktoken.Tiktoken
}

func NewPDFChunker(config ChunkConfig) (*PDFChunker, error) {
    tokenizer, err := tiktoken.GetEncoding(config.Tokenizer)
    if err != nil {
        return nil, err
    }
    
    return &PDFChunker{
        config:    config,
        tokenizer: tokenizer,
    }, nil
}

func (c *PDFChunker) Chunk(content []byte, path string) ([]*Chunk, error) {
    // 1. 使用pdfcpu提取文本
    // pdfcpu通过命令行工具提取文本
    ctx, err := api.ReadContext(bytes.NewReader(content), nil)
    if err != nil {
        return nil, fmt.Errorf("failed to read PDF: %w", err)
    }
    
    // 2. 提取所有页面文本
    chunks := []*Chunk{}
    
    // 获取页数
    pageCount, err := api.PageCount(ctx)
    if err != nil {
        return nil, err
    }
    
    for pageNum := 1; pageNum <= pageCount; pageNum++ {
        // 提取单页文本
        var buf bytes.Buffer
        if err := api.ExtractContent(ctx, []string{fmt.Sprintf("%d", pageNum)}, &buf); err != nil {
            c.logger.Warn("failed to extract page",
                zap.String("path", path),
                zap.Int("page", pageNum),
                zap.Error(err))
            continue
        }
        
        text := buf.String()
        if len(strings.TrimSpace(text)) == 0 {
            continue
        }
        
        // 3. 按段落分块
        paragraphs := strings.Split(text, "\n\n")
        for i, para := range paragraphs {
            para = strings.TrimSpace(para)
            if len(para) < c.config.MinChars {
                continue
            }
            
            // 检查Token数量
            tokens := len(c.tokenizer.Encode(para, nil, nil))
            
            chunk := &Chunk{
                ID:           fmt.Sprintf("%s-p%d-%d", generateID(path), pageNum, i),
                ContentRaw:   para,
                HeadingPath:  fmt.Sprintf("Page %d", pageNum),
                HeadingLevel: 1,
                TokenCount:   tokens,
            }
            
            // 如果超过最大Token，需要进一步分割
            if tokens > c.config.MaxTokens {
                subChunks := c.splitLongParagraph(para, path, pageNum, i)
                chunks = append(chunks, subChunks...)
            } else {
                chunks = append(chunks, chunk)
            }
        }
    }
    
    return chunks, nil
}

// 分割超长段落
func (c *PDFChunker) splitLongParagraph(para string, path string, pageNum, paraIndex int) []*Chunk {
    sentences := splitBySentence(para)
    chunks := []*Chunk{}
    
    currentContent := ""
    currentTokens := 0
    
    for _, sentence := range sentences {
        sentenceTokens := len(c.tokenizer.Encode(sentence, nil, nil))
        
        if currentTokens+sentenceTokens > c.config.MaxTokens {
            if currentContent != "" {
                chunks = append(chunks, &Chunk{
                    ID:           fmt.Sprintf("%s-p%d-%d-%d", generateID(path), pageNum, paraIndex, len(chunks)),
                    ContentRaw:   currentContent,
                    HeadingPath:  fmt.Sprintf("Page %d", pageNum),
                    HeadingLevel: 1,
                    TokenCount:   currentTokens,
                })
            }
            currentContent = sentence
            currentTokens = sentenceTokens
        } else {
            currentContent += " " + sentence
            currentTokens += sentenceTokens
        }
    }
    
    // 最后一块
    if currentContent != "" {
        chunks = append(chunks, &Chunk{
            ID:           fmt.Sprintf("%s-p%d-%d-%d", generateID(path), pageNum, paraIndex, len(chunks)),
            ContentRaw:   currentContent,
            HeadingPath:  fmt.Sprintf("Page %d", pageNum),
            HeadingLevel: 1,
            TokenCount:   currentTokens,
        })
    }
    
    return chunks
}

// 按句子分割
func splitBySentence(text string) []string {
    // 简单实现：按句号、问号、感叹号分割
    // 生产环境可用更复杂的NLP分割
    delimiters := []string{"。", "！", "？", ".", "!", "?"}
    
    sentences := []string{}
    current := ""
    
    for _, ch := range text {
        current += string(ch)
        for _, d := range delimiters {
            if string(ch) == d {
                sentences = append(sentences, current)
                current = ""
                break
            }
        }
    }
    
    if current != "" {
        sentences = append(sentences, current)
    }
    
    return sentences
}

func (c *PDFChunker) Name() string {
    return "pdf"
}
```

**pdfcpu局限性**：
- 不支持OCR（扫描件PDF无法提取文本）
- 表格提取精度一般（无法保持表格结构）
- 图片中的文字无法提取
- 复杂排版可能丢失格式

**替代方案（如需更高级功能）**：
- OCR需求：集成Tesseract（开源）
- 表格提取：使用tabula-py（Python库，通过CGO调用）
- 生产级方案：考虑商业化API（如AWS Textract、Google Document AI）
```

### 5.3 增量索引算法（完整版）

```go
package index

import (
    "crypto/sha256"
    "encoding/hex"
    "io/ioutil"
    "os"
    "path/filepath"
    "strings"
    "time"
    
    "github.com/fsnotify/fsnotify"
    "github.com/moss/cortex/internal/chunker"
    "github.com/moss/cortex/internal/embedding"
    "github.com/moss/cortex/internal/storage"
    "go.uber.org/zap"
)

type Indexer struct {
    storage   storage.Storage
    chunker   map[string]chunker.Chunker // 按文件类型选择chunker
    embedding embedding.EmbeddingProvider
    logger    *zap.Logger
    watcher   *fsnotify.Watcher
}

// 索引目录
func (idx *Indexer) IndexDirectory(rootPath string, recursive bool) (*IndexResult, error) {
    start := time.Now()
    result := &IndexResult{}
    
    // 1. 遍历文件
    files, err := idx.walkFiles(rootPath, recursive)
    if err != nil {
        return nil, err
    }
    result.Total = len(files)
    
    // 2. 批量处理（并行）
    type fileResult struct {
        indexed bool
        skipped bool
        err     error
    }
    
    resultCh := make(chan fileResult, len(files))
    sem := make(chan struct{}, 4) // 并发限制
    
    for _, file := range files {
        sem <- struct{}{}
        go func(path string) {
            defer func() { <-sem }()
            
            indexed, skipped, err := idx.indexFile(path)
            resultCh <- fileResult{indexed: indexed, skipped: skipped, err: err}
        }(file)
    }
    
    // 收集结果
    for i := 0; i < len(files); i++ {
        res := <-resultCh
        if res.indexed {
            result.Indexed++
        }
        if res.skipped {
            result.Skipped++
        }
        if res.err != nil {
            result.Failed++
            result.Errors = append(result.Errors, res.err.Error())
        }
    }
    
    result.Duration = time.Since(start).Milliseconds()
    idx.logger.Info("index completed",
        zap.Int("total", result.Total),
        zap.Int("indexed", result.Indexed),
        zap.Int("skipped", result.Skipped),
        zap.Int64("duration_ms", result.Duration),
    )
    
    return result, nil
}

// 索引单个文件
func (idx *Indexer) indexFile(path string) (indexed bool, skipped bool, err error) {
    // 1. 计算文件hash
    content, err := ioutil.ReadFile(path)
    if err != nil {
        return false, false, err
    }
    
    hash := sha256.Sum256(content)
    hashStr := hex.EncodeToString(hash[:])
    
    // 2. 查询数据库
    doc, err := idx.storage.GetDocumentByPath(path)
    if err == nil && doc.ContentHash == hashStr {
        // 未修改，跳过
        return false, true, nil
    }
    
    // 3. 删除旧分块
    if doc != nil {
        if err := idx.storage.DeleteChunksByDocument(doc.ID); err != nil {
            idx.logger.Error("failed to delete old chunks", 
                zap.String("path", path),
                zap.Error(err))
        }
    }
    
    // 4. 选择chunker
    fileType := idx.getFileType(path)
    chunker, ok := idx.chunker[fileType]
    if !ok {
        return false, false, fmt.Errorf("unsupported file type: %s", fileType)
    }
    
    // 5. 分块
    chunks, err := chunker.Chunk(string(content), path)
    if err != nil {
        return false, false, err
    }
    
    // 6. 批量生成向量
    texts := make([]string, len(chunks))
    for i, c := range chunks {
        texts[i] = c.ContentRaw
    }
    
    embeddings, err := idx.embedding.EmbedBatch(texts)
    if err != nil {
        return false, false, err
    }
    
    // 7. 填充向量
    for i, c := range chunks {
        c.Embedding = embeddings[i]
        c.EmbeddingModel = idx.embedding.Name()
    }
    
    // 8. 保存文档
    docID := hashStr[:16]
    newDoc := &models.Document{
        ID:          docID,
        Path:        path,
        FileType:    fileType,
        ContentHash: hashStr,
        FileSize:    int64(len(content)),
        ChunkCount:  len(chunks),
        Status:      "indexed",
    }
    
    if err := idx.storage.SaveDocument(newDoc); err != nil {
        return false, false, err
    }
    
    // 9. 保存分块
    for _, c := range chunks {
        c.DocumentID = docID
    }
    if err := idx.storage.SaveChunks(chunks); err != nil {
        return false, false, err
    }
    
    return true, false, nil
}

// 启动文件监听
func (idx *Indexer) StartWatcher(rootPath string) error {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return err
    }
    idx.watcher = watcher
    
    // 添加监听
    err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if info.IsDir() {
            return watcher.Add(path)
        }
        return nil
    })
    if err != nil {
        return err
    }
    
    // 处理事件
    go func() {
        for {
            select {
            case event, ok := <-watcher.Events:
                if !ok {
                    return
                }
                if event.Op&fsnotify.Write == fsnotify.Write ||
                   event.Op&fsnotify.Create == fsnotify.Create {
                    idx.logger.Info("file changed, reindexing", 
                        zap.String("path", event.Name))
                    idx.indexFile(event.Name)
                }
                if event.Op&fsnotify.Remove == fsnotify.Remove {
                    idx.logger.Info("file removed, cleaning", 
                        zap.String("path", event.Name))
                    idx.storage.DeleteDocumentByPath(event.Name)
                }
                
            case err, ok := <-watcher.Errors:
                if !ok {
                    return
                }
                idx.logger.Error("watcher error", zap.Error(err))
            }
        }
    }()
    
    idx.logger.Info("file watcher started", zap.String("root", rootPath))
    return nil
}

// 获取文件类型
func (idx *Indexer) getFileType(path string) string {
    ext := strings.ToLower(filepath.Ext(path))
    switch ext {
    case ".md", ".markdown":
        return "md"
    case ".pdf":
        return "pdf"
    case ".docx", ".doc":
        return "docx"
    case ".txt":
        return "txt"
    default:
        return "unknown"
    }
}

// 遍历文件
func (idx *Indexer) walkFiles(rootPath string, recursive bool) ([]string, error) {
    files := []string{}
    
    err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if info.IsDir() {
            if !recursive && path != rootPath {
                return filepath.SkipDir
            }
            return nil
        }
        
        // 过滤支持的文件类型
        if idx.isSupportedFile(path) {
            files = append(files, path)
        }
        return nil
    })
    
    return files, err
}
```

### 5.4 混合搜索算法（完整版，带缓存）

```go
package search

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "sync"
    "time"
    
    "github.com/moss/cortex/internal/models"
    "github.com/moss/cortex/internal/storage"
    "github.com/patrickmn/go-cache"
    "go.uber.org/zap"
)

type HybridSearchEngine struct {
    storage   storage.Storage
    embedding embedding.EmbeddingProvider
    cache     *cache.Cache
    logger    *zap.Logger
}

// 搜索入口
func (s *HybridSearchEngine) Search(
    ctx context.Context,
    query string,
    opts models.SearchOptions,
) ([]*models.SearchResult, error) {
    start := time.Now()
    
    // 1. 检查缓存
    cacheKey := s.getCacheKey(query, opts)
    if cached, found := s.cache.Get(cacheKey); found {
        s.logger.Debug("cache hit", zap.String("query", query))
        return cached.([]*models.SearchResult), nil
    }
    
    // 2. 根据模式选择搜索方式
    var results []*models.SearchResult
    var err error
    
    switch opts.Mode {
    case "vector":
        results, err = s.vectorSearch(ctx, query, opts)
    case "fts":
        results, err = s.ftsSearch(ctx, query, opts)
    case "hybrid":
        results, err = s.hybridSearch(ctx, query, opts)
    default:
        results, err = s.hybridSearch(ctx, query, opts)
    }
    
    if err != nil {
        return nil, err
    }
    
    // 3. 分数归一化
    results = s.normalizeScores(results)
    
    // 4. 排序
    s.sortResults(results)
    
    // 5. TopK截断
    if len(results) > opts.TopK {
        results = results[:opts.TopK]
    }
    
    // 6. 填充排名
    for i := range results {
        results[i].Rank = i + 1
    }
    
    // 7. 写入缓存
    s.cache.Set(cacheKey, results, cache.DefaultExpiration)
    
    s.logger.Info("search completed",
        zap.String("query", query),
        zap.String("mode", opts.Mode),
        zap.Int("results", len(results)),
        zap.Int64("duration_ms", time.Since(start).Milliseconds()),
    )
    
    return results, nil
}

// 混合搜索
func (s *HybridSearchEngine) hybridSearch(
    ctx context.Context,
    query string,
    opts models.SearchOptions,
) ([]*models.SearchResult, error) {
    // 并行执行向量搜索和全文搜索
    var wg sync.WaitGroup
    var vectorResults, ftsResults []*models.SearchResult
    var vectorErr, ftsErr error
    
    wg.Add(2)
    
    // 向量搜索
    go func() {
        defer wg.Done()
        vectorResults, vectorErr = s.vectorSearch(ctx, query, opts)
    }()
    
    // 全文搜索
    go func() {
        defer wg.Done()
        ftsResults, ftsErr = s.ftsSearch(ctx, query, opts)
    }()
    
    wg.Wait()
    
    if vectorErr != nil {
        s.logger.Warn("vector search failed", zap.Error(vectorErr))
    }
    if ftsErr != nil {
        s.logger.Warn("fts search failed", zap.Error(ftsErr))
    }
    
    // 如果两种搜索都失败，返回错误
    if vectorErr != nil && ftsErr != nil {
        return nil, fmt.Errorf("both search failed: vector=%v, fts=%v", vectorErr, ftsErr)
    }
    
    // 如果只有一种成功，直接返回
    if vectorErr != nil {
        return ftsResults, nil
    }
    if ftsErr != nil {
        return vectorResults, nil
    }
    
    // RRF融合
    return s.rrfMerge(vectorResults, ftsResults), nil
}

// 向量搜索
func (s *HybridSearchEngine) vectorSearch(
    ctx context.Context,
    query string,
    opts models.SearchOptions,
) ([]*models.SearchResult, error) {
    // 1. 生成查询向量
    queryVector, err := s.embedding.Embed(query)
    if err != nil {
        return nil, err
    }
    
    // 2. 向量搜索
    topK := opts.TopK * 3 // 取更多候选
    results, err := s.storage.VectorSearch(queryVector, topK)
    if err != nil {
        return nil, err
    }
    
    // 3. 填充分数
    for _, r := range results {
        r.VectorScore = r.Score
    }
    
    return results, nil
}

// 全文搜索
func (s *HybridSearchEngine) ftsSearch(
    ctx context.Context,
    query string,
    opts models.SearchOptions,
) ([]*models.SearchResult, error) {
    topK := opts.TopK * 3
    results, err := s.storage.FTSSearch(query, topK)
    if err != nil {
        return nil, err
    }
    
    for _, r := range results {
        r.FTSScore = r.Score
    }
    
    return results, nil
}

// RRF融合
func (s *HybridSearchEngine) rrfMerge(
    vectorResults []*models.SearchResult,
    ftsResults []*models.SearchResult,
) []*models.SearchResult {
    k := 60 // RRF参数
    
    scores := make(map[string]float64)
    results := make(map[string]*models.SearchResult)
    
    // 向量结果
    for rank, r := range vectorResults {
        id := r.Chunk.ID
        scores[id] += 1.0 / float64(k + rank + 1)
        if _, exists := results[id]; !exists {
            results[id] = r
        }
    }
    
    // 全文结果
    for rank, r := range ftsResults {
        id := r.Chunk.ID
        scores[id] += 1.0 / float64(k + rank + 1)
        if _, exists := results[id]; !exists {
            results[id] = r
        }
    }
    
    // 组装结果
    merged := make([]*models.SearchResult, 0, len(results))
    for id, score := range scores {
        r := results[id]
        r.Score = score
        merged = append(merged, r)
    }
    
    return merged
}

// 分数归一化
func (s *HybridSearchEngine) normalizeScores(results []*models.SearchResult) []*models.SearchResult {
    if len(results) == 0 {
        return results
    }
    
    // 找到最大分数
    maxScore := 0.0
    for _, r := range results {
        if r.Score > maxScore {
            maxScore = r.Score
        }
    }
    
    // 归一化到0-1
    if maxScore > 0 {
        for _, r := range results {
            r.Score = r.Score / maxScore
        }
    }
    
    return results
}

// 排序
func (s *HybridSearchEngine) sortResults(results []*models.SearchResult) {
    sort.Slice(results, func(i, j int) bool {
        return results[i].Score > results[j].Score
    })
}

// 生成缓存key
func (s *HybridSearchEngine) getCacheKey(query string, opts models.SearchOptions) string {
    data := map[string]interface{}{
        "query":  query,
        "top_k":  opts.TopK,
        "mode":   opts.Mode,
        "filter": opts.Filter,
    }
    bytes, _ := json.Marshal(data)
    hash := sha256.Sum256(bytes)
    return hex.EncodeToString(hash[:])
}
```

### 5.5 RAG上下文组装（完整版）

```go
package rag

import (
    "context"
    "github.com/moss/cortex/internal/models"
    "github.com/moss/cortex/internal/search"
    "github.com/pkoukk/tiktoken-go"
)

type RAGBuilder struct {
    search   *search.HybridSearchEngine
    tokenizer *tiktoken.Tiktoken
}

// 构建RAG上下文
func (b *RAGBuilder) BuildContext(
    ctx context.Context,
    query string,
    tokenBudget int,
    opts models.SearchOptions,
) (*models.RAGContext, error) {
    // 1. 搜索相关分块
    opts.TopK = 50 // 取更多候选
    results, err := b.search.Search(ctx, query, opts)
    if err != nil {
        return nil, err
    }
    
    // 2. 组装上下文
    context := strings.Builder{}
    sources := []models.Source{}
    totalTokens := 0
    truncated := false
    
    for _, r := range results {
        // 计算Token
        chunkTokens := r.Chunk.TokenCount
        
        // 预留分隔符Token
        separatorTokens := 2
        if context.Len() > 0 {
            if totalTokens + separatorTokens + chunkTokens > tokenBudget {
                // 超出预算，标记截断
                truncated = true
                break
            }
            
            // 添加分隔符
            context.WriteString("\n\n---\n\n")
            totalTokens += separatorTokens
        }
        
        // 写入内容
        context.WriteString(r.Chunk.Content)
        totalTokens += chunkTokens
        
        // 记录来源
        sources = append(sources, models.Source{
            Path:        r.Chunk.Document.Path,
            HeadingPath: r.Chunk.HeadingPath,
            LineStart:   r.Chunk.LineStart,
            LineEnd:     r.Chunk.LineEnd,
            Score:       r.Score,
        })
    }
    
    return &models.RAGContext{
        Context:     context.String(),
        TokenCount:  totalTokens,
        TokenBudget: tokenBudget,
        Sources:     sources,
        Truncated:   truncated,
    }, nil
}

// 带截断策略的上下文组装
func (b *RAGBuilder) BuildContextWithTruncation(
    ctx context.Context,
    query string,
    tokenBudget int,
    truncationStrategy string, // "smart" / "simple"
    opts models.SearchOptions,
) (*models.RAGContext, error) {
    // 先尝试正常组装
    ragContext, err := b.BuildContext(ctx, query, tokenBudget, opts)
    if err != nil {
        return nil, err
    }
    
    // 如果超出预算，应用截断策略
    if ragContext.Truncated && truncationStrategy == "smart" {
        // 智能截断：尝试总结超长分块
        ragContext = b.applySmartTruncation(ragContext, tokenBudget)
    }
    
    return ragContext, nil
}
```

---

## 六、Embedding提供商实现

### 6.1 Ollama本地嵌入

```go
package embedding

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
    
    "go.uber.org/zap"
)

type OllamaEmbedding struct {
    BaseURL   string
    Model     string
    Dimension int
    Timeout   time.Duration
    logger    *zap.Logger
}

func NewOllamaEmbedding(baseURL, model string, dimension int, logger *zap.Logger) *OllamaEmbedding {
    return &OllamaEmbedding{
        BaseURL:   baseURL,
        Model:     model,
        Dimension: dimension,
        Timeout:   30 * time.Second,
        logger:    logger,
    }
}

type ollamaRequest struct {
    Model  string   `json:"model"`
    Prompt string   `json:"prompt"`
}

type ollamaBatchRequest struct {
    Model   string   `json:"model"`
    Prompts []string `json:"prompts"`
}

type ollamaResponse struct {
    Embedding []float32 `json:"embedding"`
}

type ollamaBatchResponse struct {
    Embeddings [][]float32 `json:"embeddings"`
}

func (o *OllamaEmbedding) Embed(text string) ([]float32, error) {
    req := ollamaRequest{
        Model:  o.Model,
        Prompt: text,
    }
    
    body, err := json.Marshal(req)
    if err != nil {
        return nil, err
    }
    
    resp, err := o.doRequest("/api/embeddings", body)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result ollamaResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return result.Embedding, nil
}

func (o *OllamaEmbedding) EmbedBatch(texts []string) ([][]float32, error) {
    // Ollama不支持批量API，需要逐个调用
    // 使用goroutine并行化
    results := make([][]float32, len(texts))
    errors := make([]error, len(texts))
    
    var wg sync.WaitGroup
    sem := make(chan struct{}, 4) // 并发限制
    
    for i, text := range texts {
        wg.Add(1)
        go func(idx int, txt string) {
            defer wg.Done()
            sem <- struct{}{}
            defer func() { <-sem }()
            
            emb, err := o.Embed(txt)
            results[idx] = emb
            errors[idx] = err
        }(i, text)
    }
    
    wg.Wait()
    
    // 检查错误
    for _, err := range errors {
        if err != nil {
            return nil, err
        }
    }
    
    return results, nil
}

func (o *OllamaEmbedding) Dimension() int {
    return o.Dimension
}

func (o *OllamaEmbedding) Name() string {
    return fmt.Sprintf("ollama:%s", o.Model)
}

func (o *OllamaEmbedding) Health() error {
    resp, err := http.Get(o.BaseURL + "/api/tags")
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("ollama health check failed: %d", resp.StatusCode)
    }
    
    return nil
}

func (o *OllamaEmbedding) doRequest(endpoint string, body []byte) (*http.Response, error) {
    client := &http.Client{Timeout: o.Timeout}
    
    req, err := http.NewRequest("POST", o.BaseURL+endpoint, bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/json")
    
    return client.Do(req)
}
```

### 6.2 OpenAI云端嵌入（Fallback）

```go
package embedding

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
    
    "go.uber.org/zap"
)

type OpenAIEmbedding struct {
    APIKey    string
    Model     string // text-embedding-3-small / text-embedding-3-large
    Dimension int
    Timeout   time.Duration
    logger    *zap.Logger
}

func NewOpenAIEmbedding(apiKey, model string, dimension int, logger *zap.Logger) *OpenAIEmbedding {
    return &OpenAIEmbedding{
        APIKey:    apiKey,
        Model:     model,
        Dimension: dimension,
        Timeout:   30 * time.Second,
        logger:    logger,
    }
}

type openaiRequest struct {
    Input []string `json:"input"`
    Model string   `json:"model"`
}

type openaiResponse struct {
    Data []struct {
        Embedding []float32 `json:"embedding"`
        Index     int       `json:"index"`
    } `json:"data"`
    Usage struct {
        TotalTokens int `json:"total_tokens"`
    } `json:"usage"`
    Error *struct {
        Message string `json:"message"`
    } `json:"error,omitempty"`
}

func (o *OpenAIEmbedding) Embed(text string) ([]float32, error) {
    results, err := o.EmbedBatch([]string{text})
    if err != nil {
        return nil, err
    }
    return results[0], nil
}

func (o *OpenAIEmbedding) EmbedBatch(texts []string) ([][]float32, error) {
    // OpenAI支持批量，每次最多2048个文本
    if len(texts) > 2048 {
        // 分批处理
        allResults := [][]float32{}
        for i := 0; i < len(texts); i += 2048 {
            end := i + 2048
            if end > len(texts) {
                end = len(texts)
            }
            
            batch := texts[i:end]
            results, err := o.embedBatch(batch)
            if err != nil {
                return nil, err
            }
            allResults = append(allResults, results...)
        }
        return allResults, nil
    }
    
    return o.embedBatch(texts)
}

func (o *OpenAIEmbedding) embedBatch(texts []string) ([][]float32, error) {
    req := openaiRequest{
        Input: texts,
        Model: o.Model,
    }
    
    body, err := json.Marshal(req)
    if err != nil {
        return nil, err
    }
    
    client := &http.Client{Timeout: o.Timeout}
    httpReq, err := http.NewRequest("POST", "https://api.openai.com/v1/embeddings", bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Authorization", "Bearer "+o.APIKey)
    
    resp, err := client.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result openaiResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    if result.Error != nil {
        return nil, fmt.Errorf("openai api error: %s", result.Error.Message)
    }
    
    // 按index排序
    embeddings := make([][]float32, len(texts))
    for _, data := range result.Data {
        embeddings[data.Index] = data.Embedding
    }
    
    o.logger.Debug("openai embedding completed",
        zap.Int("count", len(texts)),
        zap.Int("tokens", result.Usage.TotalTokens),
    )
    
    return embeddings, nil
}

func (o *OpenAIEmbedding) Dimension() int {
    return o.Dimension
}

func (o *OpenAIEmbedding) Name() string {
    return fmt.Sprintf("openai:%s", o.Model)
}

func (o *OpenAIEmbedding) Health() error {
    // OpenAI无需健康检查
    return nil
}
```

### 6.3 Embedding提供商管理器（带Fallback）

```go
package embedding

import (
    "fmt"
    "go.uber.org/zap"
)

type ProviderManager struct {
    primary   EmbeddingProvider
    fallback  EmbeddingProvider
    logger    *zap.Logger
}

func NewProviderManager(primary, fallback EmbeddingProvider, logger *zap.Logger) *ProviderManager {
    return &ProviderManager{
        primary:  primary,
        fallback: fallback,
        logger:   logger,
    }
}

func (m *ProviderManager) Embed(text string) ([]float32, error) {
    // 尝试primary
    if m.primary != nil {
        if err := m.primary.Health(); err == nil {
            return m.primary.Embed(text)
        }
        m.logger.Warn("primary embedding provider unhealthy, trying fallback",
            zap.String("provider", m.primary.Name()),
            zap.Error(err))
    }
    
    // 尝试fallback
    if m.fallback != nil {
        return m.fallback.Embed(text)
    }
    
    return nil, fmt.Errorf("no available embedding provider")
}

func (m *ProviderManager) EmbedBatch(texts []string) ([][]float32, error) {
    // 尝试primary
    if m.primary != nil {
        if err := m.primary.Health(); err == nil {
            return m.primary.EmbedBatch(texts)
        }
        m.logger.Warn("primary embedding provider unhealthy, trying fallback",
            zap.String("provider", m.primary.Name()),
            zap.Error(err))
    }
    
    // 尝试fallback
    if m.fallback != nil {
        return m.fallback.EmbedBatch(texts)
    }
    
    return nil, fmt.Errorf("no available embedding provider")
}

func (m *ProviderManager) Dimension() int {
    if m.primary != nil {
        return m.primary.Dimension()
    }
    if m.fallback != nil {
        return m.fallback.Dimension()
    }
    return 0
}

func (m *ProviderManager) Name() string {
    if m.primary != nil {
        return m.primary.Name()
    }
    if m.fallback != nil {
        return m.fallback.Name()
    }
    return "none"
}
```

---

## 七、API接口定义

### 7.1 REST API（完整版）

```go
package api

import (
    "net/http"
    "time"
    
    "github.com/gin-contrib/cors"
    "github.com/gin-contrib/pprof"
    "github.com/gin-gonic/gin"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "go.uber.org/zap"
)

type Server struct {
    router    *gin.Engine
    indexer   *index.Indexer
    search    *search.HybridSearchEngine
    rag       *rag.RAGBuilder
    config    *config.Config
    logger    *zap.Logger
}

func NewServer(cfg *config.Config, logger *zap.Logger) *Server {
    // 设置Gin模式
    if cfg.Server.Mode == "release" {
        gin.SetMode(gin.ReleaseMode)
    }
    
    router := gin.New()
    
    // 中间件
    router.Use(gin.Recovery())
    router.Use(RequestLogger(logger))
    router.Use(cors.Default())
    
    s := &Server{
        router: router,
        config: cfg,
        logger: logger,
    }
    
    // 注册路由
    s.registerRoutes()
    
    return s
}

func (s *Server) registerRoutes() {
    // 健康检查
    s.router.GET("/health", s.healthCheck)
    
    // API v1
    v1 := s.router.Group("/api/v1")
    {
        // 索引
        v1.POST("/index", s.indexDirectory)
        v1.GET("/index/status", s.indexStatus)
        v1.POST("/index/file", s.indexFile)
        
        // 搜索
        v1.GET("/search", s.search)
        v1.POST("/search", s.searchPost)
        
        // RAG上下文
        v1.GET("/context", s.getContext)
        v1.POST("/context", s.getContextPost)
        
        // 文档
        v1.GET("/documents", s.listDocuments)
        v1.GET("/documents/:id", s.getDocument)
        v1.DELETE("/documents/:id", s.deleteDocument)
        
        // 分块
        v1.GET("/documents/:id/chunks", s.listChunks)
        v1.GET("/chunks/:id", s.getChunk)
        
        // 系统状态
        v1.GET("/status", s.systemStatus)
        
        // 重新生成向量
        v1.POST("/embed", s.regenerateEmbeddings)
    }
    
    // 监控
    s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))
    
    // 性能分析（开发模式）
    if s.config.Server.Mode == "debug" {
        pprof.Register(s.router)
    }
}

// 索引目录
func (s *Server) indexDirectory(c *gin.Context) {
    var req struct {
        Path      string   `json:"path" binding:"required"`
        Recursive bool     `json:"recursive"`
        Exclude   []string `json:"exclude"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    result, err := s.indexer.IndexDirectory(req.Path, req.Recursive)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, result)
}

// 搜索
func (s *Server) search(c *gin.Context) {
    query := c.Query("q")
    if query == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "query is required"})
        return
    }
    
    opts := models.SearchOptions{
        TopK:        parseIntDefault(c.Query("top_k"), 10),
        Mode:        parseStringDefault(c.Query("mode"), "hybrid"),
        Filter:      c.Query("filter"),
        TokenBudget: parseIntDefault(c.Query("token_budget"), 0),
        MinScore:    parseFloatDefault(c.Query("min_score"), 0.0),
    }
    
    results, err := s.search.Search(c.Request.Context(), query, opts)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "results": results,
        "query":   query,
        "mode":    opts.Mode,
        "count":   len(results),
    })
}

// 获取RAG上下文
func (s *Server) getContext(c *gin.Context) {
    query := c.Query("q")
    if query == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "query is required"})
        return
    }
    
    tokenBudget := parseIntDefault(c.Query("token_budget"), 2000)
    
    opts := models.SearchOptions{
        TopK:    50,
        Mode:    "hybrid",
    }
    
    context, err := s.rag.BuildContext(c.Request.Context(), query, tokenBudget, opts)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, context)
}

// 系统状态
func (s *Server) systemStatus(c *gin.Context) {
    status := s.getStatus()
    c.JSON(http.StatusOK, status)
}

// 健康检查
func (s *Server) healthCheck(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "status":  "healthy",
        "version": Version,
    })
}

// 启动服务器
func (s *Server) Start() error {
    addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
    s.logger.Info("server starting", zap.String("addr", addr))
    
    return s.router.Run(addr)
}
```

### 7.2 MCP Server实现

**MCP协议版本**：`2025-06-18` (Stable)

**协议说明**：
- MCP (Model Context Protocol) 是Anthropic推出的Agent通信协议
- 版本采用日期格式（YYYY-MM-DD），表示最后向后不兼容更改的日期
- 2025-06-18是当前稳定版本，支持结构化工具输出、资源链接等特性
- 协议文档：https://modelcontextprotocol.io/specification/2025-06-18

**与主流Agent平台的兼容性**：

| 平台 | MCP支持 | 集成方式 |
|------|---------|----------|
| Claude Desktop | ✅ 原生支持 | 配置文件添加cortex mcp命令 |
| Coze | ✅ 支持 | 通过MCP Server连接 |
| Dify | ✅ 支持 | 通过MCP Server连接 |
| LangChain | ⚠️ 需适配 | 使用LangChain MCP适配器 |
| LlamaIndex | ⚠️ 需适配 | 使用LlamaIndex MCP适配器 |

**Claude Desktop集成示例**：
```json
// ~/Library/Application Support/Claude/claude_desktop_config.json
{
  "mcpServers": {
    "cortex": {
      "command": "/usr/local/bin/cortex",
      "args": ["mcp"],
      "env": {
        "CORTEX_CONFIG": "~/.cortex/config.yaml"
      }
    }
  }
}
```

```go
package mcp

import (
    "context"
    "encoding/json"
    "fmt"
    
    "github.com/modelcontextprotocol/go-sdk/mcp"
    "github.com/moss/cortex/internal/models"
    "github.com/moss/cortex/internal/search"
    "go.uber.org/zap"
)

const (
    // MCP协议版本
    MCPProtocolVersion = "2025-06-18"
    // 服务器名称
    ServerName = "cortex"
)

type MCPServer struct {
    server  *mcp.Server
    search  *search.HybridSearchEngine
    rag     *rag.RAGBuilder
    storage storage.Storage
    logger  *zap.Logger
}

func NewMCPServer(cfg *config.Config, logger *zap.Logger) *MCPServer {
    s := &MCPServer{
        logger: logger,
    }
    
    // 创建MCP服务器
    s.server = mcp.NewServer(&mcp.Implementation{
        Name:    ServerName,
        Version: Version,
    }, &mcp.ServerOptions{
        ProtocolVersion: MCPProtocolVersion,
    })
    
    // 注册工具
    s.registerTools()
    
    return s
}

func (s *MCPServer) registerTools() {
    // cortex_search
    s.server.AddTool(mcp.Tool{
        Name:        "cortex_search",
        Description: "Search the knowledge base and return relevant chunks",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]any{
                "query": map[string]any{
                    "type":        "string",
                    "description": "Search query",
                },
                "top_k": map[string]any{
                    "type":        "integer",
                    "description": "Number of results to return",
                    "default":     10,
                },
                "mode": map[string]any{
                    "type":        "string",
                    "enum":        []string{"hybrid", "vector", "fts"},
                    "description": "Search mode",
                    "default":     "hybrid",
                },
            },
            Required: []string{"query"},
        },
    }, s.handleSearch)
    
    // cortex_context
    s.server.AddTool(mcp.Tool{
        Name:        "cortex_context",
        Description: "Get RAG context with token budget control",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]any{
                "query": map[string]any{
                    "type":        "string",
                    "description": "Search query",
                },
                "token_budget": map[string]any{
                    "type":        "integer",
                    "description": "Maximum tokens for context",
                    "default":     2000,
                },
            },
            Required: []string{"query"},
        },
    }, s.handleContext)
    
    // cortex_get
    s.server.AddTool(mcp.Tool{
        Name:        "cortex_get",
        Description: "Get a specific document or chunk by path or ID",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]any{
                "path": map[string]any{
                    "type":        "string",
                    "description": "Document path",
                },
                "chunk_id": map[string]any{
                    "type":        "string",
                    "description": "Chunk ID",
                },
                "full": map[string]any{
                    "type":        "boolean",
                    "description": "Return full document",
                    "default":     false,
                },
            },
        },
    }, s.handleGet)
    
    // cortex_status
    s.server.AddTool(mcp.Tool{
        Name:        "cortex_status",
        Description: "Get knowledge base status",
        InputSchema: mcp.ToolInputSchema{
            Type:       "object",
            Properties: map[string]any{},
        },
    }, s.handleStatus)
}

// 处理搜索
func (s *MCPServer) handleSearch(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
    query, ok := args["query"].(string)
    if !ok {
        return nil, fmt.Errorf("query is required")
    }
    
    topK := 10
    if v, ok := args["top_k"].(float64); ok {
        topK = int(v)
    }
    
    mode := "hybrid"
    if v, ok := args["mode"].(string); ok {
        mode = v
    }
    
    opts := models.SearchOptions{
        TopK: topK,
        Mode: mode,
    }
    
    results, err := s.search.Search(ctx, query, opts)
    if err != nil {
        return nil, err
    }
    
    // 格式化输出
    output := s.formatSearchResults(results)
    
    return mcp.NewToolResultText(output), nil
}

// 处理上下文请求
func (s *MCPServer) handleContext(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
    query, ok := args["query"].(string)
    if !ok {
        return nil, fmt.Errorf("query is required")
    }
    
    tokenBudget := 2000
    if v, ok := args["token_budget"].(float64); ok {
        tokenBudget = int(v)
    }
    
    opts := models.SearchOptions{
        TopK: 50,
        Mode: "hybrid",
    }
    
    context, err := s.rag.BuildContext(ctx, query, tokenBudget, opts)
    if err != nil {
        return nil, err
    }
    
    // 格式化输出
    output := fmt.Sprintf("Context (%d/%d tokens):\n\n%s\n\nSources:\n%s",
        context.TokenCount, context.TokenBudget, context.Context, s.formatSources(context.Sources))
    
    return mcp.NewToolResultText(output), nil
}

// 格式化搜索结果
func (s *MCPServer) formatSearchResults(results []*models.SearchResult) string {
    var sb strings.Builder
    for i, r := range results {
        sb.WriteString(fmt.Sprintf("[%d] Score: %.3f\n", i+1, r.Score))
        sb.WriteString(fmt.Sprintf("Path: %s\n", r.Chunk.Document.Path))
        if r.Chunk.HeadingPath != "" {
            sb.WriteString(fmt.Sprintf("Section: %s\n", r.Chunk.HeadingPath))
        }
        sb.WriteString(fmt.Sprintf("Lines: %d-%d\n", r.Chunk.LineStart, r.Chunk.LineEnd))
        sb.WriteString(fmt.Sprintf("\n%s\n", truncateText(r.Chunk.ContentRaw, 300)))
        sb.WriteString("\n---\n\n")
    }
    return sb.String()
}

// 运行MCP服务器
func (s *MCPServer) Run() error {
    return s.server.Run()
}
```

---

## 八、工程化规范

### 8.1 日志规范

```go
package logger

import (
    "os"
    
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

func NewLogger(level, mode string) (*zap.Logger, error) {
    // 日志级别
    var zapLevel zapcore.Level
    switch level {
    case "debug":
        zapLevel = zapcore.DebugLevel
    case "info":
        zapLevel = zapcore.InfoLevel
    case "warn":
        zapLevel = zapcore.WarnLevel
    case "error":
        zapLevel = zapcore.ErrorLevel
    default:
        zapLevel = zapcore.InfoLevel
    }
    
    // 编码器配置
    encoderConfig := zapcore.EncoderConfig{
        TimeKey:        "time",
        LevelKey:       "level",
        NameKey:        "logger",
        CallerKey:      "caller",
        FunctionKey:    zapcore.OmitKey,
        MessageKey:     "msg",
        StacktraceKey:  "stacktrace",
        LineEnding:     zapcore.DefaultLineEnding,
        EncodeLevel:    zapcore.LowercaseLevelEncoder,
        EncodeTime:     zapcore.ISO8601TimeEncoder,
        EncodeDuration: zapcore.SecondsDurationEncoder,
        EncodeCaller:   zapcore.ShortCallerEncoder,
    }
    
    // 开发模式使用console，生产模式使用json
    var encoder zapcore.Encoder
    if mode == "debug" {
        encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
        encoder = zapcore.NewConsoleEncoder(encoderConfig)
    } else {
        encoder = zapcore.NewJSONEncoder(encoderConfig)
    }
    
    // 核心配置
    core := zapcore.NewCore(
        encoder,
        zapcore.AddSync(os.Stdout),
        zapLevel,
    )
    
    // 创建logger
    logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
    
    return logger, nil
}

// 请求日志中间件
func RequestLogger(logger *zap.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        path := c.Request.URL.Path
        query := c.Request.URL.RawQuery
        
        c.Next()
        
        // 记录请求
        logger.Info("request",
            zap.Int("status", c.Writer.Status()),
            zap.String("method", c.Request.Method),
            zap.String("path", path),
            zap.String("query", query),
            zap.String("ip", c.ClientIP()),
            zap.Duration("latency", time.Since(start)),
            zap.String("user-agent", c.Request.UserAgent()),
        )
    }
}
```

### 8.2 监控指标

```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // 索引指标
    DocumentsIndexed = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "cortex_documents_indexed_total",
        Help: "Total number of documents indexed",
    }, []string{"status"}) // success, failed, skipped
    
    IndexDuration = promauto.NewHistogram(prometheus.HistogramOpts{
        Name:    "cortex_index_duration_seconds",
        Help:    "Time spent indexing documents",
        Buckets: prometheus.DefBuckets,
    })
    
    // 搜索指标
    SearchRequests = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "cortex_search_requests_total",
        Help: "Total number of search requests",
    }, []string{"mode", "status"}) // hybrid/vector/fts, success/error
    
    SearchDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "cortex_search_duration_seconds",
        Help:    "Time spent on search",
        Buckets: prometheus.DefBuckets,
    }, []string{"mode"})
    
    SearchResults = promauto.NewHistogram(prometheus.HistogramOpts{
        Name:    "cortex_search_results_count",
        Help:    "Number of results returned",
        Buckets: []float64{1, 5, 10, 20, 50, 100},
    })
    
    // 缓存指标
    CacheHits = promauto.NewCounter(prometheus.CounterOpts{
        Name: "cortex_cache_hits_total",
        Help: "Total number of cache hits",
    })
    
    CacheMisses = promauto.NewCounter(prometheus.CounterOpts{
        Name: "cortex_cache_misses_total",
        Help: "Total number of cache misses",
    })
    
    // 存储指标
    StorageSize = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "cortex_storage_size_bytes",
        Help: "Size of the storage database in bytes",
    })
    
    ChunkCount = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "cortex_chunks_total",
        Help: "Total number of chunks in storage",
    })
    
    DocumentCount = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "cortex_documents_total",
        Help: "Total number of documents in storage",
    })
    
    // Embedding指标
    EmbeddingRequests = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "cortex_embedding_requests_total",
        Help: "Total number of embedding requests",
    }, []string{"provider", "status"})
    
    EmbeddingDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "cortex_embedding_duration_seconds",
        Help:    "Time spent on embedding generation",
        Buckets: prometheus.DefBuckets,
    }, []string{"provider"})
)
```

### 8.3 错误处理规范

```go
package errors

import (
    "fmt"
    "net/http"
)

// 自定义错误类型
type CortexError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Status  int    `json:"-"`
    Cause   error  `json:"-"`
}

func (e *CortexError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *CortexError) Unwrap() error {
    return e.Cause
}

// 预定义错误
var (
    ErrInvalidInput = &CortexError{
        Code:    "INVALID_INPUT",
        Message: "Invalid input parameter",
        Status:  http.StatusBadRequest,
    }
    
    ErrDocumentNotFound = &CortexError{
        Code:    "DOCUMENT_NOT_FOUND",
        Message: "Document not found",
        Status:  http.StatusNotFound,
    }
    
    ErrIndexFailed = &CortexError{
        Code:    "INDEX_FAILED",
        Message: "Failed to index document",
        Status:  http.StatusInternalServerError,
    }
    
    ErrSearchFailed = &CortexError{
        Code:    "SEARCH_FAILED",
        Message: "Search operation failed",
        Status:  http.StatusInternalServerError,
    }
    
    ErrEmbeddingFailed = &CortexError{
        Code:    "EMBEDDING_FAILED",
        Message: "Failed to generate embedding",
        Status:  http.StatusInternalServerError,
    }
    
    ErrStorageFailed = &CortexError{
        Code:    "STORAGE_FAILED",
        Message: "Storage operation failed",
        Status:  http.StatusInternalServerError,
    }
    
    ErrUnsupportedFileType = &CortexError{
        Code:    "UNSUPPORTED_FILE_TYPE",
        Message: "Unsupported file type",
        Status:  http.StatusBadRequest,
    }
)

// 包装错误
func Wrap(err *CortexError, cause error, message string) *CortexError {
    return &CortexError{
        Code:    err.Code,
        Message: message,
        Status:  err.Status,
        Cause:   cause,
    }
}

// 错误处理中间件
func ErrorHandler() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()
        
        // 检查是否有错误
        if len(c.Errors) > 0 {
            err := c.Errors.Last().Err
            
            var cortexErr *CortexError
            if errors.As(err, &cortexErr) {
                c.JSON(cortexErr.Status, gin.H{
                    "error": cortexErr,
                })
                return
            }
            
            // 未知错误
            c.JSON(http.StatusInternalServerError, gin.H{
                "error": &CortexError{
                    Code:    "INTERNAL_ERROR",
                    Message: err.Error(),
                    Status:  http.StatusInternalServerError,
                },
            })
        }
    }
}
```

---

## 九、测试策略

### 9.1 单元测试

```go
package chunker_test

import (
    "testing"
    
    "github.com/moss/cortex/internal/chunker"
    "github.com/stretchr/testify/assert"
)

func TestMarkdownChunker_Chunk(t *testing.T) {
    config := chunker.ChunkConfig{
        MaxTokens:         512,
        OverlapTokens:     64,
        MinChars:          50,
        IncludeBreadcrumb: true,
        Tokenizer:         "cl100k_base",
    }
    
    ch, err := chunker.NewMarkdownChunker(config)
    assert.NoError(t, err)
    
    tests := []struct {
        name     string
        content  string
        expected int
    }{
        {
            name: "simple paragraph",
            content: `# Title

This is a simple paragraph with enough characters to pass the minimum threshold.`,
            expected: 1,
        },
        {
            name: "multiple sections",
            content: `# Section 1

Content for section 1 with enough characters to be included.

## Subsection 1.1

More content here that should create a separate chunk.

# Section 2

Another section with its own content.`,
            expected: 3,
        },
        {
            name: "code block",
            content: `# Code Example

` + "```go" + `
func main() {
    fmt.Println("Hello, World!")
}
` + "```",
            expected: 1,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            chunks, err := ch.Chunk(tt.content, "test.md")
            assert.NoError(t, err)
            assert.GreaterOrEqual(t, len(chunks), tt.expected)
        })
    }
}

func TestMarkdownChunker_TokenCount(t *testing.T) {
    config := chunker.ChunkConfig{
        MaxTokens:         100,
        MinChars:          10,
        IncludeBreadcrumb: false,
        Tokenizer:         "cl100k_base",
    }
    
    ch, err := chunker.NewMarkdownChunker(config)
    assert.NoError(t, err)
    
    content := "This is a test paragraph."
    chunks, err := ch.Chunk(content, "test.md")
    assert.NoError(t, err)
    assert.Len(t, chunks, 1)
    assert.LessOrEqual(t, chunks[0].TokenCount, config.MaxTokens)
}
```

### 9.2 集成测试

```go
package integration_test

import (
    "context"
    "os"
    "path/filepath"
    "testing"
    
    "github.com/moss/cortex/internal/index"
    "github.com/moss/cortex/internal/search"
    "github.com/moss/cortex/internal/storage"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/suite"
)

type IntegrationTestSuite struct {
    suite.Suite
    storage   storage.Storage
    indexer   *index.Indexer
    search    *search.HybridSearchEngine
    tempDir   string
}

func (s *IntegrationTestSuite) SetupSuite() {
    // 创建临时目录
    s.tempDir = s.T().TempDir()
    
    // 初始化存储
    dbPath := filepath.Join(s.tempDir, "test.db")
    s.storage = storage.NewSQLiteStorage(dbPath)
    
    // 初始化索引器
    s.indexer = index.NewIndexer(s.storage, /* ... */)
    
    // 初始化搜索引擎
    s.search = search.NewHybridSearchEngine(s.storage, /* ... */)
}

func (s *IntegrationTestSuite) TearDownSuite() {
    s.storage.Close()
}

func (s *IntegrationTestSuite) TestIndexAndSearch() {
    // 创建测试文件
    testFile := filepath.Join(s.tempDir, "test.md")
    content := `# Test Document

This is a test document about machine learning and AI.

## Introduction

Machine learning is a subset of artificial intelligence.

## Applications

AI has many applications in healthcare, finance, and more.
`
    err := os.WriteFile(testFile, []byte(content), 0644)
    assert.NoError(s.T(), err)
    
    // 索引
    result, err := s.indexer.IndexDirectory(s.tempDir, true)
    assert.NoError(s.T(), err)
    assert.Equal(s.T(), 1, result.Indexed)
    
    // 搜索
    results, err := s.search.Search(context.Background(), "machine learning", models.SearchOptions{
        TopK: 10,
        Mode: "hybrid",
    })
    assert.NoError(s.T(), err)
    assert.Greater(s.T(), len(results), 0)
    
    // 验证结果
    assert.Contains(s.T(), results[0].Chunk.ContentRaw, "machine learning")
}

func TestIntegrationSuite(t *testing.T) {
    suite.Run(t, new(IntegrationTestSuite))
}
```

### 9.3 性能测试

```go
package benchmark_test

import (
    "testing"
    
    "github.com/moss/cortex/internal/chunker"
)

func BenchmarkMarkdownChunker_Chunk(b *testing.B) {
    config := chunker.ChunkConfig{
        MaxTokens:         512,
        OverlapTokens:     64,
        MinChars:          50,
        IncludeBreadcrumb: true,
        Tokenizer:         "cl100k_base",
    }
    
    ch, _ := chunker.NewMarkdownChunker(config)
    
    // 生成大型Markdown文档
    content := generateLargeMarkdown(10000) // 10000行
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        ch.Chunk(content, "test.md")
    }
}

func BenchmarkHybridSearch_Search(b *testing.B) {
    // 初始化搜索引擎（预加载数据）
    search := setupSearchEngine(b)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        search.Search(context.Background(), "test query", models.SearchOptions{
            TopK: 10,
            Mode: "hybrid",
        })
    }
}
```

---

## 十、部署与运维

### 10.1 配置文件（完整版）

```yaml
# ~/.cortex/config.yaml

# 服务配置
server:
  host: "127.0.0.1"
  port: 7832
  mode: "release"         # debug/release
  read_timeout: 30s
  write_timeout: 30s

# 存储配置
storage:
  type: "sqlite"
  path: "~/.cortex/cortex.db"
  pragmas:
    journal_mode: "WAL"
    synchronous: "NORMAL"
    cache_size: 10000
    busy_timeout: 5000

# 知识库集合
collections:
  - path: "~/notes"
    name: "notes"
    recursive: true
    exclude:
      - ".git"
      - "node_modules"
      - "*.tmp"
  - path: "~/work/docs"
    name: "docs"
    recursive: true

# Embedding配置
embedding:
  provider: "ollama"       # ollama/openai/onnx
  
  # Ollama配置
  ollama:
    base_url: "http://localhost:11434"
    model: "nomic-embed-text"
    dimension: 768
    timeout: 30s
  
  # OpenAI配置（fallback）
  openai:
    api_key: "${OPENAI_API_KEY}"  # 从环境变量读取
    model: "text-embedding-3-small"
    dimension: 1536
    timeout: 30s
  
  # ONNX配置
  onnx:
    model_path: "~/.cortex/models/nomic-embed-text.onnx"
    dimension: 768

# 分块配置
chunking:
  max_tokens: 512
  overlap_tokens: 64
  min_chars: 50
  include_breadcrumb: true
  tokenizer: "cl100k_base"  # tiktoken模型

# 搜索配置
search:
  default_mode: "hybrid"
  default_top_k: 10
  cache_ttl: 300s
  rrf_k: 60

# 索引配置
index:
  batch_size: 100
  concurrency: 4
  watch_enabled: true
  watch_debounce: 1s

# 日志配置
logging:
  level: "info"           # debug/info/warn/error
  format: "json"          # json/console
  output: "stdout"        # stdout/file
  file_path: "~/.cortex/cortex.log"

# 监控配置
monitoring:
  enabled: true
  metrics_path: "/metrics"
  prometheus:
    enabled: true
    port: 9090
```

### 10.2 Docker部署

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 安装依赖
RUN apk add --no-cache git gcc musl-dev

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o cortex ./cmd/cortex

# 运行镜像
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# 复制二进制
COPY --from=builder /app/cortex .
COPY --from=builder /app/config.example.yaml ./config.yaml

# 创建数据目录
RUN mkdir -p /data

# 暴露端口
EXPOSE 7832 9090

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s \
  CMD wget -q --spider http://localhost:7832/health || exit 1

# 运行
ENTRYPOINT ["./cortex"]
CMD ["serve"]
```

```yaml
# docker-compose.yml
version: '3.8'

services:
  cortex:
    build: .
    container_name: cortex
    ports:
      - "7832:7832"
      - "9090:9090"
    volumes:
      - ./config.yaml:/app/config.yaml
      - cortex-data:/data
      - ~/notes:/notes:ro
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    restart: unless-stopped
    depends_on:
      - ollama
  
  ollama:
    image: ollama/ollama:latest
    container_name: ollama
    ports:
      - "11434:11434"
    volumes:
      - ollama-data:/root/.ollama
    restart: unless-stopped

volumes:
  cortex-data:
  ollama-data:
```

### 10.3 一键安装脚本

```bash
#!/bin/bash
# install.sh

set -e

VERSION="v1.0.0"
BINARY_NAME="cortex"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="$HOME/.cortex"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检测系统
detect_os() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case $ARCH in
        x86_64) ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *) log_error "Unsupported architecture: $ARCH"; exit 1 ;;
    esac
    
    log_info "Detected OS: $OS, Architecture: $ARCH"
}

# 下载二进制
download_binary() {
    URL="https://github.com/your-org/cortex/releases/download/${VERSION}/cortex-${OS}-${ARCH}"
    
    log_info "Downloading Cortex ${VERSION}..."
    
    if command -v wget &> /dev/null; then
        wget -q -O ${BINARY_NAME} ${URL}
    elif command -v curl &> /dev/null; then
        curl -sL -o ${BINARY_NAME} ${URL}
    else
        log_error "wget or curl required"
        exit 1
    fi
    
    chmod +x ${BINARY_NAME}
}

# 安装
install() {
    log_info "Installing Cortex..."
    
    # 创建配置目录
    mkdir -p ${CONFIG_DIR}
    
    # 移动二进制
    sudo mv ${BINARY_NAME} ${INSTALL_DIR}/${BINARY_NAME}
    
    # 生成默认配置
    if [ ! -f "${CONFIG_DIR}/config.yaml" ]; then
        cat > ${CONFIG_DIR}/config.yaml << EOF
# Cortex Configuration
# See: https://github.com/your-org/cortex#configuration

server:
  host: "127.0.0.1"
  port: 7832

collections:
  - path: "~/notes"
    name: "notes"

embedding:
  provider: "ollama"
  ollama:
    base_url: "http://localhost:11434"
    model: "nomic-embed-text"
EOF
        log_info "Created default config at ${CONFIG_DIR}/config.yaml"
    fi
    
    log_info "Cortex installed to ${INSTALL_DIR}/${BINARY_NAME}"
}

# 检查Ollama
check_ollama() {
    if ! command -v ollama &> /dev/null; then
        log_warn "Ollama not found. Install it from https://ollama.ai"
        log_warn "Or use OpenAI as fallback by setting OPENAI_API_KEY"
    else
        log_info "Ollama found"
        
        # 检查模型
        if ! ollama list | grep -q "nomic-embed-text"; then
            log_info "Pulling nomic-embed-text model..."
            ollama pull nomic-embed-text
        fi
    fi
}

# 验证安装
verify() {
    log_info "Verifying installation..."
    
    if command -v cortex &> /dev/null; then
        VERSION=$(cortex version)
        log_info "Cortex ${VERSION} installed successfully!"
        log_info ""
        log_info "Quick start:"
        log_info "  cortex index ~/notes          # Index your notes"
        log_info "  cortex search \"machine learning\"  # Search"
        log_info "  cortex serve                  # Start API server"
    else
        log_error "Installation verification failed"
        exit 1
    fi
}

# 主流程
main() {
    log_info "Installing Cortex ${VERSION}..."
    
    detect_os
    download_binary
    install
    check_ollama
    verify
}

main "$@"
```

---

## 十一、项目结构（完整版）

```
cortex/
├── cmd/
│   ├── cortex/
│   │   └── main.go           # 入口
│   ├── root.go               # 根命令
│   ├── index.go              # index命令
│   ├── search.go             # search命令
│   ├── context.go            # context命令
│   ├── serve.go              # serve命令
│   ├── mcp.go                # mcp命令
│   ├── status.go             # status命令
│   └── version.go            # version命令
├── internal/
│   ├── chunker/              # 分块算法
│   │   ├── chunker.go        # 接口定义
│   │   ├── markdown.go       # Markdown分块
│   │   ├── pdf.go            # PDF分块
│   │   ├── docx.go           # Word分块
│   │   └── chunker_test.go
│   ├── embedding/            # Embedding接口
│   │   ├── provider.go       # 接口定义
│   │   ├── manager.go        # 提供商管理器
│   │   ├── ollama.go         # Ollama实现
│   │   ├── openai.go         # OpenAI实现
│   │   ├── onnx.go           # ONNX实现
│   │   └── embedding_test.go
│   ├── search/               # 搜索算法
│   │   ├── engine.go         # 搜索引擎
│   │   ├── vector.go         # 向量搜索
│   │   ├── fts.go            # 全文搜索
│   │   ├── rrf.go            # RRF融合
│   │   └── search_test.go
│   ├── storage/              # 存储层
│   │   ├── storage.go        # 接口定义
│   │   ├── sqlite.go         # SQLite实现
│   │   ├── schema.sql        # 数据库Schema
│   │   └── storage_test.go
│   ├── index/                # 索引逻辑
│   │   ├── indexer.go        # 索引器
│   │   ├── watcher.go        # 文件监听
│   │   └── indexer_test.go
│   ├── rag/                  # RAG组装
│   │   ├── builder.go        # 上下文构建
│   │   └── rag_test.go
│   ├── api/                  # API层
│   │   ├── server.go         # REST服务器
│   │   ├── handlers.go       # 请求处理
│   │   ├── middleware.go     # 中间件
│   │   └── mcp.go            # MCP服务器
│   ├── models/               # 数据模型
│   │   ├── document.go
│   │   ├── chunk.go
│   │   └── search.go
│   ├── errors/               # 错误定义
│   │   └── errors.go
│   ├── logger/               # 日志
│   │   └── logger.go
│   └── metrics/              # 监控指标
│       └── metrics.go
├── pkg/
│   ├── config/               # 配置管理
│   │   ├── config.go
│   │   └── config_test.go
│   └── utils/                # 工具函数
│       ├── hash.go
│       ├── token.go
│       └── truncate.go
├── scripts/
│   ├── install.sh            # 一键安装脚本
│   └── release.sh            # 发布脚本
├── configs/
│   └── config.example.yaml   # 配置示例
├── docs/
│   ├── getting-started.md
│   ├── configuration.md
│   ├── api.md
│   └── mcp.md
├── test/
│   ├── integration/          # 集成测试
│   └── benchmark/            # 性能测试
├── .github/
│   └── workflows/
│       ├── test.yml          # 测试工作流
│       ├── release.yml       # 发布工作流
│       └── docker.yml        # Docker构建
├── go.mod
├── go.sum
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── README.md
```

---

## 十二、实现路线图

### Phase 1: MVP核心（2周）

**Week 1: 基础架构**
- [ ] 项目初始化（go mod init）
- [ ] SQLite存储层实现
- [ ] 配置管理（viper）
- [ ] 日志系统（zap）
- [ ] 错误处理框架
- [ ] 监控指标（prometheus）

**Week 2: 核心功能**
- [ ] Markdown层级分块算法
- [ ] tiktoken集成
- [ ] Ollama Embedding集成
- [ ] 增量索引
- [ ] 混合搜索（向量+BM25+RRF）
- [ ] CLI命令（index/search/status）
- [ ] 单元测试

### Phase 2: API与服务（1周）

**Week 3: 接口层**
- [ ] REST API服务器
- [ ] MCP Server实现
- [ ] RAG上下文组装
- [ ] OpenAI Embedding fallback
- [ ] 搜索缓存
- [ ] 集成测试

### Phase 3: 优化与扩展（1周）

**Week 4: 性能与体验**
- [ ] ONNX本地Embedding
- [ ] 文件监听自动索引
- [ ] PDF解析支持
- [ ] 批量嵌入优化
- [ ] 性能测试与优化
- [ ] Docker镜像
- [ ] 一键安装脚本

### Phase 4: 生态与增强（后续迭代）

**v1.1: 文档格式扩展**
- [ ] Word文档解析
- [ ] HTML解析
- [ ] 图片OCR（可选）

**v1.2: 可视化界面**
- [ ] 轻量Web UI
- [ ] 索引状态可视化
- [ ] 分块预览
- [ ] 搜索结果对比

**v1.3: 生态对接**
- [ ] LangChain适配器
- [ ] LlamaIndex适配器
- [ ] Python SDK
- [ ] TypeScript SDK

**v2.0: 团队版**
- [ ] 多租户支持
- [ ] 权限管理
- [ ] 云端同步（可选）
- [ ] 协作功能

---

## 十三、风险与应对

| 风险类型 | 风险描述 | 概率 | 影响 | 应对措施 |
|----------|----------|------|------|----------|
| **技术风险** | SQLite高并发性能不足 | 中 | 高 | 启用WAL模式；增加缓存层；考虑PostgreSQL扩展 |
| **技术风险** | Embedding服务不稳定 | 高 | 中 | 多提供商fallback；本地ONNX兜底；健康检查 |
| **技术风险** | Token计算精度问题 | 低 | 中 | 使用tiktoken专业库；定期校准 |
| **产品风险** | 用户部署门槛高 | 中 | 高 | 提供预编译二进制；一键安装脚本；云端fallback |
| **产品风险** | 文档格式支持不足 | 中 | 中 | 优先支持PDF；提供格式转换工具建议 |
| **产品风险** | 与竞品差异化不足 | 中 | 中 | 强化"5分钟上手"；打磨CLI体验；完善文档 |
| **运维风险** | 监控告警缺失 | 低 | 中 | 集成prometheus；提供健康检查接口 |
| **运维风险** | 数据迁移困难 | 低 | 低 | SQLite单文件易于备份；提供导出工具 |

---

## 十四、用户体验设计

### 14.1 索引进度反馈

**CLI进度显示**：

```go
package cli

import (
    "fmt"
    "time"
    
    "github.com/schollz/progressbar/v3"
)

// 索引进度条
func showIndexProgress(result *IndexResult) {
    bar := progressbar.NewOptions(result.Total,
        progressbar.OptionSetDescription("Indexing..."),
        progressbar.OptionSetWriter(os.Stderr),
        progressbar.OptionShowCount(),
        progressbar.OptionShowIts(),
        progressbar.OptionSetItsString("files"),
        progressbar.OptionOnCompletion(func() {
            fmt.Fprint(os.Stderr, "\n")
        }),
    )
    
    for i := 0; i < result.Indexed; i++ {
        bar.Add(1)
        time.Sleep(10 * time.Millisecond)
    }
    
    // 显示结果
    fmt.Printf("\n✓ Indexed: %d | Skipped: %d | Failed: %d | Duration: %dms\n",
        result.Indexed, result.Skipped, result.Failed, result.Duration)
    
    if len(result.Errors) > 0 {
        fmt.Printf("\nErrors:\n")
        for _, err := range result.Errors {
            fmt.Printf("  - %s\n", err)
        }
    }
}
```

**实时进度流（REST API）**：

```go
// Server-Sent Events for real-time progress
func (s *Server) indexProgressSSE(c *gin.Context) {
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")
    
    // 创建进度通道
    progressCh := make(chan ProgressUpdate, 100)
    
    // 启动索引任务
    go func() {
        s.indexer.IndexDirectoryWithProgress(c.Query("path"), progressCh)
    }()
    
    // 发送进度更新
    for update := range progressCh {
        data, _ := json.Marshal(update)
        c.SSEvent("progress", string(data))
        c.Writer.Flush()
    }
}

type ProgressUpdate struct {
    Total     int    `json:"total"`
    Processed int    `json:"processed"`
    Current   string `json:"current"`  // 当前文件
    Status    string `json:"status"`   // processing/success/error
    Error     string `json:"error,omitempty"`
}
```

**进度示例输出**：

```
Indexing ~/notes...
[████████████████████░░░░░░░░] 67% | 134/200 files | 2.5s

✓ Indexed: 180 | Skipped: 15 | Failed: 5 | Duration: 3245ms

Errors:
  - ~/notes/damaged.pdf: failed to extract text
  - ~/notes/encrypted.docx: password required
```

### 14.2 错误提示设计

**错误分级**：

| 级别 | 含义 | 示例 | 用户提示 |
|------|------|------|----------|
| Error | 阻塞操作 | Embedding服务不可用 | 明确说明问题 + 解决方案 |
| Warn | 非阻塞但有影响 | PDF无法提取文本 | 提示问题，继续执行 |
| Info | 信息提示 | 文件未修改，跳过 | 轻量提示 |

**错误提示模板**：

```go
package errors

// 用户友好的错误提示
var UserFriendlyErrors = map[string]struct {
    Title       string
    Description string
    Solution    string
}{
    "EMBEDDING_FAILED": {
        Title:       "Embedding service unavailable",
        Description: "Cannot connect to Ollama at http://localhost:11434",
        Solution:    "Run 'ollama serve' or set OPENAI_API_KEY for fallback",
    },
    "PDF_EXTRACTION_FAILED": {
        Title:       "PDF text extraction failed",
        Description: "This PDF may be scanned or encrypted",
        Solution:    "For scanned PDFs, try OCR tools like Tesseract",
    },
    "DOCUMENT_NOT_FOUND": {
        Title:       "Document not found",
        Description: "The specified path does not exist",
        Solution:    "Check the path and try again",
    },
}

func FormatUserError(err *CortexError) string {
    if info, ok := UserFriendlyErrors[err.Code]; ok {
        return fmt.Sprintf(`
❌ %s

   %s

💡 Solution: %s
`, info.Title, info.Description, info.Solution)
    }
    
    // 默认格式
    return fmt.Sprintf("❌ Error: %s\n\n   %s", err.Code, err.Message)
}
```

**CLI错误输出示例**：

```
❌ Embedding service unavailable

   Cannot connect to Ollama at http://localhost:11434

💡 Solution: Run 'ollama serve' or set OPENAI_API_KEY for fallback
```

### 14.3 交互式确认

**危险操作确认**：

```go
func confirmAction(message string) bool {
    fmt.Printf("%s [y/N]: ", message)
    
    var response string
    fmt.Scanln(&response)
    
    return strings.ToLower(response) == "y"
}

// 使用示例
if len(documents) > 100 {
    if !confirmAction(fmt.Sprintf("This will delete %d documents. Continue?", len(documents))) {
        fmt.Println("Cancelled")
        return
    }
}
```

**输出示例**：

```
This will delete 150 documents. Continue? [y/N]: n
Cancelled
```

### 14.4 状态命令

**cortex status 命令**：

```bash
$ cortex status

Cortex Status
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Version:        v1.0.0
Uptime:         2h 34m
Storage:        ~/.cortex/cortex.db (156MB)

Documents:      1,234
Chunks:         45,678
Storage Size:   156MB

Embedding:      ollama:nomic-embed-text
Last Indexed:   2026-04-17 02:15:30

Collections:
  ✓ notes       (856 docs, 32K chunks)
  ✓ docs        (378 docs, 13K chunks)

Health:
  ✓ Database:   healthy
  ✓ Embedding:  healthy (latency: 45ms)
  ⚠  Memory:    180MB/200MB (90%)

Recent Errors:
  - [2h ago] PDF extraction failed: scanned.pdf
```

### 14.5 日志级别控制

```yaml
# config.yaml
logging:
  level: "info"           # debug/info/warn/error
  format: "console"       # console/json
  file: "~/.cortex/cortex.log"
  max_size: 10MB
  max_backups: 5
  max_age: 30             # days
```

**CLI日志控制**：

```bash
# 详细日志
cortex index ~/notes --verbose

# 静默模式
cortex index ~/notes --quiet

# 调试模式
cortex index ~/notes --debug
```

### 14.6 配置检查命令

```bash
$ cortex config check

Configuration Check
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✓ Config file: ~/.cortex/config.yaml
✓ Database: ~/.cortex/cortex.db (writable)
✓ Ollama: http://localhost:11434 (healthy)
✓ Model: nomic-embed-text (downloaded)
✓ Collections:
    ~/notes: exists (writable)
    ~/work/docs: exists (writable)

All checks passed! ✓
```

```bash
# 配置问题示例
$ cortex config check

Configuration Check
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✓ Config file: ~/.cortex/config.yaml
✓ Database: ~/.cortex/cortex.db (writable)
✗ Ollama: http://localhost:11434 (connection refused)
  └─ Fallback: OpenAI API key not set
✗ Collection: ~/notes (directory not found)

⚠ 2 issues found. Run 'cortex config fix' for suggestions.
```

---

## 附录

### A. API文档（Swagger格式）

```yaml
openapi: 3.0.0
info:
  title: Cortex API
  version: 1.0.0
  description: Agent Knowledge Base API

servers:
  - url: http://localhost:7832/api/v1

paths:
  /index:
    post:
      summary: Index a directory
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required:
                - path
              properties:
                path:
                  type: string
                recursive:
                  type: boolean
                  default: true
      responses:
        '200':
          description: Index result
          
  /search:
    get:
      summary: Search the knowledge base
      parameters:
        - name: q
          in: query
          required: true
          schema:
            type: string
        - name: top_k
          in: query
          schema:
            type: integer
            default: 10
        - name: mode
          in: query
          schema:
            type: string
            enum: [hybrid, vector, fts]
            default: hybrid
      responses:
        '200':
          description: Search results
```

### B. 性能基准

| 指标 | 目标值 | 测试条件 |
|------|--------|----------|
| 索引速度 | >100 docs/min | 单核，100KB/文档 |
| 搜索延迟 | <100ms | 1000文档，Top10结果 |
| 内存占用 | <200MB | 10000文档 |
| 启动时间 | <1s | 冷启动 |
| 数据库大小 | ~3x文档大小 | 含向量索引 |

### C. 贡献指南

```markdown
# Contributing to Cortex

## Development Setup

1. Clone the repo
2. Install Go 1.21+
3. Run `go mod download`
4. Run tests: `make test`

## Code Style

- Use `gofmt` for formatting
- Run `golangci-lint` before submitting
- Write tests for new features

## Pull Request Process

1. Create a feature branch
2. Make your changes
3. Run tests and linters
4. Submit PR with description
```

---

**文档版本**：v3.0  
**最后更新**：2026-04-17  
**维护者**：Cortex Team

**下一步**：按Phase 1开始开发，遇到具体问题参考本文档相应章节。
