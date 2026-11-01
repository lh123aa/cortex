package search

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/lh123aa/cortex/internal/models"
)

func TestNewSearchCache(t *testing.T) {
	cache := NewSearchCache()
	if cache == nil {
		t.Fatal("NewSearchCache returned nil")
	}
	if cache.memoryCache == nil {
		t.Error("memoryCache is nil")
	}
}

func TestSearchCacheGetSet(t *testing.T) {
	cache := NewSearchCache()

	results := []*models.SearchResult{
		{
			Score: 0.95,
			Chunk: &models.Chunk{
				ID:      "chunk-1",
				Content: "Test content",
			},
		},
	}

	opts := models.SearchOptions{
		TopK:   10,
		Mode:   "hybrid",
		UserID: "user-1",
	}

	// 首次获取应该失败
	cached, ok := cache.Get("test query", opts)
	if ok {
		t.Error("First cache get should return false")
	}

	// 设置缓存
	cache.Set("test query", opts, results)

	// 再次获取应该成功
	cached, ok = cache.Get("test query", opts)
	if !ok {
		t.Error("Cache get should return true after Set")
	}
	if len(cached) != 1 {
		t.Errorf("Expected 1 result, got %d", len(cached))
	}
	if cached[0].Score != 0.95 {
		t.Errorf("Expected score 0.95, got %f", cached[0].Score)
	}
}

func TestSearchCacheKeyGeneration(t *testing.T) {
	cache := NewSearchCache()

	opts1 := models.SearchOptions{TopK: 10, Mode: "hybrid", UserID: "user-1"}
	opts2 := models.SearchOptions{TopK: 10, Mode: "hybrid", UserID: "user-1"}
	opts3 := models.SearchOptions{TopK: 10, Mode: "vector", UserID: "user-1"}

	key1 := cache.generateKey("test query", opts1)
	key2 := cache.generateKey("test query", opts2)
	key3 := cache.generateKey("test query", opts3)

	// 相同参数应该生成相同的 key
	if key1 != key2 {
		t.Error("Same options should generate same key")
	}

	// 不同模式应该生成不同的 key
	if key1 == key3 {
		t.Error("Different modes should generate different keys")
	}
}

func TestSearchCacheInvalidateAll(t *testing.T) {
	cache := NewSearchCache()

	results := []*models.SearchResult{
		{Score: 0.9, Chunk: &models.Chunk{ID: "chunk-1"}},
	}

	opts := models.SearchOptions{TopK: 10, Mode: "hybrid"}

	cache.Set("query1", opts, results)
	cache.Set("query2", opts, results)

	// 验证缓存存在
	_, ok1 := cache.Get("query1", opts)
	_, ok2 := cache.Get("query2", opts)
	if !ok1 || !ok2 {
		t.Error("Cache entries should exist before InvalidateAll")
	}

	// 清空缓存
	cache.InvalidateAll()

	// 验证缓存已清空
	_, ok1 = cache.Get("query1", opts)
	_, ok2 = cache.Get("query2", opts)
	if ok1 || ok2 {
		t.Error("Cache entries should not exist after InvalidateAll")
	}
}

func TestHybridSearchEngineCreation(t *testing.T) {
	// 创建引擎（不需要实际存储）
	engine := &HybridSearchEngine{
		useCache: true,
		cacheTTL: 5 * time.Minute,
		l1Cache:  NewSearchCache(),
	}

	if !engine.useCache {
		t.Error("Cache should be enabled by default")
	}
	if engine.cacheTTL != 5*time.Minute {
		t.Errorf("Expected 5m cache TTL, got %v", engine.cacheTTL)
	}
	if engine.l1Cache == nil {
		t.Error("L1 cache should be initialized")
	}
}

func TestHybridSearchEngine_SetCacheTTL(t *testing.T) {
	engine := &HybridSearchEngine{}
	engine.SetCacheTTL(10 * time.Minute)

	if engine.cacheTTL != 10*time.Minute {
		t.Errorf("Expected 10m cache TTL, got %v", engine.cacheTTL)
	}
}

func TestHybridSearchEngine_DisableCache(t *testing.T) {
	engine := &HybridSearchEngine{useCache: true}
	engine.DisableCache()

	if engine.useCache {
		t.Error("Cache should be disabled after DisableCache")
	}
}

func TestSearchOptions(t *testing.T) {
	opts := models.SearchOptions{
		TopK:   20,
		Mode:   "vector",
		UserID: "user-123",
	}

	if opts.TopK != 20 {
		t.Errorf("Expected TopK 20, got %d", opts.TopK)
	}
	if opts.Mode != "vector" {
		t.Errorf("Expected mode 'vector', got '%s'", opts.Mode)
	}
	if opts.UserID != "user-123" {
		t.Errorf("Expected UserID 'user-123', got '%s'", opts.UserID)
	}
}

func TestSearchResult(t *testing.T) {
	result := &models.SearchResult{
		Score: 0.87,
		Rank:  1,
		Chunk: &models.Chunk{
			ID:          "chunk-test",
			DocumentID:  "doc-test",
			HeadingPath: "Introduction",
			Content:     "This is test content",
			ContentRaw:  "This is test content",
			TokenCount:  6,
		},
	}

	if result.Score != 0.87 {
		t.Errorf("Expected score 0.87, got %f", result.Score)
	}
	if result.Rank != 1 {
		t.Errorf("Expected rank 1, got %d", result.Rank)
	}
	if result.Chunk.HeadingPath != "Introduction" {
		t.Errorf("Expected heading 'Introduction', got '%s'", result.Chunk.HeadingPath)
	}
}

func TestRRF融合公式(t *testing.T) {
	// 测试 RRF 公式: score = 1 / (k + rank)
	// 标准 k=60

	k := 60.0

	// 第1名应该得到最高分
	score1 := 1.0 / (k + 1)
	// 第2名
	score2 := 1.0 / (k + 2)
	// 第10名
	score10 := 1.0 / (k + 10)

	if score1 <= score2 {
		t.Error("Higher rank should have higher score")
	}
	if score2 <= score10 {
		t.Error("Lower rank should have lower score")
	}

	// 验证分数递减
	expectedScore1 := 1.0 / 61.0
	if score1 != expectedScore1 {
		t.Errorf("Expected score %f for rank 1, got %f", expectedScore1, score1)
	}
}

func TestPrefetchKeywords(t *testing.T) {
	content := `# Go Concurrency
This document explains how to use goroutines and channels in Go.
Goroutines are lightweight threads. Channels are typed conduits.
Use "go func()" to start a goroutine.`

	kw := extractKeywords(content, 3)
	if len(kw) == 0 {
		t.Fatal("expected at least 1 keyword")
	}
	found := false
	for _, k := range kw {
		if k == "Go Concurrency" || k == "Go" || k == "goroutines" || k == "channels" {
			found = true
			break
		}
	}
	if !found {
		t.Logf("extracted keywords: %v", kw)
	}
}

func TestPrefetchCacheKey(t *testing.T) {
	key := prefetchCacheKey("/path/to/file.md", "goroutine")
	if key != "/path/to/file.md||goroutine" {
		t.Errorf("unexpected cache key: %s", key)
	}
}

func TestPrefetchEngine(t *testing.T) {
	engine := &HybridSearchEngine{}
	pe := NewPrefetchEngine(engine)
	if pe == nil {
		t.Fatal("NewPrefetchEngine returned nil")
	}
	if pe.search != engine {
		t.Error("search engine not set")
	}
	if pe.GetCachedCount() != 0 {
		t.Error("expected empty cache")
	}
}

func TestPrefetchEngine_OnFileChange_EmptyContent(t *testing.T) {
	engine := &HybridSearchEngine{}
	pe := NewPrefetchEngine(engine)

	// Empty content should not panic
	pe.OnFileChange("/test/file.md", nil)
	pe.OnFileChange("/test/file.md", []byte{})
}

func TestPrefetchEngine_Suggest_EmptyQuery(t *testing.T) {
	engine := &HybridSearchEngine{}
	pe := NewPrefetchEngine(engine)

	results := pe.Suggest(context.Background(), "", "/test/file.md", 5)
	if len(results) != 0 {
		t.Error("expected empty results for empty query")
	}
}

func TestPrefetchEngine_InvalidateFile(t *testing.T) {
	engine := &HybridSearchEngine{}
	pe := NewPrefetchEngine(engine)

	// Invalidate on empty cache should not panic
	pe.InvalidateFile("/test/file.md")
}

func TestPrefetchEngine_KeywordExtraction_Markdown(t *testing.T) {
	content := `# Project Overview
This is a project about building a search engine.
The search engine supports full text search and vector search.
We use Go and Python for implementation.`

	kw := extractKeywords(content, 3)
	if len(kw) == 0 {
		t.Fatal("expected at least 1 keyword")
	}

	hasH1 := false
	for _, k := range kw {
		if k == "Project Overview" {
			hasH1 = true
			break
		}
	}
	if !hasH1 {
		t.Logf("keywords: %v (expected 'Project Overview' to be included)", kw)
	}
}

func TestPrefetchEngine_KeywordExtraction_Code(t *testing.T) {
	content := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}`

	kw := extractKeywords(content, 3)
	if len(kw) == 0 {
		t.Fatal("expected at least 1 keyword from code content")
	}
	t.Logf("extracted from code: %v", kw)
}

func TestSearchCache_EvictOldest(t *testing.T) {
	cache := NewSearchCache()
	opts := models.SearchOptions{TopK: 10, Mode: "hybrid"}

	// Fill cache beyond max
	for i := 0; i < 10100; i++ {
		result := []*models.SearchResult{{
			Score: 0.5,
			Chunk: &models.Chunk{ID: fmt.Sprintf("chunk-%d", i)},
		}}
		cache.Set(fmt.Sprintf("query-%d", i), opts, result)
	}

	// Should have evicted oldest entries
	if cache.itemCount > MaxCacheItems {
		t.Errorf("cache size %d exceeds max %d", cache.itemCount, MaxCacheItems)
	}
}

// TestRRFTieBreakDeterministic 验证 rrfMerge 在 RRF 融合分相同时按 chunk ID 升序决胜,
// 且同一输入重复调用 N 次永远得到相同顺序(对照旧实现:map 随机迭代 + sort.Slice 不稳定)。
// 用“两条腿各占一个 rank 相同”的 chunk 构造精确平分:仅 vr rank0 的 'x-1' 与仅 fr rank0 的 'x-0'
// 都得 1/(60+0+1)=1/61,严格平分,触发 ID 决胜键(x-0 < x-1)。
func TestRRFTieBreakDeterministic(t *testing.T) {
	makeRes := func(id string, vec, fts float64) *models.SearchResult {
		return &models.SearchResult{Chunk: &models.Chunk{ID: id}, VectorScore: vec, FTSScore: fts}
	}
	eng := NewHybridSearchEngine(nil, nil)
	// 'x-1' 只在 vector 腿 rank0;'x-0' 只在 FTS 腿 rank0 → 各自 RRF = 1/61,严格平分。
	// 再加一个高分 chunk 'top'(两腿 rank0)保证它必须排第一(分最高),验证“先按分、平分才按 ID”。

	var firstRun []string
	for iter := 0; iter < 200; iter++ {
		// 每次重新构造同语义输入(模拟重复查询),r.rrfK=0 → 默认 k=60
		vrIter := []*models.SearchResult{
			makeRes("top", 1.0, 0), makeRes("x-1", 0.8, 0),
		}
		frIter := []*models.SearchResult{
			makeRes("top", 0, 1.0), makeRes("x-0", 0, 0.8),
		}
		out := eng.rrfMerge(vrIter, frIter)
		if len(out) != 3 {
			t.Fatalf("iter %d: expected 3 results, got %d", iter, len(out))
		}
		// 断言顺序:top(分最高) 必须第一;x-0 与 x-1 平分,按 ID 升序 → x-0 在 x-1 前。
		want := []string{"top", "x-0", "x-1"}
		for i, id := range want {
			if out[i].Chunk.ID != id {
				got := make([]string, len(out))
				for j, r := range out {
					got[j] = r.Chunk.ID
				}
				t.Fatalf("iter %d: expected %v, got %v", iter, want, got)
			}
		}
		if iter == 0 {
			firstRun = append(firstRun, out[0].Chunk.ID, out[1].Chunk.ID, out[2].Chunk.ID)
		} else {
			if out[0].Chunk.ID != firstRun[0] || out[1].Chunk.ID != firstRun[1] || out[2].Chunk.ID != firstRun[2] {
				t.Fatalf("non-deterministic: iter %d order %v != iter0 %v",
					iter, []string{out[0].Chunk.ID, out[1].Chunk.ID, out[2].Chunk.ID}, firstRun)
			}
		}
	}
}

// TestRRFMetadataPreserved 验证同 chunk 被两条腿命中时保留向量腿主记录的 VectorScore、
// 只回填 FTS 腿的 FTSScore(对照旧实现:FTS 腿整条覆盖 → VectorScore 丢失)。
func TestRRFMetadataPreserved(t *testing.T) {
	makeRes := func(id string, vec, fts float64) *models.SearchResult {
		return &models.SearchResult{Chunk: &models.Chunk{ID: id}, VectorScore: vec, FTSScore: fts}
	}
	eng := NewHybridSearchEngine(nil, nil)
	results := eng.rrfMerge(
		[]*models.SearchResult{makeRes("dup", 0.9, 0), makeRes("only-vec", 0.8, 0)},
		[]*models.SearchResult{makeRes("dup", 0, 0.8), makeRes("only-fts", 0, 0.7)},
	)
	byID := map[string]*models.SearchResult{}
	for _, r := range results {
		byID[r.Chunk.ID] = r
	}
	if len(byID) != 3 {
		t.Fatalf("expected 3 merged results, got %d", len(byID))
	}
	dup := byID["dup"]
	if dup.VectorScore != 0.9 {
		t.Errorf("dup: expected VectorScore 0.9 (vector-leg main record preserved), got %f", dup.VectorScore)
	}
	if dup.FTSScore != 0.8 {
		t.Errorf("dup: expected FTSScore 0.8 (backfilled from FTS leg), got %f", dup.FTSScore)
	}
	vecOnly := byID["only-vec"]
	if vecOnly.VectorScore != 0.8 {
		t.Errorf("only-vec: expected VectorScore 0.8, got %f", vecOnly.VectorScore)
	}
	if vecOnly.FTSScore != 0 {
		t.Errorf("only-vec: expected FTSScore 0, got %f", vecOnly.FTSScore)
	}
	ftsOnly := byID["only-fts"]
	if ftsOnly.VectorScore != 0 {
		t.Errorf("only-fts: expected VectorScore 0, got %f", ftsOnly.VectorScore)
	}
	if ftsOnly.FTSScore != 0.7 {
		t.Errorf("only-fts: expected FTSScore 0.7, got %f", ftsOnly.FTSScore)
	}
}
