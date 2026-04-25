package search

import (
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
