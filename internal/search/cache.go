package search

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/lh123aa/cortex/internal/models"
	"github.com/patrickmn/go-cache"
)

const (
	// MaxCacheItems 限制最大缓存条目数，防止内存耗尽
	MaxCacheItems = 10000
)

type SearchCache struct {
	memoryCache *cache.Cache
	mu          sync.Mutex
	itemCount   int
}

// NewSearchCache 创建带容量限制的搜索缓存
// PRD 要求 300s TTL，设定为 5 分钟，每 10 分钟执行一次淘汰清理
func NewSearchCache() *SearchCache {
	return &SearchCache{
		memoryCache: cache.New(5*time.Minute, 10*time.Minute),
		itemCount:   0,
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
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.generateKey(query, opts)

	// 检查是否已存在，避免重复计数
	if _, found := c.memoryCache.Get(key); !found {
		c.itemCount++
		// 如果超过最大条目数，清除最旧的条目
		if c.itemCount > MaxCacheItems {
			c.evictOldest(1000) // 清除1000个最旧的条目
		}
	}

	c.memoryCache.Set(key, results, cache.DefaultExpiration)
}

// evictOldest 清除最旧的 N 个缓存条目
func (c *SearchCache) evictOldest(count int) {
	items := c.memoryCache.Items()
	evicted := 0
	for key := range items {
		if evicted >= count {
			break
		}
		c.memoryCache.Delete(key)
		c.itemCount--
		evicted++
	}
}

// InvalidateAll 粗暴缓存清空（遇到文档发生增量更替可调用）
func (c *SearchCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.memoryCache.Flush()
	c.itemCount = 0
}

// ItemCount 返回当前缓存条目数（用于监控）
func (c *SearchCache) ItemCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.itemCount
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
