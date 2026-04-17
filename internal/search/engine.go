package search

import (
	"context"
	"sort"

	"github.com/lh123aa/cortex/internal/embedding"
	"github.com/lh123aa/cortex/internal/models"
	"github.com/lh123aa/cortex/internal/storage"
)

type HybridSearchEngine struct {
	storage   storage.Storage
	embedding embedding.EmbeddingProvider
}

// NewHybridSearchEngine 初始化搜索引擎
func NewHybridSearchEngine(s storage.Storage, emb embedding.EmbeddingProvider) *HybridSearchEngine {
	return &HybridSearchEngine{
		storage:   s,
		embedding: emb,
	}
}

// Search 执行整体搜索与融合逻辑
func (s *HybridSearchEngine) Search(ctx context.Context, query string, opts models.SearchOptions) ([]*models.SearchResult, error) {
	var vectorResults []*models.SearchResult
	var ftsResults []*models.SearchResult

	// 1. 如果有向量模型且非纯粹FTS，执行向量召回
	if s.embedding != nil && opts.Mode != "fts" {
		qVec, err := s.embedding.Embed(query)
		if err == nil && len(qVec) > 0 {
			vRes, err := s.storage.VectorSearch(qVec, opts.TopK*2)
			if err == nil {
				vectorResults = vRes
			}
		}
	}

	// 2. 如果非纯粹Vector，执行FTS基于BM25召回
	if opts.Mode != "vector" {
		fRes, err := s.storage.FTSSearch(query, opts.TopK*2)
		if err == nil {
			ftsResults = fRes
		}
	}

	// 3. 执行融合
	var finalResults []*models.SearchResult
	if opts.Mode == "vector" {
		finalResults = s.normalizeScores(vectorResults)
	} else if opts.Mode == "fts" {
		finalResults = s.normalizeScores(ftsResults)
	} else { // hybrid
		finalResults = s.rrfMerge(vectorResults, ftsResults)
	}

	// 4. TopK 截断
	if len(finalResults) > opts.TopK {
		finalResults = finalResults[:opts.TopK]
	}

	// 5. Rank
	for i := range finalResults {
		finalResults[i].Rank = i + 1
	}

	return finalResults, nil
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
