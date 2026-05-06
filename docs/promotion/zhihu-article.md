# 🧠 我写了个 AI Agent 的"第二大脑"：单二进制 + MCP 原生 + 零依赖

> 给 Claude Code、OpenCode、Cursor 装上永久记忆

本文同步发布于 [掘金](https://juejin.cn) | [GitHub](https://github.com/lh123aa/cortex)

---

## 先说痛点

用过 AI Agent（比如 Claude Code、OpenCode、Cursor）的朋友一定遇到过这个问题：

**Agent 没有记忆。**

你让它查了项目文档，下一个对话它全忘了。每次都要重新说一遍上下文。你解释了你的技术偏好，下次对话它还是一张白纸。

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

它在一个二进制文件里整合了：

- 📄 **文档索引** — 支持 Markdown / PDF / DOCX / 代码文件
- 🔍 **混合搜索** — 向量语义 + BM25 关键词 + RRF 融合排序
- 🧠 **记忆系统** — Agent 的长期记忆，跨会话持久化
- 🔌 **MCP 协议** — AI Agent 原生通信，5 个内置工具
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

## 核心优势

### 1. 单二进制，零依赖

```bash
curl -fsSL https://github.com/lh123aa/cortex/releases/latest/download/cortex-linux-amd64.zip | unzip -
./cortex index ~/my-docs
./cortex mcp
```

不需要 Python、Node.js、Docker。v2.2 开始连 Ollama 都不需要了（内置 FTS5 全文搜索）。

### 2. MCP 协议原生

Cortex 内置 5 个 MCP 工具，开箱即用：

| 工具 | 功能 |
|----------|------|
| `cortex_search` | 混合搜索 |
| `cortex_context` | RAG 上下文组装 |
| `cortex_memory_write` | 写入 Agent 记忆 |
| `cortex_memory_search` | 搜索历史记忆 |
| `cortex_memory_delete` | 删除记忆 |

### 3. 生产级性能

| 指标 | 数值 |
|------|------|
| 搜索延迟 P50 | < 50ms（缓存命中 < 1ms） |
| L1+L2 两级缓存 | 搜索速度提升 10x |
| API 限流 | 100 req/s，突发 200 |
| 单元测试 | 114 个 |

### 4. 隐私优先

100% 本地运行。数据存在你本地的 SQLite 文件里，不需要任何外部服务。MIT 协议，商用自由。

---

## 技术选型：为什么用 Go？

1. **单二进制分发** — Go 编译出来就是一个文件，下载即用
2. **零 CGO** — v2.2 切换到 `modernc.org/sqlite`，无需 gcc，`go build` 一行编译
3. **嵌入式存储** — SQLite + WAL，一个文件就是整个数据库
4. **HNSW 向量搜索** — 性能跟 FAISS 差不多，但没有 Python 依赖

---

## 快速上手

```bash
# 下载
curl -fsSL https://github.com/lh123aa/cortex/releases/latest/download/cortex-linux-amd64.zip | unzip -
chmod +x cortex

# 索引文档
./cortex index ~/my-docs

# 启动 MCP 服务器
./cortex mcp

# 然后配置到你的 Claude Code / OpenCode 中即可
```

支持的文件格式：Markdown、PDF、DOCX、纯文本、代码文件（Go/Python/JS 等）、配置文件（YAML/JSON/TOML）

---

## 写在最后

Cortex 是完全开源的（MIT 协议），你可以随意使用、修改、商用。

如果对你有帮助：
- ⭐ **GitHub 点个 Star**：https://github.com/lh123aa/cortex
- 🐛 提 Issue 或 PR
- 📣 分享给朋友或团队

欢迎在评论区留言讨论！👇
