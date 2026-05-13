<p align="center">
  <img src="docs/assets/img/cortex-logo-static.svg" width="120" height="120" alt="Cortex Logo">
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/License-MIT-green?style=for-the-badge" alt="MIT">
  <img src="https://img.shields.io/badge/Version-2.2-blue?style=for-the-badge" alt="v2.2">
  <img src="https://goreportcard.com/badge/github.com/lh123aa/cortex?style=for-the-badge" alt="Go Report Card">
  <img src="https://img.shields.io/badge/Tests-114_passing-green?style=for-the-badge" alt="Tests">
  <img src="https://img.shields.io/badge/MCP-Native-7B61FF?style=for-the-badge" alt="MCP">
  <img src="https://img.shields.io/github/actions/workflow/status/lh123aa/cortex/build.yml?style=for-the-badge&logo=github" alt="Build">
  <img src="https://img.shields.io/github/stars/lh123aa/cortex?style=for-the-badge&logo=github" alt="Stars">
</p>

<h1 align="center">🧠 Cortex</h1>
<h3 align="center">The Second Brain for AI Agents — Local Knowledge Base · Single Binary · MCP Native</h3>

<p align="center">
  <b>Cortex</b> is a local knowledge base engine purpose-built for AI Agents. One single binary, native <b>MCP protocol</b> support, built-in <b>hybrid search</b> (Vector + BM25) and <b>Agent Memory System</b>. 100% local, zero external dependencies.
</p>
<p align="center">
  Give Claude Code, OpenCode, Cursor and other AI Agents a permanent memory 🧠
</p>
<p align="center">
  <a href="https://github.com/lh123aa/cortex"><b>GitHub</b></a> ·
  <a href="https://lh123aa.github.io/cortex/"><b>🎨 Landing Page</b></a>
</p>

<p align="center">
  <a href="docs/index.html">
    <img src="docs/assets/img/terminal-demo.svg" width="100%" alt="Terminal Demo" style="max-width:720px;border-radius:16px;border:1px solid rgba(123,97,255,0.15);">
  </a>
  <br>
  <sub>⬆️ Click to view the full landing page (Xiaomi/Mimo style)</sub>
</p>

<p align="center">
  <a href="#-product-comparison">📊 Comparison</a> ·
  <a href="#-quick-start">⚡ Quick Start</a> ·
  <a href="#-core-features">✨ Features</a> ·
  <a href="#-architecture">🏗️ Architecture</a> ·
  <a href="#-api">📡 API</a> ·
  <a href="#-configuration">🔧 Config</a>
  <br>
  <a href="./README.md">🌐 中文版</a> ·
  <a href="./docs/TUTORIAL.en.md">📖 Tutorial</a> ·
  <a href="#-development">🛠️ Dev</a> ·
  <a href="./docs">📖 Docs</a>
</p>

---

## 📊 Product Comparison

<table>
<thead>
<tr>
  <th>Feature</th>
  <th align="center">🚀 Cortex</th>
  <th align="center">Mem0</th>
  <th align="center">AnythingLLM</th>
  <th align="center">ChromaDB</th>
  <th align="center">Qdrant</th>
  <th align="center">Dify</th>
</tr>
</thead>
<tbody>
<tr>
  <td colspan="7"><strong>📦 Deployment</strong></td>
</tr>
<tr>
  <td>Deployment</td>
  <td align="center">✅ <strong>Single Binary</strong><br><sub>Download & run</sub></td>
  <td align="center">⚠️ pip/Docker<br><sub>Python required</sub></td>
  <td align="center">⚠️ Docker/Desktop<br><sub>Node.js</sub></td>
  <td align="center">⚠️ pip/Docker<br><sub>Python</sub></td>
  <td align="center">✅ <strong>Single Binary</strong><br><sub>Download & run</sub></td>
  <td align="center">⚠️ Docker Compose<br><sub>Multi-service</sub></td>
</tr>
<tr>
  <td>Dependencies</td>
  <td align="center">✅ <strong>Zero</strong><br><sub>Ollama optional</sub></td>
  <td align="center">❌ LLM API</td>
  <td align="center">❌ LLM API</td>
  <td align="center">✅ None</td>
  <td align="center">✅ None</td>
  <td align="center">❌ Multiple</td>
</tr>
<tr>
  <td colspan="7"><strong>🤖 AI Agent Integration</strong></td>
</tr>
<tr>
  <td>MCP Protocol</td>
  <td align="center">✅ <strong>Native</strong><br><sub>cortex mcp</sub></td>
  <td align="center">⚠️ Plugin</td>
  <td align="center">✅ Supported</td>
  <td align="center">❌ No</td>
  <td align="center">❌ No</td>
  <td align="center">✅ Supported</td>
</tr>
<tr>
  <td>MCP Tools</td>
  <td align="center">🔧 <strong>5 tools</strong><br><sub>search/context/memory</sub></td>
  <td align="center">🔧 1-2</td>
  <td align="center">🔧 1</td>
  <td align="center">—</td>
  <td align="center">—</td>
  <td align="center">🔧 1-2</td>
</tr>
<tr>
  <td>Agent Memory</td>
  <td align="center">✅ <strong>Built-in</strong><br><sub>Long-term + RAG</sub></td>
  <td align="center">✅ <strong>Focused</strong><br><sub>Multi-level</sub></td>
  <td align="center">❌ Chat only</td>
  <td align="center">❌ Vector DB</td>
  <td align="center">❌ Vector DB</td>
  <td align="center">⚠️ Basic</td>
</tr>
<tr>
  <td colspan="7"><strong>🔍 Search</strong></td>
</tr>
<tr>
  <td>Search Type</td>
  <td align="center">✅ <strong>Hybrid</strong><br><sub>Vector+BM25+RRF</sub></td>
  <td align="center">✅ Hybrid</td>
  <td align="center">✅ Vector</td>
  <td align="center">✅ Vector/Hybrid</td>
  <td align="center">✅ Hybrid</td>
  <td align="center">⚠️ Backend dep.</td>
</tr>
<tr>
  <td>File Formats</td>
  <td align="center">📄 <strong>MD/PDF/DOCX</strong><br><sub>+code files</sub></td>
  <td align="center">— <sub>Memory only</sub></td>
  <td align="center">✅ <strong>Multi</strong></td>
  <td align="center">— <sub>Vectors only</sub></td>
  <td align="center">— <sub>Vectors only</sub></td>
  <td align="center">✅ Multi</td>
</tr>
<tr>
  <td colspan="7"><strong>📊 Operations</strong></td>
</tr>
<tr>
  <td>Monitoring</td>
  <td align="center">✅ <strong>Prometheus</strong><br><sub>39 metrics</sub></td>
  <td align="center">⚠️ Dashboard</td>
  <td align="center">⚠️ Basic</td>
  <td align="center">❌ None</td>
  <td align="center">❌ None</td>
  <td align="center">✅ Grafana</td>
</tr>
<tr>
  <td>Caching</td>
  <td align="center">✅ <strong>L1+L2</strong><br><sub>Memory+SQLite</sub></td>
  <td align="center">⚠️ Basic</td>
  <td align="center">⚠️ Basic</td>
  <td align="center">❌ None</td>
  <td align="center">✅ Memory</td>
  <td align="center">⚠️ Basic</td>
</tr>
<tr>
  <td>Privacy</td>
  <td align="center">✅ <strong>100% Local</strong><br><sub>Fully offline</sub></td>
  <td align="center">⚠️ Local/Cloud</td>
  <td align="center">✅ 100% Local</td>
  <td align="center">✅ 100% Local</td>
  <td align="center">✅ 100% Local</td>
  <td align="center">⚠️ Local/Cloud</td>
</tr>
<tr>
  <td>License</td>
  <td align="center">✅ <strong>MIT</strong><br><sub>Free use/ modify/commercial</sub></td>
  <td align="center">✅ Apache 2.0</td>
  <td align="center">✅ MIT</td>
  <td align="center">✅ Apache 2.0</td>
  <td align="center">✅ Apache 2.0</td>
  <td align="center">⚠️ Restricted</td>
</tr>
</tbody>
</table>

> **Cortex Differentiator**: The only tool that combines single-binary deployment + native MCP protocol + built-in agent memory + hybrid search + Prometheus monitoring — purpose-built for AI Agent scenarios.

---

## 🎯 Use Cases

| Use Case | Description |
|----------|-------------|
| 🤖 **AI Agent Memory** | Give Claude Code / OpenCode / Cursor persistent memory across sessions |
| 📚 **Team Knowledge Base** | Index team wikis, technical docs, and project specs into a searchable RAG knowledge base |
| 🔍 **Codebase Search** | Index Go/Python/JS code for natural language semantic search |
| 🏢 **Enterprise Docs** | Private, local document retrieval for internal knowledge—data never leaves your network |
| 🧪 **RAG Backend** | Serve as the retrieval layer for RAG pipelines via REST API and MCP dual protocol |
| 🔐 **Privacy-First** | Finance, healthcare, legal — deploy 100% locally for sensitive data |

---

## ✨ Changelog

### 🧠 v2.2 — MCP Memory Tools (2026-05-05)

- ✅ **5 MCP Tools** — `cortex_search` / `cortex_context` / `cortex_memory_write` / `cortex_memory_search` / `cortex_memory_delete`
- ✅ **Zero-Dependency Mode** — `embedding.provider: none`, FTS5-only, no Ollama required
- ✅ **Pure Go SQLite** — Switched to `modernc.org/sqlite`, no CGO/gcc needed
- ✅ **MCP Graceful Shutdown** — Signal handling, safe Ctrl+C exit
- ✅ **MCP Unit Tests** — 11 test cases covering all tool edge conditions

### 🔥 v2.1 — Production Hardening (2026-04-25)

- ✅ **L1+L2 Two-Level Cache** — Memory + SQLite, 10x search speed
- ✅ **Graceful Shutdown** — 30s window for in-flight requests
- ✅ **Request Timeout** — Default 30s, search 60s, index 5min
- ✅ **Rate Limiting** — Token bucket, 100 req/s burst 200
- ✅ **36 Test Cases** — Storage/Auth/Search core modules

### ✨ v2.0 — Core Features

- ✅ **Memory System API** — Full CRUD for agent memory
- ✅ **Auth Persistence** — Users/Tokens/APIKeys in SQLite
- ✅ **Prometheus Monitoring** — 39 metrics on port 9090

---

## ⚡ Quick Start

```bash
# 1. Download
# macOS/Linux
curl -fsSL https://github.com/lh123aa/cortex/releases/latest/download/cortex-linux-amd64.zip | unzip -
chmod +x cortex

# Windows
# Invoke-WebRequest -Uri "..." -OutFile "cortex.zip"

# 2. Index your documents
cortex index ~/my-docs

# 3. Start MCP server (for AI Agent integration)
cortex mcp

# 4. Search
cortex search "How to implement Go concurrency"
```

---

## ✨ Core Features

<div align="center">
<table>
<tr>
<td align="center" width="33%">
<h3>🚀 Single Binary</h3>
<p><sub>Download & run. No Python/Node/Docker<br><code>curl → cortex mcp</code></sub></p>
</td>
<td align="center" width="33%">
<h3>🔌 MCP Native</h3>
<p><sub>5 MCP tools for AI Agents<br><code>cortex_search</code> · <code>cortex_memory_write</code></sub></p>
</td>
<td align="center" width="33%">
<h3>🧠 Memory System</h3>
<p><sub>Long-term memory + RAG context<br>Cross-session user preference recall</sub></p>
</td>
</tr>
<tr>
<td align="center" width="33%">
<h3>🔍 Hybrid Search</h3>
<p><sub>Vector semantics + BM25 keywords<br>RRF fusion for precision recall</sub></p>
</td>
<td align="center" width="33%">
<h3>⚡ L1+L2 Cache</h3>
<p><sub>In-memory + SQLite two-level cache<br>10x search speed improvement</sub></p>
</td>
<td align="center" width="33%">
<h3>📊 Prometheus</h3>
<p><sub>39 monitoring metrics<br><code>:9090/metrics</code></sub></p>
</td>
</tr>
</table>
</div>

### 🔌 MCP Tools Reference

| Tool | Description | REST API Equivalent |
|------|-------------|--------------------|
| `cortex_search` | Hybrid search (vector + BM25) | `GET /v1/search` |
| `cortex_context` | RAG context assembly | `GET /v1/context` |
| `cortex_memory_write` | Write a memory entry | `POST /v1/memory` |
| `cortex_memory_search` | Search memory entries | `GET /v1/memory/search` |
| `cortex_memory_delete` | Delete a memory entry | `DELETE /v1/memory/:id` |

---

## 🏗️ Architecture

```
┌──────────────────────────────────────────────────────────┐
│                      Cortex CLI                          │
├──────────────────────────────────────────────────────────┤
│  index  │  search  │  context  │  serve  │     mcp       │
├──────────────────────────────────────────────────────────┤
│                  Hybrid Search Engine                     │
│       HNSW Vector Search    │    FTS5 (BM25)            │
├──────────────────────────────────────────────────────────┤
│               L1+L2 Cache Layer (v2.1)                    │
│          go-cache (memory)   │     SQLite                │
├──────────────────────────────────────────────────────────┤
│                     SQLite Storage                        │
│   Documents │  Chunks  │  Vectors  │  Cache  │  Users    │
├──────────────────────────────────────────────────────────┤
│              MCP Protocol · REST API                      │
│    5 MCP Tools  │  15+ REST Endpoints  │  Prometheus     │
└──────────────────────────────────────────────────────────┘
```

**Tech Stack:**
- **Language**: Go 1.21+ — Cross-platform single binary (pure Go, no CGO)
- **Storage**: SQLite + WAL — Zero-config embedded database
- **Vector**: HNSW — High-performance approximate nearest neighbor search
- **Embedding**: Ollama / ONNX / None（FTS5-only mode）
- **Protocol**: MCP SDK — Native AI Agent communication
- **Monitoring**: Prometheus — 39 built-in metrics

---

## 📡 API Reference

| Category | Endpoint | Method | Description |
|----------|----------|--------|-------------|
| **Search** | `/v1/search` | GET | Hybrid search (vector + FTS) |
| | `/v1/context` | GET | RAG context assembly |
| **Memory** | `/v1/memory` | POST | Write memory entry |
| | `/v1/memory/batch` | POST | Batch write memories |
| | `/v1/memory/search` | GET | Search memories |
| | `/v1/memory/context` | GET | Memory RAG context |
| | `/v1/memory/:id` | DELETE | Delete memory |
| **Auth** | `/auth/register` | POST | Register user |
| | `/auth/login` | POST | Login |
| | `/auth/logout` | POST | Logout |
| **Monitor** | `/health` | GET | Health check |
| | `/metrics` | GET | Prometheus metrics（:9090） |

---

## 🔧 Configuration

```yaml
# ~/.cortex/config.yaml
cortex:
  db_path: ~/.cortex/cortex.db
  log_level: info
  auth_enabled: false

embedding:
  provider: ollama    # ollama | onnx | none（FTS5-only, no external service）
  ollama:
    base_url: http://localhost:11434
    model: nomic-embed-text

index:
  workers: 8
  max_tokens: 512

search:
  cache_ttl: 5m
  default_top_k: 10

prometheus:
  enabled: true
  port: 9090
```

---

## 📦 Supported File Formats

| Format | Support | Chunking Method |
|--------|---------|----------------|
| Markdown (.md) | ✅ | Hierarchical, heading-path traceable |
| PDF (.pdf) | ✅ | Text extraction, auto-chunking |
| Word (.docx) | ✅ | Paragraph parsing, structure preserved |
| Plain Text (.txt) | ✅ | Line/paragraph based |
| Code (.go/.py/.js/.ts/.java etc.) | ✅ | Function/class based |
| Config (.yaml/.json/.toml/.ini) | ✅ | Structured extraction |

---

## 🛠️ Development

```bash
git clone https://github.com/lh123aa/cortex.git
cd cortex
go build -o cortex ./cmd/cortex   # Pure Go, no CGO required
./cortex serve
go test ./...                     # 114 tests
```

---

## 📊 Performance

| Metric | Value |
|--------|-------|
| Search Latency P50 | < 50ms (cache hit < 1ms) |
| Search Latency P95 | < 100ms |
| Cache Hit Rate | > 60% (L1+L2) |
| Index Throughput | > 100 files/min |
| Test Coverage | 114 unit tests |

---

## 🤝 Contributing

- 🐛 Found a bug? [Open an Issue](https://github.com/lh123aa/cortex/issues)
- 💡 Have an idea? [Start a Discussion](https://github.com/lh123aa/cortex/discussions)
- ⭐ Like the project? Give it a Star!

## 📄 License

**MIT License** — Free to use, modify, and commercialize.

---

## ⭐ Star History

[![Star History Chart](https://api.star-history.com/svg?repos=lh123aa/cortex&type=Date)](https://star-history.com/#lh123aa/cortex&Date)

---

## 💬 Community & Support

- ⭐ **Star the repo** — The best way to show support
- 🐛 **Report bugs** — [Open an Issue](https://github.com/lh123aa/cortex/issues)
- 💡 **Feature ideas** — [Start a Discussion](https://github.com/lh123aa/cortex/discussions)
- 📣 **Share it** — Recommend Cortex on HN, Reddit, Twitter
- 🤝 **Contribute** — Read [CONTRIBUTING.md](CONTRIBUTING.md)

## 🔗 Resources

- [MCP Protocol Specification](https://modelcontextprotocol.io) — AI Agent communication standard
- [Ollama](https://ollama.ai) — Local LLM & Embedding
- [OpenCode](https://opencode.ai) — AI Agent Framework
- [Awesome MCP Servers](https://github.com/lh123aa/awesome-mcp-servers) — MCP server list

---

<p align="center">
  <strong>🧠 Cortex v2.2 — Give AI Agents a Memory</strong>
  <br>
  <sub>Single Binary · Zero Config · MCP Native · 100% Local · MIT License</sub>
</p>

<p align="center">
  <a href="https://github.com/lh123aa/cortex/stargazers">
    <img src="https://img.shields.io/github/stars/lh123aa/cortex?style=social" alt="Star">
  </a>
  <a href="https://github.com/lh123aa/cortex/forks">
    <img src="https://img.shields.io/github/forks/lh123aa/cortex?style=social" alt="Fork">
  </a>
  <a href="https://github.com/lh123aa/cortex">
    <img src="https://img.shields.io/github/followers/lh123aa?style=social" alt="Follow">
  </a>
</p>
