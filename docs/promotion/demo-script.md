# Cortex Demo Script

用于制作 Demo GIF/视频或文章截图。

## 1. 下载 & 索引

```
$ curl -fsSL https://github.com/lh123aa/cortex/releases/latest/download/cortex-linux-amd64.zip | unzip -
$ chmod +x cortex

$ ./cortex index ~/my-docs
✅ Indexing complete:
   Total:   42 files
   Indexed: 42 files
   Failed:  0 files
   Time:    12.3s
```

## 2. 搜索

```
$ ./cortex search "Go concurrency patterns"

🔍 Search results for: Go concurrency patterns

1. [Score: 0.97] Go Concurrency Guide
   Use `go` keyword to start a goroutine:
   go func() { fmt.Println("Hello") }()

2. [Score: 0.88] Effective Go
   Channels are typed conduits for communication
```

## 3. RAG Context

```
$ ./cortex context "MCP protocol"

📝 RAG Context (budget: 4000 tokens):
[1] MCP Protocol Overview
MCP (Model Context Protocol) is a standard for AI Agent communication.
[2] MCP Transport
MCP supports two transport modes: stdio and SSE.
```

## 4. MCP Server

```
$ ./cortex mcp
MCP server started on stdio
Available tools: cortex_search, cortex_context, cortex_memory_write, cortex_memory_search, cortex_memory_delete
```

## 5. Status

```
$ ./cortex status
📊 Indexing Status:
   Total files:  42
   Total chunks: 1,247
   DB size:      28.4 MB
   Last indexed: 2026-05-05 19:37
```
