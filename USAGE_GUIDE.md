# Cortex 本地知识库使用指南

## 1. 快速开始

### 1.1 构建 Cortex

```bash
# 克隆项目
git clone https://github.com/lh123aa/cortex.git
cd cortex

# 构建
go build -o cortex ./cmd/cortex

# 或直接安装
go install ./cmd/cortex
```

### 1.2 启动 Ollama（已安装则跳过）

```bash
# 安装 Ollama (macOS/Linux)
curl -fsSL https://ollama.com/install.sh | sh

# 或 Windows: 从 https://ollama.com/download 下载

# 启动 Ollama 服务
ollama serve

# 下载 embedding 模型 (另一个终端)
ollama pull nomic-embed-text
```

### 1.3 索引文档

```bash
# 创建数据库目录
mkdir -p ~/.cortex

# 索引你的文档目录
./cortex index ~/Documents

# 或指定路径
./cortex index /path/to/your/docs
```

### 1.4 搜索

```bash
# 混合搜索 (向量 + 全文)
./cortex search "你的查询"

# 仅向量搜索
./cortex search "查询" --mode vector

# 仅全文搜索
./cortex search "查询" --mode fts
```

### 1.5 启动 API 服务器

```bash
# 启动 REST API (默认 :8080)
./cortex serve

# 自定义端口
./cortex serve --addr :9090

# 启用 API Key 认证
./cortex serve --auth
```

### 1.6 RAG 上下文

```bash
# 获取 RAG 上下文 (1500 token budget)
./cortex context "你的问题"

# 自定义 token 预算
./cortex context "问题" --budget 3000
```

---

## 2. API 使用

### 2.1 搜索接口

```bash
curl "http://localhost:8080/v1/search?q=golang%20concurrency&top_k=5"
```

响应：
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

### 2.2 RAG 上下文接口

```bash
curl "http://localhost:8080/v1/context?q=golang%20并发&budget=1500"
```

响应：
```json
{
  "query": "golang 并发",
  "context": "Go语言使用Goroutine实现并发...\n\n---\n\n通道用于Goroutine通信...",
  "token_count": 850,
  "token_budget": 1500,
  "truncated": false
}
```

### 2.3 健康检查

```bash
# 基础健康
curl http://localhost:8080/health

# Kubernetes 就绪探针
curl http://localhost:8080/health/ready

# Kubernetes 存活探针
curl http://localhost:8080/health/live
```

### 2.4 API Key 认证

```bash
# 启用认证后，所有 /v1/* 请求需要携带 API Key
curl -H "X-API-Key: your-api-key" \
     "http://localhost:8080/v1/search?q=test"
```

---

## 3. MCP 协议集成 (AI Agent)

Cortex 提供 MCP Server，可用于 AI Agent 集成：

```bash
# 启动 MCP 服务器 (stdio 模式)
./cortex mcp

# 可用工具:
# - cortex_search: 语义搜索
# - cortex_context: RAG 上下文组装
```

在 AI Agent 中配置 MCP 端点为 `stdio` 模式即可使用。

---

## 4. 配置

配置文件位置：`~/.cortex/config.yaml`

```yaml
cortex:
  db_path: ~/.cortex/cortex.db
  log_level: info

embedding:
  provider: ollama  # or "onnx"
  ollama:
    base_url: http://localhost:11434
    model: nomic-embed-text

index:
  max_tokens: 512
  overlap_tokens: 64
  min_chars: 50
  workers: 4

search:
  cache_ttl: 5m
  default_top_k: 10
```

---

## 5. 目录结构

```
~/.cortex/
├── cortex.db           # SQLite 数据库
├── config.yaml         # 配置文件
├── backups/            # 自动备份
└── cortex_vector_idx.json  # 向量索引文件
```

---

## 6. 常见问题

### Q: 索引很慢怎么办？
A: 增加 workers 数量，或使用更快的 embedding 模型。

### Q: Ollama 连接失败？
A: 确保 Ollama 服务正在运行 (`ollama serve`)，且模型已下载 (`ollama list`)。

### Q: 如何查看索引状态？
A: `curl http://localhost:8080/v1/stats`

### Q: 如何重新索引？
A: 删除数据库后重新索引：`rm ~/.cortex/cortex.db && ./cortex index ~/Documents`