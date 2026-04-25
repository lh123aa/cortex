# Cortex API 文档

## 基础信息

- **REST API地址**: `http://localhost:8080`
- **Metrics地址**: `http://localhost:9090/metrics`
- **认证**: 可选（默认关闭）

---

## 健康检查

### GET /health
检查服务健康状态

**响应:**
```json
{
  "status": "ok",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

### GET /health/ready
检查服务就绪状态

### GET /health/live
检查服务存活状态

---

## 搜索API

### GET /v1/search

混合搜索（向量 + BM25）

**参数:**
| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| q | string | 必填 | 搜索查询 |
| top_k | int | 10 | 返回结果数量 |
| mode | string | hybrid | 搜索模式: vector/fts/hybrid |

**响应:**
```json
{
  "query": "搜索内容",
  "total": 10,
  "results": [
    {
      "rank": 1,
      "score": 0.95,
      "path": "/path/to/doc",
      "section": "标题路径",
      "content_raw": "文档内容...",
      "token_count": 150
    }
  ]
}
```

---

## RAG上下文

### GET /v1/context

获取RAG上下文用于LLM增强

**参数:**
| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| q | string | 必填 | 查询内容 |
| budget | int | 1500 | token预算 |

**响应:**
```json
{
  "query": "问题",
  "context": "上下文内容...",
  "token_count": 1200,
  "token_budget": 1500,
  "truncated": false
}
```

---

## 文档管理

### GET /v1/docs
列出已索引的文档

**响应:**
```json
{
  "total": 100,
  "documents": [...]
}
```

### GET /v1/docs/:id
获取指定文档详情

---

## 统计信息

### GET /v1/stats
获取系统统计信息

---

## 记忆系统 API

### POST /v1/memory

写入单条记忆

**请求体:**
```json
{
  "content": "记忆内容，必填",
  "summary": "摘要（可选，自动生成）",
  "tags": ["标签1", "标签2"],
  "source": "conversation"
}
```

**响应 (201):**
```json
{
  "id": "abc123...",
  "content": "记忆内容",
  "summary": "摘要",
  "tags": ["标签1"],
  "source": "conversation",
  "created_at": "2024-01-01T00:00:00Z"
}
```

---

### POST /v1/memory/batch

批量写入记忆

**请求体:**
```json
{
  "memories": [
    {
      "content": "记忆内容1",
      "tags": ["标签1"]
    },
    {
      "content": "记忆内容2",
      "source": "manual"
    }
  ]
}
```

**响应 (201):**
```json
{
  "total": 2,
  "success": 2,
  "failed": 0,
  "results": [...],
  "errors": []
}
```

---

### GET /v1/memory/search

搜索记忆

**参数:**
| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| q | string | 必填 | 搜索查询 |
| top_k | int | 10 | 返回结果数量 |

**响应:**
```json
{
  "query": "搜索内容",
  "total": 5,
  "results": [
    {
      "id": "chunk-id",
      "content": "记忆内容...",
      "summary": "摘要",
      "score": 0.92,
      "source": "memory://id"
    }
  ]
}
```

---

### GET /v1/memory/context

获取记忆的RAG上下文

**参数:**
| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| q | string | 必填 | 查询内容 |
| budget | int | 1500 | token预算 |

**响应:**
```json
{
  "query": "问题",
  "context": "记忆上下文内容...",
  "token_count": 800,
  "token_budget": 1500,
  "truncated": false
}
```

---

### DELETE /v1/memory/:id

删除记忆

**参数:**
- id: 记忆ID

**响应 (200):**
```json
{
  "message": "memory deleted"
}
```

---

## 认证 API

### POST /auth/register
注册新用户

### POST /auth/login
用户登录

### POST /auth/logout
用户登出

---

## 管理 API

### GET /admin/keys
列出API密钥

---

## Prometheus 指标

### GET /metrics

Prometheus格式的监控指标

**主要指标:**
- `cortex_index_total` - 已索引文档总数
- `cortex_search_total` - 搜索请求总数
- `cortex_search_duration_seconds` - 搜索延迟
- `cortex_search_cache_hits_total` - 缓存命中数
- `cortex_search_cache_misses_total` - 缓存未命中数
- `cortex_embedding_duration_seconds` - Embedding生成延迟
- `cortex_vectors_total` - 向量总数
- `cortex_hnsw_index_size_bytes` - HNSW索引内存大小

---

## 内部 API

### GET /internal/progress/:root_path

获取索引进度

---

## 错误响应格式

所有API错误返回以下格式:

```json
{
  "error": "错误描述"
}
```

HTTP状态码:
- 400: 请求参数错误
- 401: 未授权
- 403: 禁止访问
- 500: 服务器内部错误
