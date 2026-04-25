# Cortex 测试计划

## 测试环境要求

- Go 1.21+
- Ollama 服务 (localhost:11434)
- 8GB+ RAM

---

## 单元测试

### 1. 存储层测试

```bash
go test ./internal/storage/... -v
```

**测试用例:**
- [ ] SaveDocument - 保存文档
- [ ] GetDocumentByID - 获取文档
- [ ] DeleteDocumentByPath - 删除文档
- [ ] SearchCache - 缓存读写
- [ ] GetCacheStats - 缓存统计

### 2. 搜索层测试

```bash
go test ./internal/search/... -v
```

**测试用例:**
- [ ] VectorSearch - 向量搜索
- [ ] FTSSearch - 全文搜索
- [ ] HybridSearch - 混合搜索
- [ ] RRFMerge - 结果融合
- [ ] ScoreNormalization - 分数归一化

### 3. API测试

```bash
go test ./internal/api/... -v
```

**测试用例:**
- [ ] HealthCheck - 健康检查
- [ ] MemoryWrite - 记忆写入
- [ ] MemoryBatchWrite - 批量写入
- [ ] MemorySearch - 记忆搜索
- [ ] MemoryDelete - 记忆删除

---

## 集成测试

### 启动服务

```bash
# 终端1: 启动Ollama
ollama serve

# 终端2: 启动Cortex
go run cmd/cortex/main.go serve
```

### 1. 基础功能测试

#### 健康检查
```bash
curl http://localhost:8080/health
```

#### 索引导入
```bash
go run cmd/cortex/main.go index ./test_data
```

#### 搜索测试
```bash
curl "http://localhost:8080/v1/search?q=测试&top_k=5&mode=hybrid"
```

### 2. 记忆API测试

#### 写入记忆
```bash
curl -X POST http://localhost:8080/v1/memory \
  -H "Content-Type: application/json" \
  -d '{
    "content": "今天学习了Go语言的并发编程，理解了goroutine和channel的使用",
    "tags": ["Go", "并发编程"],
    "source": "study"
  }'
```

#### 批量写入
```bash
curl -X POST http://localhost:8080/v1/memory/batch \
  -H "Content-Type: application/json" \
  -d '{
    "memories": [
      {"content": "第一条记忆", "tags": ["测试"]},
      {"content": "第二条记忆", "source": "manual"}
    ]
  }'
```

#### 搜索记忆
```bash
curl "http://localhost:8080/v1/memory/search?q=Go语言"
```

#### 获取上下文
```bash
curl "http://localhost:8080/v1/memory/context?q=并发编程&budget=1000"
```

#### 删除记忆
```bash
curl -X DELETE http://localhost:8080/v1/memory/{memory_id}
```

### 3. RAG上下文测试

```bash
curl "http://localhost:8080/v1/context?q=什么是goroutine&budget=2000"
```

---

## 性能测试

### 1. 索引性能

```bash
time go run cmd/cortex/main.go index /path/to/1000_files
```

**目标:**
- 吞吐量 > 100 files/min
- 平均延迟 < 500ms/file

### 2. 搜索延迟

```bash
# 连续100次搜索
for i in {1..100}; do
  curl -s "http://localhost:8080/v1/search?q=test" > /dev/null
done
```

**目标:**
- P50 < 50ms
- P95 < 100ms
- P99 < 200ms

### 3. 缓存命中率

```bash
# 查询相同内容2次，第二次应该命中缓存
curl "http://localhost:8080/v1/search?q=same query"
curl "http://localhost:8080/v1/search?q=same query"
```

检查metrics:
```bash
curl http://localhost:9090/metrics | grep cortex_search_cache
```

---

## 监控验证

### Prometheus Metrics

```bash
curl http://localhost:9090/metrics
```

**关键指标检查:**
- `cortex_search_total` - 搜索计数
- `cortex_search_duration_seconds` - 延迟分布
- `cortex_search_cache_hits_total` vs `cortex_search_cache_misses_total` - 缓存效率
- `cortex_vectors_total` - 向量数量
- `cortex_index_chunks_total` - 分块数量

### Grafana Dashboard (可选)

导入 `docs/grafana-dashboard.json`

---

## 压力测试

### 使用wrk进行HTTP压力测试

```bash
wrk -t4 -c100 -d30s http://localhost:8080/v1/search?q=test
```

**目标:**
- QPS > 500
- 错误率 < 1%

---

## 回归测试清单

在每次发布前验证:

- [ ] `go build ./...` 编译成功
- [ ] `go test ./...` 所有测试通过
- [ ] 健康检查端点正常
- [ ] 记忆CRUD操作正常
- [ ] 搜索返回正确结果
- [ ] RAG上下文生成正常
- [ ] Prometheus指标正常暴露
- [ ] 缓存正确工作
- [ ] 用户隔离正确实现

---

## 测试数据

测试数据位于 `test_data/` 目录:

```
test_data/
├── documents/
│   ├── Go教程.md
│   ├── Python基础.md
│   └── 项目文档.txt
└── memories/
    └── seed_memories.json
```

---

## 已知问题

1. ~~WriteMemoryBatch 函数未正确保存记忆~~ - 已修复
2. 批量embedding时错误处理不完善
3. 记忆删除后缓存未失效
