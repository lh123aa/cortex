# Twitter/X Thread

---

## 英文版

**Tweet 1:**
🧠 Cortex v2.2 is out!

A local knowledge base engine for AI Agents. Single Go binary, MCP native, zero dependencies.

Your Claude Code / OpenCode / Cursor can finally have permanent memory.

GitHub: https://github.com/lh123aa/cortex

**Tweet 2:**
Why I built it:
- Mem0 → needs Python + API keys
- ChromaDB → vector DB only, no agent memory
- Dify → Docker Compose, too heavy
- AnythingLLM → chat UI, not agent-native

I wanted: curl → unzip → it just works.

**Tweet 3:**
What's inside the box:
• Hybrid search (HNSW vector + BM25)
• Agent memory system (cross-session)
• 5 MCP tools (search/context/memory)
• L1+L2 cache (10x speed)
• Prometheus metrics
• All in ONE binary (no CGO!)

**Tweet 4:**
Quick start:
```
curl -fsSL [url] | unzip -
./cortex index ~/docs
./cortex mcp
```
10 seconds and your AI Agent has a knowledge base.

**Tweet 5:**
Tech stack:
• Go 1.21+ (pure Go, no CGO)
• SQLite + WAL
• modernc.org/sqlite
• HNSW vector search
• MCP SDK

114 unit tests, MIT license, 100% local.

**Tweet 6:**
Star it if you find it useful ⭐

https://github.com/lh123aa/cortex

Would love feedback from the community!

---

## 中文版

**Tweet 1:**
🧠 Cortex v2.2 发布了！

AI Agent 的本地知识库引擎。单 Go 二进制文件、MCP 原生、零依赖。

你的 Claude Code / OpenCode / Cursor 终于可以有永久记忆了。

GitHub: https://github.com/lh123aa/cortex

**Tweet 2:**
为什么自己写？
- Mem0 → 要 Python + API Key
- ChromaDB → 只有向量库，没有记忆系统
- Dify → Docker Compose，太重
- AnythingLLM → 聊天 UI，不是给 Agent 用的

我想要的是：下载 → 解压 → 直接能用。

**Tweet 3:**
核心能力：
• 混合搜索（HNSW 向量 + BM25）
• Agent 记忆系统（跨会话）
• 5 个 MCP 工具
• L1+L2 两级缓存（10x 加速）
• Prometheus 监控
• 全在一个二进制文件里！

**Tweet 4:**
10 秒上手：
```
curl -fsSL [链接] | unzip -
./cortex index ~/文档
./cortex mcp
```
你的 AI Agent 立刻拥有知识库。

**Tweet 5:**
技术栈：
• Go 1.21+（纯 Go，零 CGO）
• SQLite + WAL
• modernc.org/sqlite
• HNSW 向量搜索
• MCP SDK

114 个测试、MIT 协议、100% 本地、完全开源。

**Tweet 6:**
觉得有用的话点个 Star ⭐

https://github.com/lh123aa/cortex

欢迎反馈和建议！
