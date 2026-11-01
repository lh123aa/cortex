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
	rrfK      int             // RRF 常数（默认 60）
	reranker  Reranker        // 可选的重排序器
	l1Cache   *SearchCache    // L1 内存缓存
	prefetch  *PrefetchEngine // 预联想引擎，nil 时跳过

	// L2（SQLite）缓存写入队列。
	//
	// 读路径上不应该有同步磁盘写：早期实现每次搜索都 INSERT 一次缓存，
	// 把 p95 拖到 10–60ms（还要和索引写抢锁）。现在改为有界异步写入：
	// 队列满时直接丢弃（缓存本来就是可选的加速层）。
	l2Queue chan l2Write
}

type l2Write struct {
	query   string
	userID  string
	mode    string
	topK    int
	results []*models.SearchResult
}

// l2QueueSize 是异步缓存写入的缓冲上限。
const l2QueueSize = 256

// NewHybridSearchEngine 初始化搜索引擎
func NewHybridSearchEngine(s storage.Storage, emb embedding.EmbeddingProvider) *HybridSearchEngine {
	e := &HybridSearchEngine{
		storage:   s,
		embedding: emb,
		useCache:  true, // 默认启用缓存
		cacheTTL:  5 * time.Minute,
		l1Cache:   NewSearchCache(), // 初始化 L1 缓存
		l2Queue:   make(chan l2Write, l2QueueSize),
	}
	go e.l2WriterLoop()
	return e
}

// l2WriterLoop 串行消费缓存写入任务（单 goroutine，避免并发写 SQLite）。
func (s *HybridSearchEngine) l2WriterLoop() {
	for w := range s.l2Queue {
		_ = s.storage.SetCachedSearch(w.query, w.userID, w.mode, w.topK, w.results, s.cacheTTL)
	}
}

// enqueueL2 非阻塞投递缓存写入；队列满时丢弃并计数。
func (s *HybridSearchEngine) enqueueL2(w l2Write) {
	if s.l2Queue == nil {
		return
	}
	select {
	case s.l2Queue <- w:
	default:
		metrics.SearchCacheDropped.Inc()
	}
}

// SetRRFK 设置 RRF 常数（默认 60，推荐范围 30-100）
// k 值越小，排名靠前的结果权重越高
func (s *HybridSearchEngine) SetRRFK(k int) {
	if k < 1 {
		k = 60
	}
	s.rrfK = k
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

// SetPrefetchEngine 设置预联想引擎
func (s *HybridSearchEngine) SetPrefetchEngine(pe *PrefetchEngine) {
	s.prefetch = pe
}

// GetPrefetchEngine 获取预联想引擎
func (s *HybridSearchEngine) GetPrefetchEngine() *PrefetchEngine {
	return s.prefetch
}

// GetL1Cache 获取 L1 缓存实例
func (s *HybridSearchEngine) GetL1Cache() *SearchCache {
	return s.l1Cache
}

// Search 执行整体搜索与融合逻辑
// v2.2: 添加用户隔离支持
// v2.3: 添加 L1 + L2 两级缓存
func (s *HybridSearchEngine) Search(ctx context.Context, query string, opts models.SearchOptions) ([]*models.SearchResult, error) {
	start := time.Now()
	metrics.SearchTotal.Inc()
	metrics.SearchByMode.WithLabelValues(opts.Mode).Inc()

	userID := opts.UserID

	// 1. 尝试从 L1 内存缓存获取（最快）
	if s.useCache && s.l1Cache != nil {
		if cached, ok := s.l1Cache.Get(query, opts); ok {
			metrics.SearchCacheHits.Inc()
			metrics.SearchDuration.Observe(time.Since(start).Seconds())
			metrics.SearchResultsReturned.Observe(float64(len(cached)))
			return cached, nil
		}
	}

	// 2. 尝试从 L2 SQLite 缓存获取（用户隔离）
	if s.useCache {
		if cached, ok := s.storage.GetCachedSearch(query, userID, opts.Mode, opts.TopK); ok {
			metrics.SearchCacheHits.Inc()
			// 写入 L1 缓存供下次快速访问
			if s.l1Cache != nil {
				s.l1Cache.Set(query, opts, cached)
			}
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

	// 6. 分页 + TopK 截断
	if opts.Offset > 0 && opts.Offset < len(finalResults) {
		finalResults = finalResults[opts.Offset:]
	}
	if len(finalResults) > opts.TopK {
		finalResults = finalResults[:opts.TopK]
	}

	// 6.5 补充文件路径信息（用于文件类型权重）
	// 检索层已经把 d.path 带出来了，这里只兜底补空缺，
	// 避免早期实现「每条结果查一次 documents」的 N+1。
	for _, r := range finalResults {
		if r == nil || r.Chunk == nil || r.FilePath != "" || r.Chunk.DocumentID == "" {
			continue
		}
		if doc, err := s.storage.GetDocumentByID(r.Chunk.DocumentID, userID); err == nil && doc != nil {
			r.FilePath = doc.Path
		}
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

	// 9. 写入两级缓存
	if s.useCache && len(finalResults) > 0 {
		// L1 内存缓存：同步写（纯内存，微秒级）
		if s.l1Cache != nil {
			s.l1Cache.Set(query, opts, finalResults)
		}
		// L2 SQLite 缓存：异步写，避免读路径上做磁盘写（p95 的主要来源）
		s.enqueueL2(l2Write{
			query:   query,
			userID:  userID,
			mode:    opts.Mode,
			topK:    opts.TopK,
			results: finalResults,
		})
	}

	// 10. 记录指标
	metrics.SearchDuration.Observe(time.Since(start).Seconds())
	metrics.SearchResultsReturned.Observe(float64(len(finalResults)))

	return finalResults, nil
}

// InvalidateCache 使缓存失效（文档更新时调用）
func (s *HybridSearchEngine) InvalidateCache() {
	// 清空 L2 缓存
	s.storage.InvalidateSearchCache()
	// 清空 L1 缓存
	if s.l1Cache != nil {
		s.l1Cache.InvalidateAll()
	}
}

// InvalidateUserCache 使某个用户的缓存失效
func (s *HybridSearchEngine) InvalidateUserCache(userID string) {
	s.storage.InvalidateUserSearchCache(userID)
}

// rrfMerge 倒数排名融合算法实现 (Reciprocal Rank Fusion)
//
// 确定性保证(参考 XERJ fuse_finish 的 (score, seq_no, id) 决胜键):
//   - 同一 chunk 在两条腿都命中时,先写入(向量腿)的元数据(VectorScore)被保留,
//     FTS 腿的 FTSScore 只回填到缺失的字段,绝不整体覆盖记录。
//   - 排序使用 (融合分降序, chunk ID 升序) 的确定性决胜键,不依赖 Go map
//     迭代顺序或 sort.Slice 的不稳定性 —— 同一输入永远得到同一 top-N 集合。
func (s *HybridSearchEngine) rrfMerge(vr []*models.SearchResult, fr []*models.SearchResult) []*models.SearchResult {
	k := s.rrfK
	if k <= 0 {
		k = 60
	}
	scoresMap := make(map[string]float64)
	resultsMap := make(map[string]*models.SearchResult)

	// 向量腿先写入(带 VectorScore),作为“主记录”。
	for rank, r := range vr {
		id := r.Chunk.ID
		scoresMap[id] += 1.0 / float64(k+rank+1)
		if _, ok := resultsMap[id]; !ok {
			resultsMap[id] = r
		}
	}
	// FTS 腿:若 id 已存在则只累加分数并回填缺失的 FTSScore,不替换主记录。
	for rank, r := range fr {
		id := r.Chunk.ID
		scoresMap[id] += 1.0 / float64(k+rank+1)
		if existing, ok := resultsMap[id]; ok {
			if existing.FTSScore == 0 {
				existing.FTSScore = r.FTSScore
			}
		} else {
			resultsMap[id] = r
		}
	}

	merged := make([]*models.SearchResult, 0, len(resultsMap))
	for id, score := range scoresMap {
		r := resultsMap[id]
		r.Score = score
		merged = append(merged, r)
	}

	// 确定性排序:融合分降序;平分 chunk ID 升序兜底(参考 XERJ fuse_finish)。
	sort.Slice(merged, func(i, j int) bool {
		if merged[i].Score != merged[j].Score {
			return merged[i].Score > merged[j].Score
		}
		return merged[i].Chunk.ID < merged[j].Chunk.ID
	})

	// 归一化到 [0,1](保留上面的确定性顺序,不再重排)
	return s.normalizeInPlace(merged)
}

// normalizeInPlace 把 Score 归一化到 [0,1],不改变元素相对顺序。
// 与 normalizeScores 的区别:后者仍用不稳定排序,仅用于单腿模式(vector/fts)。
func (s *HybridSearchEngine) normalizeInPlace(res []*models.SearchResult) []*models.SearchResult {
	if len(res) == 0 {
		return res
	}
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
	return res
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
