# SegmentFault 思否文章

## 标题

用 Go 给 AI Agent 装一个"第二大脑"：Cortex 本地知识库实战指南

---

## 正文

### 背景

最近 AI Agent 开发工具（Claude Code、OpenCode、Cursor）越来越流行，但它们都有一个共同的痛点：**没有长期记忆**。每次开启新会话，Agent 都对你的项目、偏好、之前的结论一无所知。

解决这个问题需要两个能力：
1. **文档检索**（RAG）— 让 Agent 能搜索知识库
2. **记忆持久化** — 让 Agent 能记住用户偏好和历史

市面上有不少方案，但我想要的是：
- ❌ 不要 Docker Compose（太重）
- ❌ 不要 Python 环境（部署麻烦）
- ❌ 不要外部 API（隐私顾虑）
- ✅ 一个二进制文件搞定

所以我用 Go 写了 **Cortex**。

### 核心设计

#### 搜索引擎：混合检索

Cortex 用了组合搜索策略：

1. **向量搜索（HNSW）** — 理解语义相似度
2. **BM25（FTS5）** — 精确关键词匹配
3. **RRF 融合** — 两种结果加权合并

这在搜索技术文档时特别有效。比如搜"如何实现 Go 并发"，语义搜索能找到 goroutine 相关章节，关键词搜索能精确命中"并发"这个词，两种结果互补。

#### 记忆系统：Agent 的长期记忆

记忆系统和搜索系统不同。搜索是找文档，记忆是找"之前 Agent 知道什么"。

```bash
# 让 Agent 记住你的偏好
cortex memory write --content "用户偏好 Go 语言开发" --type preference

# 下次会话，Agent 能搜索到
cortex memory search "用户偏好"
```

这对跨会话的工作流特别有用。比如你告诉 Agent "这个项目的测试目录在 tests/ 下"，写入记忆后，下次打开新会话 Agent 仍然记得。

#### 技术选型

**为什么用 Go？**

Go 的单二进制特性让分发变得极其简单。用户不需要装任何运行时，下载一个文件就能用。

**为什么纯 Go SQLite？**

v2.2 从 `mattn/go-sqlite3`（CGO 版本）切换到了 `modernc.org/sqlite`（纯 Go 版本）。好处：
- 不需要 gcc
- 交叉编译简单
- 构建时间缩短

**为什么内置 MCP？**

MCP 是 AI Agent 的"USB 协议"。通过 MCP，任何支持该协议的 Agent 都能直接调用 Cortex 的能力，无需额外适配。

### 实战：给 Claude Code 加上知识库

1. 下载 Cortex：
```bash
curl -fsSL https://github.com/lh123aa/cortex/releases/latest/download/cortex-linux-amd64.zip | unzip -
chmod +x cortex
```

2. 索引项目文档：
```bash
./cortex index ./docs
```

3. 启动 MCP 服务器：
```bash
./cortex mcp
```

4. 在 Claude Code 的 MCP 配置中加入：
```json
{
  "mcpServers": {
    "cortex": {
      "command": "./cortex",
      "args": ["mcp"]
    }
  }
}
```

现在你的 Claude Code 就能搜索知识库了！

### 性能数据

- 搜索 P50 延迟 < 50ms
- 缓存命中 < 1ms
- 两级缓存（内存 + SQLite）提升 10x
- 索引吞吐量 > 100 文件/分钟

### 开源和社区

Cortex 使用 MIT 协议，完全开源。

GitHub: https://github.com/lh123aa/cortex

---

*欢迎在评论中分享你的使用场景和建议！*
