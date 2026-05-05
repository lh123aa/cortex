# Cortex × OpenCode 迭代计划

> 基于系统性评估生成的 4-Phase 迭代路线图
> 目标：让 Cortex 在 OpenCode 中发挥完整价值

---

## Phase 1 — MCP 暴露记忆工具

**优先级**: ★★★★★ (最高)
**预计工作量**: ~120 行
**目标文件**: `internal/api/mcp.go`
**风险**: 低

### 改动内容

在 `internal/api/mcp.go` 的 `registerTools()` 中新增 3 个 MCP Tool：

| Tool 名称 | 功能 | 映射的 REST API |
|-----------|------|----------------|
| `cortex_memory_write` | 写入单条记忆 | `POST /v1/memory` |
| `cortex_memory_search` | 搜索记忆 | `GET /v1/memory/search` |
| `cortex_memory_delete` | 删除记忆 | `DELETE /v1/memory/:id` |

### 实现步骤

1. 在 `mcp.go` 中新增 args 结构体：
   - `MemoryWriteArgs` — content, tags, source, summary
   - `MemorySearchArgs` — query, top_k
   - `MemoryDeleteArgs` — id

2. 在 `NewMCPServer` 中注入 `embedding.EmbeddingProvider` + `chunker.Chunker`：
```go
// 改动前
func NewMCPServer(se *search.HybridSearchEngine, st storage.Storage, log *zap.Logger)

// 改动后
func NewMCPServer(se *search.HybridSearchEngine, st storage.Storage, em embedding.EmbeddingProvider, log *zap.Logger)
```

3. 新建 `MemoryHandler` 实例（复用 `internal/api/memory.go` 逻辑），注册 handler

4. 更新 `cmd/cortex/main.go:366` 的 `NewMCPServer` 调用，传入 embedding

### 验收标准

- 启动 `cortex mcp` 后，OpenCode 可发现 `cortex_memory_write` / `cortex_memory_search` / `cortex_memory_delete` 三个工具
- 写入记忆后可通过搜索召回
- 删除记忆后不再出现在搜索结果中

---

## Phase 2 — MCP 优雅关闭

**优先级**: ★★★★
**预计工作量**: ~15 行
**目标文件**: `cmd/cortex/main.go`
**风险**: 零

### 改动内容

为 `runMCP()` 增加 Signal 处理，与 `runServe()` 保持一致：

```go
func runMCP(cmd *cobra.Command, args []string) {
    // ... 现有初始化代码 ...

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-sigChan
        logger.Info("shutting down MCP server...")
        // MCP Server 的 stdio 传输会在 main 退出时自动关闭
        os.Exit(0)
    }()

    if err := mcpServer.Run(); err != nil {
        logger.Error("MCP server error", zap.Error(err))
        os.Exit(1)
    }
}
```

### 验收标准

- 运行 `cortex mcp` 后按 Ctrl+C，进程在 1 秒内退出
- 退出前日志输出 "shutting down MCP server..."

---

## Phase 3 — 补充 MCP 单测

**优先级**: ★★★
**预计工作量**: ~150 行
**目标文件**: `internal/api/mcp_test.go`（新建）
**风险**: 低

### 测试覆盖

| 测试用例 | 说明 |
|---------|------|
| `TestMCPSearchHandler` | 正常搜索返回正确格式 |
| `TestMCPContextHandler` | RAG 上下文构建 |
| `TestMCPMemoryWriteHandler` | 写入记忆（依赖 Phase 1） |
| `TestMCPMemorySearchHandler` | 搜索记忆（依赖 Phase 1） |
| `TestMCPMemoryDeleteHandler` | 删除记忆（依赖 Phase 1） |
| `TestMCPEmptyQuery` | 空查询处理 |
| `TestMCPTopKDefault` | 默认 TopK 值 |

### 技术方案

- 使用 `internal/api/auth_test.go` 中的 Gin TestMode 模式
- 通过 `storage.NewSQLiteStorage(":memory:")` 使用内存 SQLite
- 对 embedding 使用 mock provider

### 依赖

- Phase 1 完成后方可测试记忆相关工具
- 可先写 `search` / `context` 的测试，独立于 Phase 1

---

## Phase 4 — 弱化 Ollama 硬依赖

**优先级**: ★★★
**预计工作量**: ~200 行
**目标文件**: `internal/embedding/`, `internal/search/engine.go`, `cmd/cortex/main.go`
**风险**: 低

### 改动内容

1. **新增 `noop` Embedding provider** (`internal/embedding/noop.go`)：
   - 返回固定维度零向量
   - `Health()` 永远返回 nil（表示可用）

2. **搜索引擎降级支持** (`internal/search/engine.go`)：
   - 当向量维度为 0 或 `search_mode == "bm25"` 时跳过向量搜索
   - 仅在混合模式下报 warning

3. **配置扩展**：
   ```yaml
   embedding:
     provider: "none"  # 新增：不启用 Embedding，仅 FTS5
   ```

4. **`initEmbedding` 逻辑更新** (`main.go:147`)：
   - `provider: "none"` → 返回 noop provider
   - 打印提示 "running in FTS5-only mode, vector search disabled"

### 验收标准

- 不启动 Ollama，设置 `provider: none`，索引和搜索能正常工作（仅 BM25）
- 启动 Ollama + `provider: ollama`，功能完全不受影响
- 日志清晰提示当前搜索模式

---

## 执行顺序依赖图

```
Phase 1 ──────────────────────┐
                              ├──→ Phase 3 (MCP 测试)
Phase 2 (独立, 可随时做) ────┘

Phase 4 (独立, 可随时做)
```

## 总工作量估算

| Phase | 文件数 | 行数 | 工时估计 |
|-------|--------|------|---------|
| 1 | 2 | ~120 | 2-3 轮 |
| 2 | 1 | ~15 | 1 轮 |
| 3 | 1 | ~150 | 2 轮 |
| 4 | 3 | ~200 | 2-3 轮 |
| **合计** | **7** | **~485** | **7-9 轮** |

---

## 验证流程

每 Phase 完成后执行：
```bash
go build ./...
go test ./...
```

最终集成验证：
```bash
# 启动 Ollama
ollama serve

# 构建并启动 MCP
go build -o cortex.exe ./cmd/cortex
./cortex.exe mcp

# 在 OpenCode 中触发：
# - "搜索 xxx"
# - "添加记忆"
# - "查找记忆"
```

---

*生成时间: 2026-05-05*
*基于系统性评估 4 项改进建议*
