# V2EX 推广帖

## 标题

🧠 开源分享：我写了个 AI Agent 的"第二大脑"——单二进制 + MCP 原生 + 零依赖

## 正文

分享一个自己用 Go 写的 AI Agent 本地知识库项目 **Cortex v2.2**。

### 解决了什么问题？

AI Agent（Claude Code、OpenCode、Cursor 等）没有记忆，每次对话都是白纸一张。
市面上要么太重（Dify 要 Docker Compose），要么太偏（ChromaDB 只是向量库），要么依赖太多。

所以我用 Go 写了一个：

### Cortex 是什么？

一个**单二进制文件**的本地知识库引擎，专为 AI Agent 设计。一个文件搞定：
- 📄 **文档索引** — MD / PDF / DOCX / 代码文件
- 🔍 **混合搜索** — 向量 + BM25 + RRF 融合排序
- 🧠 **Agent 记忆系统** — 跨会话持久化
- 🔌 **MCP 协议** — 5 个内置工具，与 Claude Code 等开箱即用

### 核心优势

**单二进制，零依赖**
```bash
curl -fsSL ... && chmod +x cortex
./cortex index ~/my-docs
./cortex mcp
```
不需要 Python、Node.js、Docker。v2.2 开始连 Ollama 都不需要了。

**纯 Go，零 CGO**
v2.2 切换到 `modernc.org/sqlite`，`go build` 一行命令编译，无需 gcc。

**生产级**
- 搜索 P50 < 50ms，P95 < 100ms
- L1+L2 两级缓存，10x 加速
- 114 个单元测试
- Prometheus 39 个指标

**隐私优先**
100% 本地，MIT 协议，商用自由。

### 技术栈

Go 1.21+ / SQLite+WAL / HNSW 向量 / MCP SDK / Prometheus

### 和竞品对比

| | Cortex | Mem0 | ChromaDB | Dify |
|--|--------|------|----------|------|
| 部署 | **单二进制** | pip/Docker | pip/Docker | Docker Compose |
| 依赖 | **零依赖** | 需 LLM API | 无 | 多服务 |
| MCP | **5 个工具** | 插件 | 不支持 | 1-2 个 |
| 记忆 | **内置** | 专注 | 无 | 基础 |
| 监控 | **Prometheus** | Dashboard | 无 | Grafana |

GitHub: https://github.com/lh123aa/cortex

欢迎 Star ⭐ 和提 Issue/PR！有任何问题或建议欢迎在回复里讨论 👇
