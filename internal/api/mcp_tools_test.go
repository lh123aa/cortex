package api

import (
	"context"
	"strings"
	"testing"

	"github.com/lh123aa/cortex/internal/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// callToolArgs 调用一个带类型参数的工具 handler，返回文本输出与 IsError。
func callToolArgs[T any](t *testing.T, srv *MCPServer, fn func(context.Context, *mcp.CallToolRequest, T) (*mcp.CallToolResult, any, error), args T) (string, bool) {
	t.Helper()
	res, _, err := fn(context.Background(), makeArgs(t, args), args)
	if err != nil {
		t.Fatalf("tool returned transport error: %v", err)
	}
	if res == nil {
		t.Fatal("tool returned nil result")
	}
	return resultTextOf(res), res.IsError
}

func resultTextOf(res *mcp.CallToolResult) string {
	if res == nil {
		return ""
	}
	var sb strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			sb.WriteString(tc.Text)
		}
	}
	return sb.String()
}

// TestMCPToolCount 确保 MCP 工具集扩展到 14 个。
func TestMCPToolCount(t *testing.T) {
	srv, _, cleanup := setupMCPTest(t)
	defer cleanup()

	if got := srv.ToolCount(); got != 14 {
		t.Fatalf("registered tool count = %d, want 14", got)
	}
}

// TestAllToolsAreCallable 逐个调用 14 个工具，确保都能正常返回。
func TestAllToolsAreCallable(t *testing.T) {
	srv, st, cleanup := setupMCPTest(t)
	defer cleanup()

	_ = st.SaveDocument(&models.Document{
		ID: "d1", Path: "/proj/a.md", FileType: "md", Status: "indexed", ChunkCount: 1,
	})
	_ = st.SaveChunks([]*models.Chunk{{
		ID: "c1", DocumentID: "d1", Content: "File: /proj/a.md | section\n\nCortex 支持混合检索。",
		ContentRaw: "Cortex 支 持 混 合 检 索 。", HeadingPath: "检索",
	}})

	check := func(name string, text string, isErr bool) {
		t.Helper()
		if isErr {
			t.Fatalf("%s returned error result: %s", name, text)
		}
		if strings.TrimSpace(text) == "" {
			t.Fatalf("%s returned empty text", name)
		}
	}

	t.Run("cortex_search", func(t *testing.T) {
		text, e := callToolArgs(t, srv, srv.handleSearchTool, SearchArgs{Query: "混合检索", TopK: 3})
		check("cortex_search", text, e)
	})
	t.Run("cortex_context", func(t *testing.T) {
		text, e := callToolArgs(t, srv, srv.handleContextTool, ContextArgs{Query: "混合检索", TokenBudget: 500})
		check("cortex_context", text, e)
	})
	t.Run("cortex_health", func(t *testing.T) {
		text, e := callToolArgs(t, srv, srv.handleHealthTool, struct{}{})
		check("cortex_health", text, e)
	})
	t.Run("cortex_suggest", func(t *testing.T) {
		text, e := callToolArgs(t, srv, srv.handleSuggestTool, SuggestArgs{Query: "混", TopK: 3})
		check("cortex_suggest", text, e)
	})
	t.Run("cortex_map", func(t *testing.T) {
		text, e := callToolArgs(t, srv, srv.handleMapTool, MapArgs{Limit: 5})
		check("cortex_map", text, e)
	})
	t.Run("cortex_stats", func(t *testing.T) {
		text, e := callToolArgs(t, srv, srv.handleStatsTool, StatsArgs{Verbose: true})
		check("cortex_stats", text, e)
	})
	t.Run("cortex_plan", func(t *testing.T) {
		text, e := callToolArgs(t, srv, srv.handlePlanTool, PlanArgs{Question: "混合检索怎么工作"})
		check("cortex_plan", text, e)
	})
	t.Run("cortex_project_context", func(t *testing.T) {
		text, e := callToolArgs(t, srv, srv.handleProjectContextTool, ProjectContextArgs{Path: "/proj", TopK: 3})
		check("cortex_project_context", text, e)
	})
	t.Run("cortex_code_search", func(t *testing.T) {
		text, e := callToolArgs(t, srv, srv.handleCodeSearchTool, CodeSearchArgs{Query: "混合检索", TopK: 3})
		check("cortex_code_search", text, e)
	})
	t.Run("cortex_memory_write", func(t *testing.T) {
		text, e := callToolArgs(t, srv, srv.handleMemoryWriteTool, MemoryWriteArgs{Content: "一条测试记忆内容，长度足够触发分块处理。"})
		check("cortex_memory_write", text, e)
	})
	t.Run("cortex_memory_search", func(t *testing.T) {
		text, e := callToolArgs(t, srv, srv.handleMemorySearchTool, MemorySearchArgs{Query: "测试", TopK: 3})
		check("cortex_memory_search", text, e)
	})
	t.Run("cortex_memory_batch", func(t *testing.T) {
		text, e := callToolArgs(t, srv, srv.handleMemoryBatchTool, MemoryBatchArgs{
			Items: []MemoryBatchItem{{Content: "批量记忆一，用于验证工具可用。"}, {Content: "批量记忆二，用于验证工具可用。"}},
		})
		check("cortex_memory_batch", text, e)
	})
}

func TestCortexMapReportsCounts(t *testing.T) {
	srv, st, cleanup := setupMCPTest(t)
	defer cleanup()

	for i, p := range []string{"/a/x.md", "/a/y.md", "/b/z.go"} {
		_ = st.SaveDocument(&models.Document{
			ID: "d" + string(rune('a'+i)), Path: p,
			FileType: strings.TrimPrefix(p[strings.LastIndex(p, "."):], "."), Status: "indexed",
		})
	}

	text, isErr := callToolArgs(t, srv, srv.handleMapTool, MapArgs{Limit: 10})
	if isErr {
		t.Fatalf("map error: %s", text)
	}
	for _, want := range []string{"Knowledge Map", "Documents:", "File types:", "Top directories:"} {
		if !strings.Contains(text, want) {
			t.Fatalf("map output missing %q:\n%s", want, text)
		}
	}
	if !strings.Contains(text, "md") || !strings.Contains(text, "go") {
		t.Fatalf("map should list extensions, got:\n%s", text)
	}
}

func TestCortexStatsShowsPlanAndLimits(t *testing.T) {
	srv, _, cleanup := setupMCPTest(t)
	defer cleanup()

	text, isErr := callToolArgs(t, srv, srv.handleStatsTool, StatsArgs{})
	if isErr {
		t.Fatalf("stats error: %s", text)
	}
	for _, want := range []string{"Cortex Stats", "Documents:", "Chunks:", "Plan:", "Storage:"} {
		if !strings.Contains(text, want) {
			t.Fatalf("stats output missing %q:\n%s", want, text)
		}
	}
}

func TestCortexPlanGivesConcreteSteps(t *testing.T) {
	srv, _, cleanup := setupMCPTest(t)
	defer cleanup()

	text, isErr := callToolArgs(t, srv, srv.handlePlanTool, PlanArgs{Question: "how does indexing work"})
	if isErr {
		t.Fatalf("plan error: %s", text)
	}
	for _, want := range []string{"Retrieval plan", "cortex_map", "cortex_search", "cortex_context"} {
		if !strings.Contains(text, want) {
			t.Fatalf("plan output missing %q:\n%s", want, text)
		}
	}
}

func TestCortexPlanRequiresQuestion(t *testing.T) {
	srv, _, cleanup := setupMCPTest(t)
	defer cleanup()

	_, isErr := callToolArgs(t, srv, srv.handlePlanTool, PlanArgs{})
	if !isErr {
		t.Fatal("empty question should return error result")
	}
}

func TestCortexCodeSearchRequiresQuery(t *testing.T) {
	srv, _, cleanup := setupMCPTest(t)
	defer cleanup()

	_, isErr := callToolArgs(t, srv, srv.handleCodeSearchTool, CodeSearchArgs{})
	if !isErr {
		t.Fatal("empty query should return error result")
	}
}

func TestCortexProjectContextRequiresPath(t *testing.T) {
	srv, _, cleanup := setupMCPTest(t)
	defer cleanup()

	_, isErr := callToolArgs(t, srv, srv.handleProjectContextTool, ProjectContextArgs{})
	if !isErr {
		t.Fatal("empty path should return error result")
	}
}

func TestCortexProjectContextNoMatchGivesGuidance(t *testing.T) {
	srv, _, cleanup := setupMCPTest(t)
	defer cleanup()

	text, isErr := callToolArgs(t, srv, srv.handleProjectContextTool, ProjectContextArgs{Path: "/nonexistent/xyz"})
	if isErr {
		t.Fatalf("no-match should not be an error: %s", text)
	}
	if !strings.Contains(text, "No indexed documents") {
		t.Fatalf("expected guidance for no match:\n%s", text)
	}
}

func TestCortexMemoryBatchWritesAndReportsFailures(t *testing.T) {
	srv, st, cleanup := setupMCPTest(t)
	defer cleanup()

	text, isErr := callToolArgs(t, srv, srv.handleMemoryBatchTool, MemoryBatchArgs{
		Items: []MemoryBatchItem{
			{Content: "用户偏好深色主题，界面语言为中文。"},
			{Content: "项目使用 Go 1.25，构建命令为 make build。"},
			{Content: ""},
		},
	})
	if isErr {
		t.Fatalf("batch returned error: %s", text)
	}
	if !strings.Contains(text, "2 written") || !strings.Contains(text, "1 failed") {
		t.Fatalf("unexpected batch summary:\n%s", text)
	}

	docs, _ := st.GetDocumentsCount("")
	if docs < 2 {
		t.Fatalf("expected at least 2 memory docs persisted, got %d", docs)
	}
}

func TestCortexMemoryBatchRejectsEmpty(t *testing.T) {
	srv, _, cleanup := setupMCPTest(t)
	defer cleanup()

	_, isErr := callToolArgs(t, srv, srv.handleMemoryBatchTool, MemoryBatchArgs{})
	if !isErr {
		t.Fatal("empty items should be rejected")
	}
}

func TestCortexMemoryBatchRejectsTooMany(t *testing.T) {
	srv, _, cleanup := setupMCPTest(t)
	defer cleanup()

	items := make([]MemoryBatchItem, 51)
	for i := range items {
		items[i] = MemoryBatchItem{Content: "x"}
	}
	_, isErr := callToolArgs(t, srv, srv.handleMemoryBatchTool, MemoryBatchArgs{Items: items})
	if !isErr {
		t.Fatal("more than 50 items should be rejected")
	}
}

// TestMemoryWriteRefactorParity 确保抽取 writeMemoryEntry 后单条写入行为不变。
func TestMemoryWriteRefactorParity(t *testing.T) {
	srv, st, cleanup := setupMCPTest(t)
	defer cleanup()

	text, isErr := callToolArgs(t, srv, srv.handleMemoryWriteTool, MemoryWriteArgs{
		Content: "这是一条用于验证重构后行为的记忆内容，长度足够触发分块。",
		Tags:    []string{"test"},
		Source:  "unit-test",
	})
	if isErr {
		t.Fatalf("write error: %s", text)
	}
	if !strings.Contains(text, "Memory written successfully") {
		t.Fatalf("unexpected output: %s", text)
	}
	docs, _ := st.GetDocumentsCount("")
	if docs != 1 {
		t.Fatalf("expected 1 memory doc, got %d", docs)
	}
}
