# 开源中国 (OSCHINA) 项目提交

## 推荐标题

🧠 Cortex v2.2 — AI Agent 本地知识库引擎，单二进制、MCP 原生、混合搜索

## 项目简介

Cortex 是一个用 Go 语言编写的本地知识库引擎，专为 AI Agent 设计。只需一个二进制文件，即可为 Claude Code、OpenCode、Cursor 等 AI Agent 提供知识库检索和长期记忆能力。

## 核心特性

- **单二进制部署**：下载即用，无需 Python/Node.js/Docker，甚至无需 Ollama
- **MCP 协议原生**：5 个内置 MCP 工具，与 AI Agent 开箱即用
- **混合搜索**：HNSW 向量搜索 + BM25 全文搜索 + RRF 融合排序
- **Agent 记忆系统**：跨会话持久化记忆，支持写入/搜索/删除
- **100% 本地隐私**：数据存储在本地 SQLite，无需任何外部服务
- **生产级性能**：搜索 P50 < 50ms，L1+L2 两级缓存，Prometheus 监控
- **多格式支持**：Markdown、PDF、DOCX、代码文件、配置文件
- **MIT 开源协议**：可自由使用、修改、商业化

## 技术栈

- 语言：Go 1.21+（纯 Go，零 CGO）
- 存储：SQLite + WAL
- 向量：HNSW 算法
- 嵌入：Ollama / ONNX / FTS5-only（零依赖模式）
- 协议：MCP SDK + REST API
- 监控：Prometheus（39 个指标）

## 快速开始

```bash
# 下载
curl -fsSL https://github.com/lh123aa/cortex/releases/latest/download/cortex-linux-amd64.zip | unzip -
chmod +x cortex

# 索引文档
./cortex index ~/my-docs

# 启动 MCP 服务器
./cortex mcp
```

## 项目地址

https://github.com/lh123aa/cortex
