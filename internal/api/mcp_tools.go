package api

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lh123aa/cortex/internal/models"
	"github.com/lh123aa/cortex/internal/snippet"
	"github.com/lh123aa/cortex/internal/storage"
	"github.com/lh123aa/cortex/internal/version"
	"github.com/lh123aa/cortex/pkg/license"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// 本文件扩展 MCP 工具集（8 → 14），新增：
//   cortex_map              知识库数据地图（有哪些库/文件类型/规模）
//   cortex_stats            统计与套餐用量
//   cortex_plan             根据问题给出检索计划（先查什么、用什么模式）
//   cortex_project_context  给定项目路径，汇总相关文档与代码入口
//   cortex_code_search      代码符号检索（按语言/文件过滤，返回 file:line）
//   cortex_memory_batch     批量写入记忆

type MapArgs struct {
	Limit int `json:"limit,omitempty" jsonschema:"Max number of file types / paths to list (default 20)"`
}

type StatsArgs struct {
	Verbose bool `json:"verbose,omitempty" jsonschema:"Include per-extension breakdown"`
}

type PlanArgs struct {
	Question string `json:"question" jsonschema:"The user question or task to plan retrieval for"`
}

type ProjectContextArgs struct {
	Path  string `json:"path" jsonschema:"Project root path to scope context to (prefix match)"`
	Query string `json:"query,omitempty" jsonschema:"Optional focus query within the project"`
	TopK  int    `json:"top_k,omitempty" jsonschema:"Number of chunks to return (default 8)"`
}

type CodeSearchArgs struct {
	Query     string `json:"query" jsonschema:"Symbol name, function name or code keyword to find"`
	Language  string `json:"language,omitempty" jsonschema:"Filter by language/extension, e.g. go, py, ts, js"`
	PathLike  string `json:"path_like,omitempty" jsonschema:"Only match file paths containing this substring"`
	TopK      int    `json:"top_k,omitempty" jsonschema:"Number of results (default 10)"`
	WithLines bool   `json:"with_lines,omitempty" jsonschema:"Include line numbers when available"`
}

type MemoryBatchArgs struct {
	Items []MemoryBatchItem `json:"items" jsonschema:"Memory entries to write in one call (max 50)"`
}

type MemoryBatchItem struct {
	Content string   `json:"content" jsonschema:"Memory content"`
	Tags    []string `json:"tags,omitempty" jsonschema:"Optional tags"`
	Source  string   `json:"source,omitempty" jsonschema:"Optional source identifier"`
}

func (s *MCPServer) registerExtraTools() {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "cortex_map",
		Description: "Get a map of the local knowledge base: document count, file type breakdown and sample paths. Use this first when you do not know what is indexed.",
	}, s.handleMapTool)

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "cortex_stats",
		Description: "Return Cortex statistics: documents, chunks, vectors, storage usage, current plan and its limits.",
	}, s.handleStatsTool)

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "cortex_plan",
		Description: "Given a question or task, return a concrete retrieval plan (which queries to run, which mode, what to look for). Does not itself search.",
	}, s.handlePlanTool)

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "cortex_project_context",
		Description: "Collect context scoped to a project directory: matching documents, chunks and file entry points under that path.",
	}, s.handleProjectContextTool)

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "cortex_code_search",
		Description: "Search code with file:line results. Filters by language or path, useful for locating a function or symbol definition.",
	}, s.handleCodeSearchTool)

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "cortex_memory_batch",
		Description: "Write multiple memory entries in a single call (max 50), for storing a batch of facts at the end of a session.",
	}, s.handleMemoryBatchTool)

	s.toolCount += 6
}

func (s *MCPServer) handleMapTool(ctx context.Context, req *mcp.CallToolRequest, args MapArgs) (*mcp.CallToolResult, any, error) {
	limit := args.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}

	docs, _ := s.storage.GetDocumentsCount("")
	chunks, _ := s.storage.GetChunksCount("")
	vectors, _ := s.storage.GetVectorsCount("")

	// 取最近文档做类型与路径统计
	sample, err := s.storage.ListDocuments("", 2000, 0)
	if err != nil {
		return toolError("failed to list documents: " + err.Error()), nil, nil
	}

	typeCount := map[string]int{}
	dirCount := map[string]int{}
	for _, d := range sample {
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(d.Path), "."))
		if ext == "" {
			ext = "(noext)"
		}
		typeCount[ext]++
		dir := filepath.Dir(d.Path)
		if len(dir) > 60 {
			dir = dir[:57] + "..."
		}
		dirCount[dir]++
	}

	typePair := topPairs(typeCount, limit)
	dirPair := topPairs(dirCount, limit)

	var sb strings.Builder
	fmt.Fprintf(&sb, "🗺️  Cortex Knowledge Map\n\n")
	fmt.Fprintf(&sb, "Documents: %d\nChunks:    %d\nVectors:   %d\n", docs, chunks, vectors)
	if len(sample) < docs {
		fmt.Fprintf(&sb, "(type/path breakdown based on %d most recent documents)\n", len(sample))
	}
	sb.WriteString("\nFile types:\n")
	for _, p := range typePair {
		fmt.Fprintf(&sb, "  %-12s %d\n", p.Key, p.Count)
	}
	sb.WriteString("\nTop directories:\n")
	for _, p := range dirPair {
		fmt.Fprintf(&sb, "  %-60s %d\n", p.Key, p.Count)
	}
	sb.WriteString("\nTip: use cortex_search / cortex_code_search to retrieve content, cortex_plan to get a retrieval plan.")

	return textResult(sb.String()), nil, nil
}

func (s *MCPServer) handleStatsTool(ctx context.Context, req *mcp.CallToolRequest, args StatsArgs) (*mcp.CallToolResult, any, error) {
	docs, _ := s.storage.GetDocumentsCount("")
	chunks, _ := s.storage.GetChunksCount("")
	vectors, _ := s.storage.GetVectorsCount("")
	used, _ := s.storage.CalculateStorageUsed("")

	plan := license.PlanFree
	expiry := "—"
	if info, err := license.Current(); err == nil && info != nil {
		plan = info.Plan
		if !info.ExpiresAt.IsZero() {
			expiry = info.ExpiresAt.Format("2006-01-02")
		}
	}
	lim := license.Limits(plan)

	var sb strings.Builder
	fmt.Fprintf(&sb, "📊 Cortex Stats (%s)\n\n", versionLabel())
	fmt.Fprintf(&sb, "Documents: %d\nChunks:    %d\nVectors:   %d\n", docs, chunks, vectors)
	fmt.Fprintf(&sb, "Storage:   %s\n", humanBytes(used))
	fmt.Fprintf(&sb, "Plan:      %s (expires %s)\n", plan.String(), expiry)
	if lim.Documents < 0 {
		sb.WriteString("Doc limit: unlimited\nStorage limit: unlimited\n")
	} else {
		fmt.Fprintf(&sb, "Doc limit:     %d (%.1f%% used)\n", lim.Documents, pct(float64(docs), float64(lim.Documents)))
		fmt.Fprintf(&sb, "Storage limit: %s (%.1f%% used)\n", humanBytes(lim.StorageBytes), pct(float64(used), float64(lim.StorageBytes)))
	}
	if args.Verbose {
		if sample, err := s.storage.ListDocuments("", 2000, 0); err == nil {
			typeCount := map[string]int{}
			for _, d := range sample {
				ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(d.Path), "."))
				if ext == "" {
					ext = "(noext)"
				}
				typeCount[ext]++
			}
			sb.WriteString("\nBy type:\n")
			for _, p := range topPairs(typeCount, 20) {
				fmt.Fprintf(&sb, "  %-12s %d\n", p.Key, p.Count)
			}
		}
	}
	return textResult(sb.String()), nil, nil
}

func (s *MCPServer) handlePlanTool(ctx context.Context, req *mcp.CallToolRequest, args PlanArgs) (*mcp.CallToolResult, any, error) {
	q := strings.TrimSpace(args.Question)
	if q == "" {
		return toolError("question is required"), nil, nil
	}

	// 探测知识库里是否真的存在相关内容，让计划有依据而不是空话
	probe, _ := s.search.Search(ctx, q, models.SearchOptions{TopK: 3, Mode: "hybrid", UserID: s.userID})
	evidence := len(probe)

	var sb strings.Builder
	fmt.Fprintf(&sb, "🧭 Retrieval plan for: %s\n\n", q)
	if evidence > 0 {
		fmt.Fprintf(&sb, "Quick probe: %d candidate chunks already match.\n", evidence)
		sb.WriteString("First results:\n")
		for i, r := range probe {
			if r.Chunk == nil {
				continue
			}
			path := r.FilePath
			if path == "" {
				path = r.Chunk.HeadingPath
			}
			fmt.Fprintf(&sb, "  %d. [%.3f] %s\n", i+1, r.Score, snippet.Build(path, q, snippet.Options{Width: 100}))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("Quick probe: no direct match. The vocabulary may differ from the indexed text.\n\n")
	}

	sb.WriteString("Suggested steps:\n")
	sb.WriteString("  1. cortex_map — confirm what is indexed (types, directories).\n")
	fmt.Fprintf(&sb, "  2. cortex_search {query: %q} — hybrid retrieval, widen top_k to 20.\n", q)
	sb.WriteString("  3. If the question is code-shaped (function/symbol), use cortex_code_search with a language filter.\n")
	sb.WriteString("  4. If the question is about a known project folder, use cortex_project_context with that path.\n")
	sb.WriteString("  5. cortex_context — assemble a token-budgeted answer context from the best hits.\n")
	sb.WriteString("  6. If nothing relevant exists, say so instead of inventing an answer.\n")

	return textResult(sb.String()), nil, nil
}

func (s *MCPServer) handleProjectContextTool(ctx context.Context, req *mcp.CallToolRequest, args ProjectContextArgs) (*mcp.CallToolResult, any, error) {
	root := strings.TrimSpace(args.Path)
	if root == "" {
		return toolError("path is required"), nil, nil
	}
	topK := args.TopK
	if topK <= 0 {
		topK = 8
	}
	if topK > 50 {
		topK = 50
	}

	// 用 path 作为过滤词 + 查询词组合检索
	query := strings.TrimSpace(args.Query)
	if query == "" {
		query = filepath.Base(root)
	}

	opts := models.SearchOptions{TopK: topK * 3, Mode: "hybrid", UserID: s.userID}
	results, err := s.search.Search(ctx, query, opts)
	if err != nil {
		return toolError("search failed: " + err.Error()), nil, nil
	}

	normRoot := strings.ToLower(filepath.ToSlash(root))
	matched := make([]string, 0, topK)
	type fileHit struct {
		path  string
		score float64
		head  string
	}
	var hits []fileHit
	seen := map[string]bool{}
	for _, r := range results {
		if r.Chunk == nil {
			continue
		}
		p := r.FilePath
		if p == "" {
			continue
		}
		normP := strings.ToLower(filepath.ToSlash(p))
		if !strings.Contains(normP, normRoot) {
			continue
		}
		if seen[normP] {
			continue
		}
		seen[normP] = true
		hits = append(hits, fileHit{path: p, score: r.Score, head: r.Chunk.HeadingPath})
		if len(hits) >= topK {
			break
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "📁 Project context: %s\n", root)
	if query != "" {
		fmt.Fprintf(&sb, "Focus query: %s\n", query)
	}
	fmt.Fprintf(&sb, "Matched files: %d\n\n", len(hits))
	if len(hits) == 0 {
		sb.WriteString("No indexed documents matched this path.\n")
		sb.WriteString("Hints:\n  - Confirm the folder was indexed: cortex index <path>\n  - Check cortex_map for indexed directories.\n")
		return textResult(sb.String()), nil, nil
	}
	for i, h := range hits {
		fmt.Fprintf(&sb, "%d. [%.3f] %s\n", i+1, h.score, h.path)
		if h.head != "" {
			fmt.Fprintf(&sb, "   section: %s\n", h.head)
		}
		matched = append(matched, h.path)
	}
	sb.WriteString("\nNext: cortex_context with a token_budget to get a ready-to-use answer context.")
	return textResult(sb.String()), nil, nil
}

func (s *MCPServer) handleCodeSearchTool(ctx context.Context, req *mcp.CallToolRequest, args CodeSearchArgs) (*mcp.CallToolResult, any, error) {
	query := strings.TrimSpace(args.Query)
	if query == "" {
		return toolError("query is required"), nil, nil
	}
	topK := args.TopK
	if topK <= 0 {
		topK = 10
	}
	if topK > 100 {
		topK = 100
	}

	results, err := s.search.Search(ctx, query, models.SearchOptions{TopK: topK * 4, Mode: "hybrid", UserID: s.userID})
	if err != nil {
		return toolError("search failed: " + err.Error()), nil, nil
	}

	lang := normalizeLanguage(args.Language)
	pathLike := strings.ToLower(strings.TrimSpace(args.PathLike))

	var sb strings.Builder
	fmt.Fprintf(&sb, "🔎 Code search: %s\n", query)
	if lang != "" {
		fmt.Fprintf(&sb, "Language filter: %s\n", lang)
	}
	if pathLike != "" {
		fmt.Fprintf(&sb, "Path filter: %s\n", pathLike)
	}
	sb.WriteString("\n")

	// 1) 符号索引优先：按函数/类名直接给出 file:line 与签名。
	if store, ok := s.storage.(storage.SymbolStore); ok {
		if syms, err := store.SearchSymbols(query, lang, pathLike, topK); err == nil && len(syms) > 0 {
			fmt.Fprintf(&sb, "Definitions (%d):\n", len(syms))
			for i, sym := range syms {
				loc := sym.Path
				if args.WithLines && sym.Line > 0 {
					loc = fmt.Sprintf("%s:%d", sym.Path, sym.Line)
				}
				fmt.Fprintf(&sb, "%d. [%s] %s  %s\n", i+1, sym.Kind, sym.Name, loc)
				if sym.Signature != "" {
					fmt.Fprintf(&sb, "   %s\n", sym.Signature)
				}
			}
			sb.WriteString("\n")
		}
	}

	// 2) 文本命中作为补充（符号索引未覆盖时也能找到内容）
	type hit struct {
		path  string
		line  int
		head  string
		score float64
		body  string
	}
	var hits []hit
	for _, r := range results {
		if r.Chunk == nil {
			continue
		}
		p := r.FilePath
		if p == "" {
			continue
		}
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(p), "."))
		if lang != "" && ext != lang {
			continue
		}
		if pathLike != "" && !strings.Contains(strings.ToLower(filepath.ToSlash(p)), pathLike) {
			continue
		}
		hits = append(hits, hit{
			path:  p,
			line:  r.Chunk.LineStart,
			head:  r.Chunk.HeadingPath,
			score: r.Score,
			body:  snippet.Build(r.Chunk.DisplayText(), query, snippet.Options{Width: 200, PreserveNewlines: true}),
		})
		if len(hits) >= topK {
			break
		}
	}

	fmt.Fprintf(&sb, "Text hits: %d\n\n", len(hits))
	if len(hits) == 0 && sb.Len() < 200 {
		sb.WriteString("No code matches. Try cortex_search for prose, or drop the language/path filters.\n")
		return textResult(sb.String()), nil, nil
	}
	for i, h := range hits {
		loc := h.path
		if args.WithLines && h.line > 0 {
			loc = fmt.Sprintf("%s:%d", h.path, h.line)
		}
		fmt.Fprintf(&sb, "%d. [%.3f] %s\n", i+1, h.score, loc)
		if h.head != "" {
			fmt.Fprintf(&sb, "   section: %s\n", h.head)
		}
		if h.body != "" {
			for _, ln := range strings.Split(h.body, "\n") {
				fmt.Fprintf(&sb, "   | %s\n", ln)
			}
		}
		sb.WriteString("\n")
	}
	return textResult(sb.String()), nil, nil
}

func (s *MCPServer) handleMemoryBatchTool(ctx context.Context, req *mcp.CallToolRequest, args MemoryBatchArgs) (*mcp.CallToolResult, any, error) {
	if len(args.Items) == 0 {
		return toolError("items is required"), nil, nil
	}
	if len(args.Items) > 50 {
		return toolError("too many items (max 50 per call)"), nil, nil
	}

	var ok, failed int
	var errs []string
	for i, item := range args.Items {
		content := strings.TrimSpace(item.Content)
		if content == "" {
			failed++
			errs = append(errs, fmt.Sprintf("item %d: empty content", i+1))
			continue
		}
		if _, _, err := s.writeMemoryEntry(content, item.SummarySafe(), s.userID); err != nil {
			failed++
			errs = append(errs, fmt.Sprintf("item %d: %v", i+1, err))
			continue
		}
		ok++
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "🧠 Memory batch: %d written, %d failed\n", ok, failed)
	if len(errs) > 0 {
		sb.WriteString("\nErrors:\n")
		for _, e := range errs {
			fmt.Fprintf(&sb, "  - %s\n", e)
		}
	}
	return textResult(sb.String()), nil, nil
}

// SummarySafe 返回用作摘要的短文本（批量写入时自动生成）。
func (i MemoryBatchItem) SummarySafe() string {
	c := strings.TrimSpace(i.Content)
	r := []rune(c)
	if len(r) > 100 {
		return string(r[:100]) + "..."
	}
	return c
}

// ---- 小工具函数 ----

// versionLabel 返回带 v 前缀的版本标签，且不会出现 "vv4.0.0" 这种重复前缀。
// version.Version 在发布构建里本身已含 "v"（如 v4.0.0），本地开发构建则是 "dev"。
func versionLabel() string {
	v := strings.TrimSpace(version.Version)
	// 本地开发构建的占位值不加前缀，否则会变成 "vdev"
	if v == "" || v == "dev" || v == "unknown" {
		return "dev"
	}
	if strings.HasPrefix(v, "v") {
		return v
	}
	return "v" + v
}

// normalizeLanguage 把用户常用的语言别名归一化成索引里存的语言名。
// 例如用户写 "py"/".py"/"Python" 都应匹配到 "python"，
// 否则「按语言过滤」会静默返回空结果，看起来像索引没建好。
func normalizeLanguage(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.TrimPrefix(s, ".")
	aliases := map[string]string{
		"py": "python", "python3": "python", "pyi": "python",
		"js": "javascript", "jsx": "javascript", "mjs": "javascript", "node": "javascript",
		"ts": "typescript", "tsx": "typescript",
		"golang": "go",
		"rs":     "rust",
		"java":   "java",
	}
	if v, ok := aliases[s]; ok {
		return v
	}
	return s
}

// humanBytes 人类可读字节数。

type keyCount struct {
	Key   string
	Count int
}

func topPairs(m map[string]int, limit int) []keyCount {
	out := make([]keyCount, 0, len(m))
	for k, v := range m {
		out = append(out, keyCount{Key: k, Count: v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Key < out[j].Key
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func humanBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func pct(used, limit float64) float64 {
	if limit <= 0 {
		return 0
	}
	return used / limit * 100
}

// textResult 构造纯文本 MCP 结果。
func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

// jsonResult 构造 JSON 文本 MCP 结果。
func jsonResult(v any) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return textResult(string(b)), nil
}
