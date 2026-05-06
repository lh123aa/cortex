# 🧠 我写了个 AI Agent 的"第二大脑"：单二进制 + MCP 原生 + 零依赖

> 给 Claude Code、OpenCode、Cursor 装上永久记忆

---

## 先说痛点

用过 AI Agent（比如 Claude Code、OpenCode、Cursor）的朋友一定遇到过这个问题：

**Agent 没有记忆。**

你让它查了项目文档，下一个对话它全忘了。每次都要重新说一遍上下文。你解释了你的技术偏好，下次对话它还是一张白纸。就好像跟一个有失忆症的人对话——很痛苦。

市面上有解决方案吗？有。但我在调研了一圈之后发现，现有的方案各有各的硬伤：

| 方案 | 问题 |
|------|------|
| **Mem0** | 要 Python 环境，要 API Key，不是给 Agent 直接用的 |
| **ChromaDB / Qdrant** | 只是向量数据库，没有 Agent 记忆的概念，要自己搭全套 |
| **Dify** | Docker Compose 全套部署，太重了 |
| **AnythingLLM** | 好用，但定位是聊天 UI，不是给 Agent 集成的 |

所以我自己用 Go 写了一个：**Cortex**。

---

## 什么是 Cortex？

Cortex 是一个 **本地知识库引擎**，专门为 AI Agent 设计。

它在一个二进制文件里（没错，就一个文件）整合了：

- 📄 **文档索引** — 支持 Markdown / PDF / DOCX / 代码文件 / 配置文件
- 🔍 **混合搜索** — 向量语义搜索 + BM25 关键词搜索 + RRF 融合排序
- 🧠 **记忆系统** — Agent 的长期记忆，跨会话持久化
- 🔌 **MCP 协议** — AI Agent 原生通信协议，5 个内置工具
- 📊 **Prometheus 监控** — 39 个指标，生产就绪

### 核心架构

```
┌──────────────────────────────────────────────┐
│                 Cortex CLI                    │
├──────────────────────────────────────────────┤
│  index  │  search  │  context  │  mcp         │
├──────────────────────────────────────────────┤
│          混合搜索引擎                         │
│   HNSW 向量搜索    │    FTS5 (BM25)          │
├──────────────────────────────────────────────┤
│        L1+L2 两级缓存                         │
│   go-cache (内存)   │   SQLite               │
├──────────────────────────────────────────────┤
│              SQLite 存储层                     │
│  文档 │ 分块 │ 向量 │ 缓存 │ 用户              │
├──────────────────────────────────────────────┤
│        MCP 协议 · REST API 双协议              │
└──────────────────────────────────────────────┘
```

---

## 为什么和其他方案不一样？

### 1. 单二进制，零依赖

```bash
# 下载就一个文件
curl -fsSL https://github.com/lh123aa/cortex/releases/latest/download/cortex-linux-amd64.zip | unzip -

# 索引文档
./cortex index ~/my-docs

# 启动 MCP 服务（供 AI Agent 使用）
./cortex mcp
```

不需要 Python，不需要 Node.js，不需要 Docker。甚至 v2.2 开始连 **Ollama 都不需要了**（内置 FTS5 全文搜索，零依赖模式）。

这在和其他方案对比时优势很明显：

| 维度 | Cortex | Mem0 | ChromaDB | Dify |
|------|--------|------|----------|------|
| 部署方式 | **单二进制** | pip/Docker | pip/Docker | Docker Compose |
| 外部依赖 | **零依赖** | 需 LLM API | 无 | 多服务 |
| 启动时间 | **< 1秒** | 分钟级 | 分钟级 | 分钟级 |

### 2. MCP 协议原生

MCP (Model Context Protocol) 是 Anthropic 推出的 AI Agent 通信标准。Claude Code、OpenCode、Cursor 等都支持。

Cortex 内置了 **5 个 MCP 工具**，开箱即用：

| MCP 工具 | 功能 | 
|----------|------|
| `cortex_search` | 混合搜索（向量 + BM25） |
| `cortex_context` | RAG 上下文组装 |
| `cortex_memory_write` | 写入 Agent 记忆 |
| `cortex_memory_search` | 搜索历史记忆 |
| `cortex_memory_delete` | 删除记忆 |

在 Claude Code 中配置一下 MCP 服务器，你的 Agent 就能直接搜索知识库、记住用户偏好了。

### 3. 生产级性能

这不是一个玩具项目。Cortex 经过了两轮生产环境加固：

| 指标 | 数值 |
|------|------|
| 搜索延迟 P50 | < 50ms（缓存命中 < 1ms） |
| 搜索延迟 P95 | < 100ms |
| L1+L2 两级缓存 | 搜索速度提升 10x |
| API 限流 | 100 req/s，突发 200 |
| 请求超时控制 | 默认 30s，搜索 60s，索引 5min |
| 单元测试 | 114 个，覆盖率可靠 |

### 4. 隐私优先

100% 本地运行。数据存在你自己的 SQLite 文件里，不需要任何外部服务。MIT 协议，商用自由。

这对金融、医疗、法律等隐私敏感场景特别重要。

---

## 技术选型：为什么用 Go？

我选 Go 有几个核心考量：

**1. 单二进制分发**

Go 编译出来就是一个二进制文件，下载即用。用户不需要装任何运行时。

对比一下：
- Python 方案 → 用户要装 Python、pip install、各种依赖
- Node.js 方案 → 用户要装 Node、npm install
- Rust 方案 → 编译慢，生态相对小众
- **Go → go build 一个文件搞定**

**2. 零 CGO**

v2.2 我特意把 SQLite 驱动从 `mattn/go-sqlite3`（需要 CGO + gcc）切换到了 `modernc.org/sqlite`（纯 Go 实现）。

这意味着：
- 不需要安装 gcc
- `go build` 一行命令编译
- 交叉编译超级简单

**3. 嵌入式存储**

SQLite + WAL 模式，零配置。用户不需要安装 PostgreSQL 或 MySQL。一个文件就是整个数据库。

向量搜索我用了 **HNSW 算法**（分层可导航小世界图），性能跟 FAISS 差不多，但没有 Python 依赖。

---

## 快速上手

### 作为 MCP 服务器（推荐）

```bash
# 1. 下载
curl -fsSL https://github.com/lh123aa/cortex/releases/latest/download/cortex-linux-amd64.zip | unzip -
chmod +x cortex

# 2. 索引你的文档
./cortex index ~/my-docs

# 3. 启动 MCP 服务器
./cortex mcp
```

然后在 Claude Code 或 OpenCode 中配置 MCP 服务器即可。

### 作为 REST API 服务

```bash
# 启动 HTTP 服务
./cortex serve

# 搜索
curl "http://localhost:8080/v1/search?q=Go+并发&top_k=5"

# RAG 上下文
curl "http://localhost:8080/v1/context?q=什么是MCP协议&max_tokens=2000"

# 写入记忆
curl -X POST "http://localhost:8080/v1/memory" \
  -H "Content-Type: application/json" \
  -d '{"content":"用户偏好使用Go语言开发","type":"preference"}'
```

---

## 支持的文档格式

| 格式 | 支持 | 分块方式 |
|------|------|---------|
| Markdown (.md) | ✅ | 层级分块，标题路径追溯 |
| PDF (.pdf) | ✅ | 文本提取，自动分块 |
| Word (.docx) | ✅ | 段落解析，结构保持 |
| 纯文本 (.txt) | ✅ | 按行/段落分块 |
| 代码 (.go/.py/.js文件) | ✅ | 按函数/类分块 |
| 配置 (.yaml/.json等) | ✅ | 结构化提取 |

---

## 未来规划

| 版本 | 计划功能 | 预计时间 |
|------|---------|---------|
| v2.3 | 多模态支持（图片、音频索引） | 进行中 |
| v2.4 | 分布式部署 / 多节点同步 | 规划中 |
| v3.0 | 插件系统 / 自定义分块策略 | 规划中 |

---

## 写在最后

Cortex 是完全开源的（MIT 协议），你可以随意使用、修改、商用。

如果你：
- 🧑‍💻 在用 AI Agent 但受困于"没有记忆"
- 📚 需要给自己的文档搭一个可搜索的知识库
- 🔐 数据不能上云，需要纯本地方案
- 🦀 Go 爱好者，想看看纯 Go 做的向量搜索 + MCP 实现

那这个项目可能对你有帮助。

**GitHub：** [https://github.com/lh123aa/cortex](https://github.com/lh123aa/cortex)

如果觉得有用，欢迎：
- ⭐ **点个 Star**（这是对开源项目最大的支持）
- 🐛 提 Issue 或 PR
- 📣 分享给朋友或团队

也欢迎在评论区留言讨论你的使用场景或建议！

---

*本文同步发布在 [GitHub](https://github.com/lh123aa/cortex) | [掘金]() | [知乎]()*
