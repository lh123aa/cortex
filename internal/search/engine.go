package search

import (
	"context"
	"sort"
	"time"

	"github.com/lh123aa/cortex/internal/embedding"
	"github.com/lh123aa/cortex/internal/metrics"
	"github.com/lh123aa/cortex/internal/models"
	"github.com/lh123aa/cortex/internal/storage"
)

type HybridSearchEngine struct {
	storage   storage.Storage
	embedding embedding.EmbeddingProvider
	useCache  bool
	cacheTTL  time.Duration
	reranker  Reranker // 可选的重排序器
}

// NewHybridSearchEngine 初始化搜索引擎
func NewHybridSearchEngine(s storage.Storage, emb embedding.EmbeddingProvider) *HybridSearchEngine {
	return &HybridSearchEngine{
		storage:   s,
		embedding: emb,
		useCache:  true, // 默认启用缓存
		cacheTTL:  5 * time.Minute,
	}
}

// SetReranker 设置重排序器
func (s *HybridSearchEngine) SetReranker(r Reranker) {
	s.reranker = r
}

// SetCacheTTL 设置缓存 TTL
func (s *HybridSearchEngine) SetCacheTTL(ttl time.Duration) {
	s.cacheTTL = ttl
}

// DisableCache 禁用缓存
func (s *HybridSearchEngine) DisableCache() {
	s.useCache = false
}

// Search 执行整体搜索与融合逻辑
// v2.2: 添加用户隔离支持
func (s *HybridSearchEngine) Search(ctx context.Context, query string, opts models.SearchOptions) ([]*models.SearchResult, error) {
	start := time.Now()
	metrics.SearchTotal.Inc()
	metrics.SearchByMode.WithLabelValues(opts.Mode).Inc()

	userID := opts.UserID

	// 1. 尝试从缓存获取（用户隔离缓存）
	if s.useCache {
		if cached, ok := s.storage.GetCachedSearch(query, userID, opts.Mode, opts.TopK); ok {
			metrics.SearchCacheHits.Inc()
			metrics.SearchDuration.Observe(time.Since(start).Seconds())
			metrics.SearchResultsReturned.Observe(float64(len(cached)))
			return cached, nil
		}
	}
	metrics.SearchCacheMisses.Inc()

	// 2. 执行实际搜索
	var vectorResults []*models.SearchResult
	var ftsResults []*models.SearchResult

	// 3. 如果有向量模型且非纯粹FTS，执行向量召回
	if s.embedding != nil && opts.Mode != "fts" {
		vecStart := time.Now()
		qVec, err := s.embedding.Embed(query)
		if err == nil && len(qVec) > 0 {
			metrics.SearchLatency.Observe(time.Since(vecStart).Seconds())
			vRes, err := s.storage.VectorSearch(qVec, userID, opts.TopK*2)
			if err == nil {
				vectorResults = vRes
			}
		}
	}

	// 4. 如果非纯粹Vector，执行FTS基于BM25召回
	if opts.Mode != "vector" {
		fRes, err := s.storage.FTSSearch(query, userID, opts.TopK*2)
		if err == nil {
			ftsResults = fRes
		}
	}

	// 5. 执行融合
	var finalResults []*models.SearchResult
	if opts.Mode == "vector" {
		finalResults = s.normalizeScores(vectorResults)
	} else if opts.Mode == "fts" {
		finalResults = s.normalizeScores(ftsResults)
	} else { // hybrid
		finalResults = s.rrfMerge(vectorResults, ftsResults)
	}

	// 6. TopK 截断
	if len(finalResults) > opts.TopK {
		finalResults = finalResults[:opts.TopK]
	}

	// 7. 可选: 重排序
	if s.reranker != nil && len(finalResults) > 0 {
		reranked, err := s.reranker.Rerank(ctx, query, finalResults, opts.TopK)
		if err == nil {
			finalResults = reranked
		}
	}

	// 8. Rank
	for i := range finalResults {
		finalResults[i].Rank = i + 1
	}

	// 9. 写入缓存（用户隔离缓存）
	if s.useCache && len(finalResults) > 0 {
		s.storage.SetCachedSearch(query, userID, opts.Mode, opts.TopK, finalResults, s.cacheTTL)
	}

	// 10. 记录指标
	metrics.SearchDuration.Observe(time.Since(start).Seconds())
	metrics.SearchResultsReturned.Observe(float64(len(finalResults)))

	return finalResults, nil
}

// InvalidateCache 使缓存失效（文档更新时调用）
func (s *HybridSearchEngine) InvalidateCache() {
	s.storage.InvalidateSearchCache()
}

// InvalidateUserCache 使某个用户的缓存失效
func (s *HybridSearchEngine) InvalidateUserCache(userID string) {
	s.storage.InvalidateUserSearchCache(userID)
}

// rrfMerge 倒数排名融合算法实现 (Reciprocal Rank Fusion)
func (s *HybridSearchEngine) rrfMerge(vr []*models.SearchResult, fr []*models.SearchResult) []*models.SearchResult {
	k := 60
	scoresMap := make(map[string]float64)
	resultsMap := make(map[string]*models.SearchResult)

	for rank, r := range vr {
		id := r.Chunk.ID
		scoresMap[id] += 1.0 / float64(k+rank+1)
		resultsMap[id] = r
	}

	for rank, r := range fr {
		id := r.Chunk.ID
		scoresMap[id] += 1.0 / float64(k+rank+1)
		resultsMap[id] = r
	}

	merged := make([]*models.SearchResult, 0, len(resultsMap))
	for id, score := range scoresMap {
		r := resultsMap[id]
		r.Score = score
		merged = append(merged, r)
	}

	// 按照 RRF 累加得分倒序
	return s.normalizeScores(merged)
}

func (s *HybridSearchEngine) normalizeScores(res []*models.SearchResult) []*models.SearchResult {
	maxScore := 0.0
	for _, r := range res {
		if r.Score > maxScore {
			maxScore = r.Score
		}
	}
	if maxScore > 0 {
		for _, r := range res {
			r.Score = r.Score / maxScore
		}
	}

	// v1.1 修复 O(n^2) 冒泡排序，改用标准库原生快排
	sort.Slice(res, func(i, j int) bool {
		return res[i].Score > res[j].Score
	})
	return res
}

// GetStats 返回搜索引擎统计信息（用户隔离）
func (s *HybridSearchEngine) GetStats(userID string) (docs, chunks, vectors int) {
	docs, _ = s.storage.GetDocumentsCount(userID)
	chunks, _ = s.storage.GetChunksCount(userID)
	vectors, _ = s.storage.GetVectorsCount(userID)
	return
}