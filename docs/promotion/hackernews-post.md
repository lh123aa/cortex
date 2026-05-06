# Hacker News / Reddit 英文推广帖

---

## Hacker News — Show HN

**Title:**
Show HN: Cortex v2.2 – Local knowledge base for AI Agents. Single binary, MCP native, zero deps

**Body:**

I built Cortex, a local knowledge base engine for AI Agents (Claude Code, OpenCode, Cursor, etc.).

It's a single Go binary that gives AI Agents permanent memory and document search capabilities.

**Why another knowledge base tool?**

Existing solutions have friction:
- Mem0 needs Python + API keys
- ChromaDB/Qdrant are vector DBs only (no agent memory concept)
- Dify requires Docker Compose
- AnythingLLM is a chat UI, not an Agent tool

I wanted something I can `curl` and start using in 10 seconds.

**What makes Cortex different:**

- Single binary, zero dependencies (not even Ollama needed in v2.2)
- MCP protocol native — 5 built-in tools that any MCP-compatible Agent can call
- Hybrid search (HNSW vector + BM25 + RRF fusion)
- Built-in agent memory system (cross-session persistence)
- 100% local, MIT license
- Prometheus monitoring (39 metrics)

**Quick start:**

```bash
curl -fsSL https://github.com/lh123aa/cortex/releases/latest/download/cortex-linux-amd64.zip | unzip -
chmod +x cortex
./cortex index ~/my-docs
./cortex mcp   # Starts MCP server, now your Agent has a knowledge base!
```

**Tech stack:**
- Pure Go (no CGO), single binary
- SQLite + WAL for storage
- HNSW for vector search
- modernc.org/sqlite (pure Go SQLite driver)
- Prometheus for monitoring

**Performance:**
- Search P50 < 50ms (cache hit < 1ms)
- L1+L2 two-level cache (in-memory + SQLite)
- 114 unit tests
- 100+ files/min indexing throughput

GitHub: https://github.com/lh123aa/cortex

Would love to hear your feedback!

---

## Reddit r/golang

**Title:**
Cortex v2.2 – AI Agent knowledge base in a single Go binary, MCP native, zero CGO

**Body:**

Hey Gophers! I've been working on Cortex, a local knowledge base engine for AI Agents written entirely in Go.

What I love about this stack:
- `modernc.org/sqlite` — pure Go SQLite, no CGO at all
- HNSW vector search implemented in Go
- Single binary cross-compilation (the killer Go feature)

The v2.2 release is a milestone — I switched from CGO SQLite to pure Go, which means `go build` just works with no gcc needed.

It serves as an MCP (Model Context Protocol) server with 5 tools that Claude Code, OpenCode and other AI Agents can use directly.

If you're building AI tooling or just interested in Go + vector search implementations, check it out:

https://github.com/lh123aa/cortex

Happy to answer questions about the architecture!

---

## Reddit r/LocalLLaMA (备选)

**Title:**
Cortex v2.2 – Local knowledge base for AI Agents, MCP native, single binary

**Body:**
Just shipped v2.2 of Cortex, a local knowledge base engine for AI Agents.

Key features for the local AI community:
- 100% local, no cloud dependencies
- FTS5-only mode in v2.2 — no Ollama or any external service needed
- Pure Go SQLite — no Python/CGO required
- 5 MCP tools for Agent memory and search
- Prometheus monitoring included

Great companion for local LLM setups.

GitHub: https://github.com/lh123aa/cortex
