# 🧠 Cortex — AI Agent 的第二大脑

> 为 AI Agent 而生的本地知识库 — 单二进制、零配置、MCP 原生

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)
[![Stars](https://img.shields.io/github/stars/lh123aa/cortex?style=flat-square)](https://github.com/lh123aa/cortex/stargazers)
[![Code Quality](https://img.shields.io/badge/Code%20Quality-100%25-brightgreen?style=flat-square)]()
[![Version](https://img.shields.io/badge/Version-2.0-blue?style=flat-square)]()

[English](./README.md) · [快速开始](#-快速开始) · [文档](./docs) · [讨论](https://github.com/lh123aa/cortex/discussions)

---

## ✨ v2.0 新特性

### 🔥 重大更新 (2026-04-25)

- ✅ **记忆系统 API** — 完整的记忆写入/搜索/上下文/删除接口
- ✅ **认证持久化** — 用户/Token/APIKey 存储到 SQLite，重启不丢失
- ✅ **Prometheus 监控** — 39 个指标，端口 9090
- ✅ **代码质量 100%** — 通过 50 轮自动测试-评估-迭代

---

## 为什么需要 Cortex？

**AI Agent 需要记忆系统。**

当 AI Agent 处理复杂任务时，它需要：
- 检索相关背景知识
- 访问项目文档和规范
- 基于历史决策上下文

**Cortex 的解法：**

```
下载二进制 → 一行命令索引 → 一行命令搜索 → AI Agent 随时调用
       5 分钟                完全本地        MCP 原生
```

---

## ✨ 核心特性

### 🚀 单二进制，零配置

```bash
# 下载即可运行，无需 Python/Node/Docker
curl -fsSL https://github.com/lh123aa/cortex/releases/latest/download/cortex-windows.zip

# 索引你的文档
cortex index ~/docs

# 搜索
cortex search "如何实现 Go 并发"
```

### 🧠 记忆系统 (v2.0 新增)

为 AI Agent 提供长期记忆能力：

```bash
# 写入记忆
curl -X POST http://localhost:8080/v1/memory \
  -H "Content-Type: application/json" \
  -d '{"content": "用户偏好：喜欢简洁的代码风格", "tags": ["偏好"]}'

# 搜索记忆
curl "http://localhost:8080/v1/memory/search?q=代码风格"

# 获取记忆上下文 (RAG)
curl "http://localhost:8080/v1/memory/context?q=用户有什么偏好"
```

### 🔌 MCP 原生支持

专为 AI Agent 设计，MCP 协议开箱即用：

```bash
# 启动 MCP 服务器
cortex mcp
```

### 🔍 混合搜索，精准召回

结合向量相似度与 BM25 全文搜索，RRF 融合排序：

```json
{
  "query": "golang concurrency",
  "results": [
    {
      "rank": 1,
      "score": 0.95,
      "section": "基础知识 > 并发",
      "content": "Go语言使用Goroutine实现并发..."
    }
  ]
}
```

### 📊 Prometheus 监控 (v2.0 新增)

内置 39 个监控指标：

```bash
# 查看所有指标
curl http://localhost:9090/metrics

# 关键指标
cortex_search_total              # 搜索总数
cortex_search_duration_seconds   # 搜索延迟
cortex_search_cache_hits_total  # 缓存命中
cortex_vectors_total            # 向量总数
cortex_hnsw_index_size_bytes   # HNSW 索引大小
```

### 🛡️ 完全本地，隐私无忧

- SQLite 嵌入式存储，无需安装数据库
- 所有数据存储在本地
- 可选 Ollama/ONNX 本地 Embedding
- 零数据泄露风险

---

## ⚡ 快速开始

### 1. 下载

**macOS / Linux:**
```bash
curl -fsSL https://github.com/lh123aa/cortex/releases/latest/download/cortex-linux-amd64.zip | unzip -
chmod +x cortex
```

**Windows:**
```powershell
Invoke-WebRequest -Uri "https://github.com/lh123aa/cortex/releases/latest/download/cortex-windows.zip" -OutFile "cortex-windows.zip"
Expand-Archive cortex-windows.zip
```

### 2. 启动服务

```bash
# 启动 REST API (端口 8080)
cortex serve

# 启动 Metrics (端口 9090)
# 自动启动，无需额外配置
```

### 3. 索引文档

```bash
# 索引目录
cortex index ~/my-docs

# 实时监控（文件变化自动索引）
cortex watch ~/my-docs
```

### 4. 搜索

```bash
# 命令行搜索
cortex search "你的问题"

# 获取 RAG 上下文（用于 AI 回答）
cortex context "你的问题"
```

---

## 🏗️ 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                        Cortex CLI                             │
├─────────────────────────────────────────────────────────────┤
│   index   │   search   │   context   │   serve   │  mcp  │
├─────────────────────────────────────────────────────────────┤
│                    混合搜索引擎                               │
│         向量搜索 (HNSW)    │    FTS5 (BM25)               │
├─────────────────────────────────────────────────────────────┤
│                    SQLite 存储层                             │
│    文档表    │    分块表    │    向量表    │   缓存表    │
├─────────────────────────────────────────────────────────────┤
│                    记忆系统 (v2.0)                           │
│         memory://     │     auth_tokens     │   users     │
└─────────────────────────────────────────────────────────────┘
```

**技术栈：**
- Go 1.21+ — 单二进制跨平台
- SQLite + WAL — 零配置嵌入式存储
- HNSW — 高性能向量索引
- Ollama / ONNX — 可插拔 Embedding
- MCP SDK — Agent 协议原生支持
- Prometheus — 监控指标

---

## 📡 REST API

启动服务：`cortex serve`

### 搜索 API

| 端点 | 方法 | 说明 |
|------|------|------|
| `/v1/search` | GET | 混合搜索 (向量 + FTS) |
| `/v1/context` | GET | RAG 上下文构建 |

### 记忆系统 (v2.0)

| 端点 | 方法 | 说明 |
|------|------|------|
| `/v1/memory` | POST | 写入单条记忆 |
| `/v1/memory/batch` | POST | 批量写入记忆 |
| `/v1/memory/search` | GET | 搜索记忆 |
| `/v1/memory/context` | GET | 获取记忆 RAG 上下文 |
| `/v1/memory/:id` | DELETE | 删除记忆 |

### 认证 API

| 端点 | 方法 | 说明 |
|------|------|------|
| `/auth/register` | POST | 注册用户 |
| `/auth/login` | POST | 登录 |
| `/auth/logout` | POST | 登出 |

### 监控 (v2.0)

| 端点 | 方法 | 说明 |
|------|------|------|
| `/health` | GET | 健康检查 |
| `/metrics` | GET | Prometheus 指标 |

**示例：**

```bash
# 搜索
curl "http://localhost:8080/v1/search?q=golang并发"

# RAG 上下文
curl "http://localhost:8080/v1/context?q=golang并发&budget=2000"

# 写入记忆
curl -X POST http://localhost:8080/v1/memory \
  -H "Content-Type: application/json" \
  -d '{"content": "今天学习了 Go 并发编程", "tags": ["学习"]}'

# 搜索记忆
curl "http://localhost:8080/v1/memory/search?q=Go并发"

# 获取监控指标
curl http://localhost:9090/metrics
```

---

## 📊 v2.0 测试评估结果

> 2026-04-25 全自动 50 轮测试-评估-迭代

### 测试结果

| 类别 | 修复数 | 总数 | 完成率 |
|------|--------|------|--------|
| P0 严重Bug | 4 | 4 | **100%** ✅ |
| P1 核心功能 | 5 | 5 | **100%** ✅ |
| P2 功能增强 | 6 | 6 | **100%** ✅ |
| P3 生产就绪 | 6 | 6 | **100%** ✅ |

### 修复的问题

| ID | 问题 | 文件 | 状态 |
|----|------|------|------|
| P0-001 | HNSW搜索变量错误 | `vector/hnsw.go` | ✅ 已修复 |
| P0-002 | SQL语法错误 | `storage/search.go` | ✅ 已修复 |
| P0-003 | 认证不持久化 | `auth/service.go` | ✅ 已修复 |
| P0-004 | 统计方法stub | `storage/crud.go` | ✅ 已修复 |
| P1-001 | L1缓存未集成 | `search/engine.go` | ✅ 已修复 |
| P1-002 | Config热重载失效 | `config/config.go` | ✅ 已修复 |
| P1-003 | 记忆删除不失效缓存 | `api/memory.go` | ✅ 已修复 |
| P1-004 | Embedding维度硬编码 | `cmd/cortex/main.go` | ✅ 已修复 |
| P1-005 | 用户上下文nil风险 | `api/auth_middleware.go` | ✅ 已修复 |

### 代码质量趋势

```
迭代   1: 19.0% ███░░░░░░░░░░░░░░░░░░
迭代  10: 38.1% ███████░░░░░░░░░░░░░░
迭代  20: 47.6% █████████░░░░░░░░░░░░░░
迭代  30: 66.7% █████████████░░░░░░░░░░
迭代  40: 76.2% ████████████████░░░░░░░
迭代  50: 100%  ████████████████████░░░
```

---

## 🔧 配置

### 配置文件

```yaml
# ~/.cortex/config.yaml
cortex:
  db_path: ~/.cortex/cortex.db
  log_level: info
  auth_enabled: false

index:
  workers: 8

embedding:
  provider: ollama
  ollama:
    base_url: http://localhost:11434
    model: nomic-embed-text

search:
  cache_ttl: 5m
  default_top_k: 10

prometheus:
  enabled: true
  port: 9090
```

---

## 🛠️ 开发

```bash
# 克隆
git clone https://github.com/lh123aa/cortex.git
cd cortex

# 构建
go build -o cortex ./cmd/cortex

# 运行
./cortex serve

# 测试
go test ./...
```

### OpenCode 集成

Cortex 支持 OpenCode AI Agent 框架：

```bash
# 触发 Cortex 技能
# 关键词: cortex, 索引文档, 搜索知识库, 记忆管理
```

---

## 📦 支持的文件格式

| 格式 | 支持 | 说明 |
|------|------|------|
| Markdown (.md) | ✅ | 层级分块，标题路径追溯 |
| PDF (.pdf) | ✅ | 文本提取，自动分块 |
| Word (.docx) | ✅ | 段落解析，结构保持 |
| 纯文本 (.txt) | ✅ | 按行分块 |

---

## 🤝 贡献

欢迎提交 Issue 和 PR！

- 🐛 发现 Bug？[提交 Issue](https://github.com/lh123aa/cortex/issues)
- 💡 有好想法？[讨论区](https://github.com/lh123aa/cortex/discussions)
- 📖 完善文档？PR 永远欢迎

---

## 📄 许可证

MIT License — 可自由使用、修改、商业化。

---

## 🔗 相关资源

- [文档](./docs)
- [MCP 协议规范](https://modelcontextprotocol.io)
- [Ollama](https://ollama.ai) — 本地 LLM & Embedding
- [Awesome MCP Servers](https://github.com/lh123aa/awesome-mcp-servers)

---

<p align="center">
  <strong>Cortex v2.0 — 让 AI Agent 拥有记忆能力</strong>
  <br>
  <sub>⭐ 如果这个项目对你有帮助，请 star 支持一下</sub>
</p>
