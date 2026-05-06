#!/bin/bash
# ============================================
# Cortex Demo Recording Script (for asciinema)
# ============================================
# 使用方法:
#   1. 安装 asciinema: brew install asciinema / apt install asciinema
#   2. 运行: asciinema rec -c ./scripts/demo-recording.sh cortex-demo.cast
#   3. 录制完成后上传: asciinema upload cortex-demo.cast
#   4. 用 SVG 嵌入 README: [![asciicast](url)](url)
# ============================================

# 模拟终端输出
GREEN='\033[0;32m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo ""
echo "  🧠 Cortex v2.2 Demo"
echo "  ==================="
echo ""

# Step 1: 显示版本信息
echo -e "${BLUE}$ ./cortex --version${NC}"
sleep 1
echo "Cortex v2.2 (pure Go, no CGO)"
echo ""

# Step 2: 索引文档
echo -e "${BLUE}$ ./cortex index ./demo-docs/${NC}"
sleep 1
echo ""
echo -e "${GREEN}✅ Indexing complete:${NC}"
echo "   Total:      42 files"
echo "   Indexed:    42 files"
echo "   Failed:     0 files"
echo "   Chunks:     1,247 chunks"
echo "   Time:       12.3s"
echo ""

# Step 3: 搜索演示
echo -e "${BLUE}$ ./cortex search \"Go concurrency patterns\"${NC}"
sleep 1
echo ""
echo -e "${GREEN}🔍 Top 3 results:${NC}"
echo ""
echo "  [1] Score: 0.97"
echo "      Go Concurrency Guide"
echo "      Use 'go' keyword to start a goroutine:"
echo "      > go func() { fmt.Println(\"Hello\") }()"
echo ""
echo "  [2] Score: 0.88"
echo "      Effective Go - Channels"
echo "      Channels are typed conduits for communication"
echo ""
echo "  [3] Score: 0.76"
echo "      Go Wiki: Learn Concurrency"
echo "      Goroutines are lightweight threads"
echo ""

# Step 4: RAG Context 演示
echo -e "${BLUE}$ ./cortex context \"What is MCP protocol\"${NC}"
sleep 1
echo ""
echo -e "${GREEN}📝 RAG Context (budget: 4000 tokens):${NC}"
echo ""
echo "  [1] MCP Protocol Overview"
echo "      MCP (Model Context Protocol) is a standard"
echo "      for AI Agent communication..."
echo ""
echo "  [2] MCP Transport"
echo "      MCP supports stdio and SSE transport modes"
echo ""

# Step 5: 记忆系统演示
echo -e "${BLUE}$ cortex memory write --content \"User prefers Go language\" --type preference${NC}"
sleep 1
echo ""
echo -e "${GREEN}✅ Memory written (id: mem_abc123)${NC}"
echo ""

echo -e "${BLUE}$ cortex memory search \"user preference\"${NC}"
sleep 1
echo ""
echo -e "${GREEN}🧠 Memory results:${NC}"
echo ""
echo "  [1] User prefers Go language"
echo "      Type: preference | Created: 2026-05-05"
echo ""

# Step 6: MCP 服务器启动
echo -e "${BLUE}$ ./cortex mcp${NC}"
sleep 1
echo ""
echo -e "${GREEN}🧠 MCP server started on stdio${NC}"
echo -e "${GREEN}📦 Available tools:${NC}"
echo "   • cortex_search"
echo "   • cortex_context"
echo "   • cortex_memory_write"
echo "   • cortex_memory_search"
echo "   • cortex_memory_delete"
echo ""

# Step 7: REST API 演示
echo -e "${BLUE}$ curl http://localhost:8080/v1/search?q=vector+search${NC}"
sleep 1
echo ""
echo '{"results":[{"id":"doc1","score":0.95,"content":"Vector search using HNSW algorithm...",' 
echo '"metadata":{"source":"docs/vector-search.md","chunk_type":"section"}}],'
echo '"total":12,"time_ms":4}'
echo ""

# Step 8: Status
echo -e "${BLUE}$ ./cortex status${NC}"
sleep 1
echo ""
echo -e "${GREEN}📊 Indexing Status:${NC}"
echo "   Total files:  42"
echo "   Total chunks: 1,247"
echo "   DB size:      28.4 MB"
echo "   Cache hits:   156 (L1: 89, L2: 67)"
echo "   Uptime:       3d 12h 45m"
echo ""

echo -e "${PURPLE}========================================${NC}"
echo -e "${PURPLE}  🧠 Cortex is ready for your AI Agent!  ${NC}"
echo -e "${PURPLE}========================================${NC}"
