# Cortex - Agent Knowledge Base

[English](#english) | [中文](#中文)

---

## English

Cortex is a local knowledge base system designed for AI Agents, powered by SQLite vector storage and hybrid search (vector + full-text search).

### Features

- **Hybrid Search**: Combines vector similarity search ( cosine ) with FTS5 BM25 for accurate results
- **Multiple File Formats**: Supports Markdown, PDF, and Word (.docx) documents
- **MCP Protocol**: Built-in Model Context Protocol server for AI Agent integration
- **REST API**: Full-featured HTTP API for search and RAG context generation
- **Real-time Indexing**: File system watcher for automatic document indexing
- **Configurable**: YAML configuration with environment variable overrides
- **Metrics**: Prometheus-compatible metrics endpoint
- **Graceful Shutdown**: Proper signal handling for production deployments

### Architecture

```
┌─────────────────────────────────────────────────────┐
│                     Cortex CLI                       │
├─────────────────────────────────────────────────────┤
│  index  │  search  │  context  │  serve  │  watch  │
├─────────────────────────────────────────────────────┤
│              Hybrid Search Engine                     │
│    Vector Search (Cosine)  │  FTS5 (BM25)           │
├─────────────────────────────────────────────────────┤
│              SQLite Storage Layer                    │
│   Documents │ Chunks │ Vectors │ FTS Index         │
└─────────────────────────────────────────────────────┘
```

### Installation

```bash
# Clone the repository
git clone https://github.com/lh123aa/cortex.git
cd cortex

# Build
go build -o cortex ./cmd/cortex

# Or install globally
go install ./cmd/cortex
```

### Quick Start

```bash
# Index a directory
./cortex index /path/to/docs

# Search
./cortex search "your query"

# Get RAG context
./cortex context "your query"

# Start REST API server
./cortex serve

# Start MCP server (for AI Agent integration)
./cortex mcp
```

### Configuration

Cortex uses a YAML configuration file with environment variable overrides.

Default config locations (in order):
1. `./config.yaml`
2. `~/.cortex/config.yaml`

#### Configuration File Example

```yaml
cortex:
  db_path: ~/.cortex/cortex.db
  log_level: info

embedding:
  provider: ollama  # or "onnx"
  ollama:
    base_url: http://localhost:11434
    model: nomic-embed-text
  onnx:
    base_url: http://localhost:8080
    model: embedder
    dim: 768

index:
  max_tokens: 512
  overlap_tokens: 64
  min_chars: 50
  workers: 4

search:
  cache_ttl: 5m
  default_top_k: 10

backup:
  enabled: true
  dir: ~/.cortex/backups
  max_backups: 10
  auto_backup: false
```

#### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `CORTEX_DB_PATH` | Database file path | `cortex.db` |
| `CORTEX_EMBEDDING_PROVIDER` | Provider: `ollama` or `onnx` | `ollama` |
| `OLLAMA_HOST` | Ollama server URL | `http://localhost:11434` |
| `OLLAMA_MODEL` | Ollama model name | `nomic-embed-text` |
| `ONNX_BASE_URL` | ONNX server URL | `http://localhost:8080` |
| `ONNX_MODEL` | ONNX model name | `embedder` |

### CLI Commands

| Command | Description |
|---------|-------------|
| `cortex index <path>` | Index a directory |
| `cortex search <query>` | Search the knowledge base |
| `cortex context <query>` | Get RAG assembled context |
| `cortex watch <path>` | Watch directory for changes |
| `cortex serve [--addr :8080]` | Start REST API server |
| `cortex mcp` | Start MCP server (stdio) |
| `cortex backup` | Create database backup |
| `cortex version` | Print version |

### REST API

Start the server: `cortex serve`

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/v1/search?q=<query>` | GET | Hybrid search |
| `/v1/context?q=<query>&budget=<tokens>` | GET | RAG context |
| `/v1/docs` | GET | List documents |
| `/v1/docs/:id` | GET | Get document |
| `/v1/stats` | GET | Statistics |

### MCP Integration

Cortex provides a Model Context Protocol server for AI Agent integration:

```bash
# Start MCP server
cortex mcp
```

Available tools:
- `cortex_search`: Semantic search with hybrid vector + FTS
- `cortex_context`: RAG context assembly within token budget

### API Response Examples

#### Search Response

```json
{
  "query": "golang concurrency",
  "total": 5,
  "results": [
    {
      "rank": 1,
      "score": 0.95,
      "path": "/docs/intro.md",
      "section": "基础知识 > 并发",
      "content_raw": "Go语言使用Goroutine实现并发...",
      "token_count": 128
    }
  ]
}
```

#### Context Response

```json
{
  "query": "golang concurrency",
  "context": "Go语言使用Goroutine实现并发...\n\n---\n\n通道用于Goroutine通信...",
  "token_count": 850,
  "token_budget": 1500,
  "truncated": false
}
```

### Development

```bash
# Run tests
go test ./...

# Build
go build -o cortex ./cmd/cortex
```

### License

MIT

---

## 中文

Cortex 是一个面向 AI Agent 的本地知识库系统，基于 SQLite 向量存储和混合搜索（向量 + 全文搜索）。

### 功能特性

- **混合搜索**: 结合向量相似度搜索（余弦相似度）和 FTS5 BM25 全文搜索
- **多格式支持**: 支持 Markdown、PDF 和 Word (.docx) 文档
- **MCP 协议**: 内置 Model Context Protocol 服务器，支持 AI Agent 集成
- **REST API**: 完整的 HTTP API，支持搜索和 RAG 上下文生成
- **实时索引**: 文件系统监视器，自动索引文档变化
- **灵活配置**: YAML 配置文件，支持环境变量覆盖
- **监控指标**: Prometheus 兼容的指标端点
- **优雅关闭**: 完善的生产环境信号处理

### 系统架构

```
┌─────────────────────────────────────────────────────┐
│                     Cortex CLI                       │
├─────────────────────────────────────────────────────┤
│  index  │  search  │  context  │  serve  │  watch  │
├─────────────────────────────────────────────────────┤
│              混合搜索引擎                            │
│    向量搜索 (余弦)     │     FTS5 (BM25)           │
├─────────────────────────────────────────────────────┤
│              SQLite 存储层                           │
│   文档表 │ 分块表 │ 向量表 │ 全文索引               │
└─────────────────────────────────────────────────────┘
```

### 安装

```bash
# 克隆仓库
git clone https://github.com/lh123aa/cortex.git
cd cortex

# 构建
go build -o cortex ./cmd/cortex

# 或全局安装
go install ./cmd/cortex
```

### 快速开始

```bash
# 索引目录
./cortex index /path/to/docs

# 搜索
./cortex search "你的查询"

# 获取 RAG 上下文
./cortex context "你的查询"

# 启动 REST API 服务器
./cortex serve

# 启动 MCP 服务器（AI Agent 集成）
./cortex mcp
```

### 配置

Cortex 使用 YAML 配置文件，环境变量会覆盖配置文件中的值。

配置文件查找顺序：
1. `./config.yaml`
2. `~/.cortex/config.yaml`

#### 配置文件示例

```yaml
cortex:
  db_path: ~/.cortex/cortex.db
  log_level: info

embedding:
  provider: ollama  # 或 "onnx"
  ollama:
    base_url: http://localhost:11434
    model: nomic-embed-text
  onnx:
    base_url: http://localhost:8080
    model: embedder
    dim: 768

index:
  max_tokens: 512
  overlap_tokens: 64
  min_chars: 50
  workers: 4

search:
  cache_ttl: 5m
  default_top_k: 10

backup:
  enabled: true
  dir: ~/.cortex/backups
  max_backups: 10
  auto_backup: false
```

#### 环境变量

| 变量 | 描述 | 默认值 |
|------|------|--------|
| `CORTEX_DB_PATH` | 数据库文件路径 | `cortex.db` |
| `CORTEX_EMBEDDING_PROVIDER` | 提供商：`ollama` 或 `onnx` | `ollama` |
| `OLLAMA_HOST` | Ollama 服务器地址 | `http://localhost:11434` |
| `OLLAMA_MODEL` | Ollama 模型名称 | `nomic-embed-text` |
| `ONNX_BASE_URL` | ONNX 服务器地址 | `http://localhost:8080` |
| `ONNX_MODEL` | ONNX 模型名称 | `embedder` |

### CLI 命令

| 命令 | 描述 |
|------|------|
| `cortex index <路径>` | 索引目录 |
| `cortex search <查询>` | 搜索知识库 |
| `cortex context <查询>` | 获取 RAG 组装上下文 |
| `cortex watch <路径>` | 监视目录变化 |
| `cortex serve [--addr :8080]` | 启动 REST API 服务器 |
| `cortex mcp` | 启动 MCP 服务器（stdio） |
| `cortex backup` | 创建数据库备份 |
| `cortex version` | 显示版本 |

### REST API

启动服务器：`cortex serve`

| 端点 | 方法 | 描述 |
|------|------|------|
| `/health` | GET | 健康检查 |
| `/v1/search?q=<查询>` | GET | 混合搜索 |
| `/v1/context?q=<查询>&budget=<token数>` | GET | RAG 上下文 |
| `/v1/docs` | GET | 列出文档 |
| `/v1/docs/:id` | GET | 获取文档 |
| `/v1/stats` | GET | 统计信息 |

### MCP 集成

Cortex 提供 Model Context Protocol 服务器用于 AI Agent 集成：

```bash
# 启动 MCP 服务器
cortex mcp
```

可用工具：
- `cortex_search`: 混合向量 + FTS 语义搜索
- `cortex_context`: 在 token 预算内组装 RAG 上下文

### API 响应示例

#### 搜索响应

```json
{
  "query": "golang 并发",
  "total": 5,
  "results": [
    {
      "rank": 1,
      "score": 0.95,
      "path": "/docs/intro.md",
      "section": "基础知识 > 并发",
      "content_raw": "Go语言使用Goroutine实现并发...",
      "token_count": 128
    }
  ]
}
```

#### 上下文响应

```json
{
  "query": "golang 并发",
  "context": "Go语言使用Goroutine实现并发...\n\n---\n\n通道用于Goroutine通信...",
  "token_count": 850,
  "token_budget": 1500,
  "truncated": false
}
```

### 开发

```bash
# 运行测试
go test ./...

# 构建
go build -o cortex ./cmd/cortex
```

### 许可证

MIT
