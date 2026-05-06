# GitHub PR 提交指南

> 本文档提供向 awesome-go 和 MCP Servers 目录提交 PR 的详细步骤。

---

## 前置条件

```bash
# 1. 安装 GitHub CLI
# macOS
brew install gh
# Windows (winget)
winget install GitHub.cli
# 或从 https://cli.github.com/ 下载

# 2. 登录 GitHub
gh auth login

# 3. 验证
gh auth status
```

---

## 一、提交 awesome-go

### 步骤

```bash
# 1. 克隆 awesome-go
git clone https://github.com/avelino/awesome-go.git
cd awesome-go

# 2. 创建分支
git checkout -b add-cortex

# 3. 编辑 README.md
# 在 AI > Knowledge Base 分类下添加:
# - [cortex](https://github.com/lh123aa/cortex) - AI Agent knowledge base with MCP native support, single binary deployment, hybrid search and agent memory system.

# 4. 提交
git add README.md
git commit -m "Add Cortex - AI Agent knowledge base with MCP native"

# 5. 推送到你的 Fork
# （第一次会自动 fork）
git push origin add-cortex

# 6. 创建 PR
gh pr create \
  --repo avelino/awesome-go \
  --title "Add Cortex - AI Agent knowledge base with MCP native" \
  --body "- **项目**: Cortex (https://github.com/lh123aa/cortex)
- **类别**: AI > Knowledge Base
- **说明**: AI Agent knowledge base with MCP native support, single binary deployment, hybrid search and agent memory system.
- **语言**: Go (100%)
- **License**: MIT
- **Stars**: ⭐ （提交时请更新实际数量）"
```

---

## 二、提交 MCP Servers 目录

### 步骤

```bash
# 1. 克隆 MCP Servers 项目
git clone https://github.com/modelcontextprotocol/servers.git
cd servers

# 2. 创建分支
git checkout -b add-cortex

# 3. 编辑 src/index.md
# 在 Knowledge Base / Memory 分类下添加:
# - [Cortex](https://github.com/lh123aa/cortex) - Local knowledge base with hybrid search and agent memory system. Single binary, MCP native, 100% local.

# 4. 提交
git add src/index.md
git commit -m "Add Cortex - MCP server for knowledge base and memory"

# 5. 推送到你的 Fork
git push origin add-cortex

# 6. 创建 PR
gh pr create \
  --repo modelcontextprotocol/servers \
  --title "Add Cortex - MCP server for knowledge base and agent memory" \
  --body "- **项目**: Cortex (https://github.com/lh123aa/cortex)
- **类别**: Knowledge Base / Memory
- **MCP 工具数量**: 5
- **说明**: Local knowledge base with hybrid search and agent memory system. Single binary, MCP native, 100% local.
- **亮点**: 零依赖部署、内置 HNSW 向量搜索、Agent 记忆系统、Prometheus 监控"
```

---

## 三、提交 Go Weekly

### 步骤

```bash
# 访问 https://golangweekly.com/
# 在页面底部找到 "sponsor" 或 "submit" 链接
# 或直接发邮件到 editors@golangweekly.com

# 邮件模板：

# Subject: Cortex v2.2 – AI Agent Knowledge Base in Pure Go
#
# Hi Go Weekly editors,
#
# I'd like to submit Cortex for consideration in an upcoming issue.
#
# Cortex is a local knowledge base engine for AI Agents, written entirely in Go.
# It provides MCP-native knowledge retrieval and memory for AI Agent tools like
# Claude Code, OpenCode, and Cursor.
#
# Key highlights:
# - Single binary, zero dependencies (not even Ollama needed)
# - MCP protocol native (5 built-in tools)
# - Pure Go SQLite (modernc.org/sqlite, no CGO)
# - Hybrid search (HNSW vector + BM25 + RRF fusion)
# - Built-in agent memory system with full CRUD
# - 114 unit tests, Prometheus monitoring
# - MIT license
#
# GitHub: https://github.com/lh123aa/cortex
#
# Best,
# [你的名字]
```

---

## 四、提交 Reddit r/golang

```bash
# 直接访问 https://www.reddit.com/r/golang/submit

# 标题:
# Cortex v2.2 – AI Agent knowledge base in Go, single binary, MCP native

# 内容: 使用 docs/promotion/hackernews-post.md 中的 Reddit 版本
```

---

## 五、检查进度

```bash
# 查看你的 PR 列表
gh pr list

# 查看 PR 详情
gh pr view <PR-number>
```

---

## 小贴士

1. **Star 数门槛**：awesome-go 通常建议至少 100+ stars 再提交，可以先发文章涨 Star
2. **PR 审核时间**：awesome-go 审核通常 1-2 周
3. **先发文章再 PR**：建议先发掘金/V2EX 文章，等 Star 数上升后再提交 PR
4. **保持更新**：PR 提交后关注 review 意见并及时回复
