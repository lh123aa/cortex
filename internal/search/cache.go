package search

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lh123aa/cortex/internal/models"
	"github.com/patrickmn/go-cache"
)

type SearchCache struct {
	memoryCache *cache.Cache
}

func NewSearchCache() *SearchCache {
	// PRD 要求 300s TTL，我们在这里设定为 5 分钟，每 10 分钟执行一次淘汰清理
	return &SearchCache{
		memoryCache: cache.New(5*time.Minute, 10*time.Minute),
	}
}

// Get 从内存缓存取用 search response 避免暴刷 LLM/SQLite
func (c *SearchCache) Get(query string, opts models.SearchOptions) ([]*models.SearchResult, bool) {
	key := c.generateKey(query, opts)
	if val, found := c.memoryCache.Get(key); found {
		return val.([]*models.SearchResult), true
	}
	return nil, false
}

func (c *SearchCache) Set(query string, opts models.SearchOptions, results []*models.SearchResult) {
	key := c.generateKey(query, opts)
	c.memoryCache.Set(key, results, cache.DefaultExpiration)
}

// InvalidateAll 粗暴缓存清空（遇到文档发生增量更替可调用）
func (c *SearchCache) InvalidateAll() {
	c.memoryCache.Flush()
}

func (c *SearchCache) generateKey(query string, opts models.SearchOptions) string {
	data := fmt.Sprintf("q=%s_top=%d_mode=%s", query, opts.TopK, opts.Mode)
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

// SearchCacheWrapper 是为了在 HybridSearchEngine 上加了一层装饰器
func (s *HybridSearchEngine) SearchWithCache(cacheLayer *SearchCache, ctx context.Context, query string, opts models.SearchOptions) ([]*models.SearchResult, error) {
	if cacheLayer != nil {
		if cached, ok := cacheLayer.Get(query, opts); ok {
			return cached, nil
		}
	}
	res, err := s.Search(ctx, query, opts)
	if cacheLayer != nil && err == nil {
		cacheLayer.Set(query, opts, res)
	}
	return res, err
}
