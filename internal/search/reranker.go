package search

import (
	"context"
	"sort"

	"github.com/lh123aa/cortex/internal/models"
)

// Reranker 重排序器接口
type Reranker interface {
	// Rerank 对搜索结果进行重排序
	Rerank(ctx context.Context, query string, results []*models.SearchResult, topK int) ([]*models.SearchResult, error)
}

// CrossEncoderReranker 基于 Cross-Encoder 的重排序
// 通过更精确的相关性计算来优化排序结果
type CrossEncoderReranker struct {
	rerankTopK int // 重排序时保留的结果数
}

// NewCrossEncoderReranker 创建 Cross-Encoder 重排序器
func NewCrossEncoderReranker(rerankTopK int) *CrossEncoderReranker {
	if rerankTopK <= 0 {
		rerankTopK = 20
	}
	return &CrossEncoderReranker{
		rerankTopK: rerankTopK,
	}
}

// Rerank 使用简单的文本相似度重排序
// 注意: 实际生产环境应使用真正的 Cross-Encoder 模型 (如 mxbai-rerank)
func (r *CrossEncoderReranker) Rerank(ctx context.Context, query string, results []*models.SearchResult, topK int) ([]*models.SearchResult, error) {
	if len(results) == 0 {
		return results, nil
	}

	// 计算每个结果的相关性分数
	scored := make([]*scoredResult, 0, len(results))
	for _, result := range results {
		score := r.calculateRelevance(query, result)
		scored = append(scored, &scoredResult{
			result: result,
			score:  score,
		})
	}

	// 按相关性分数排序
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// 取 topK
	if len(scored) > topK {
		scored = scored[:topK]
	}

	// 更新结果的 Score
	reRanked := make([]*models.SearchResult, len(scored))
	for i, s := range scored {
		s.result.Score = s.score
		s.result.Rank = i + 1
		reRanked[i] = s.result
	}

	return reRanked, nil
}

// scoredResult 带分数的结果
type scoredResult struct {
	result *models.SearchResult
	score  float64
}

// calculateRelevance 计算相关性分数
// 使用多种信号: 词汇匹配、语义相似度、结构化信息
func (r *CrossEncoderReranker) calculateRelevance(query string, result *models.SearchResult) float64 {
	var score float64

	queryLower := toLower(query)
	contentLower := toLower(result.Chunk.ContentRaw)
	headingLower := toLower(result.Chunk.HeadingPath)

	// 1. 精确词匹配 (关键词出现次数)
	matchCount := countWordMatches(queryLower, contentLower)
	score += float64(matchCount) * 0.1

	// 2. 标题匹配 (Heading 匹配权重更高)
	if containsWord(headingLower, queryLower) {
		score += 2.0
	}
	titleMatchCount := countWordMatches(queryLower, headingLower)
	score += float64(titleMatchCount) * 0.3

	// 3. 连续匹配奖励 (query 在 content 中连续出现)
	if containsSequential(queryLower, contentLower) {
		score += 1.5
	}
	if containsSequential(queryLower, headingLower) {
		score += 2.0
	}

	// 4. 原始分数加权 (保留原有的向量/BM25分数)
	score += result.Score * 3.0

	// 5. 位置奖励 (chunk 在文档中的位置)
	// 开头的 chunk 可能有更高的上下文价值
	if result.Chunk.LineStart > 0 && result.Chunk.LineStart < 50 {
		score += 0.2
	}

	// 6. 长度惩罚 (太短的 chunk 可能信息不足)
	contentLen := len(result.Chunk.ContentRaw)
	if contentLen < 50 {
		score -= 0.5
	} else if contentLen > 500 {
		score += 0.1
	}

	return score
}

// toLower 字符串转小写
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

// countWordMatches 计算查询词在内容中的匹配次数
func countWordMatches(query, content string) int {
	// 简单的词匹配计数
	// 实际应该使用分词器
	queryWords := splitWords(query)
	contentWords := splitWords(content)

	matchCount := 0
	contentWordSet := make(map[string]bool)
	for _, w := range contentWords {
		contentWordSet[w] = true
	}

	for _, w := range queryWords {
		if len(w) < 2 {
			continue
		}
		if contentWordSet[w] {
			matchCount++
		}
	}

	return matchCount
}

// containsWord 检查 content 是否包含 query 中的任意词
func containsWord(content, query string) bool {
	queryWords := splitWords(query)
	contentWords := splitWords(content)

	contentWordSet := make(map[string]bool)
	for _, w := range contentWords {
		contentWordSet[w] = true
	}

	for _, w := range queryWords {
		if len(w) < 2 {
			continue
		}
		if contentWordSet[w] {
			return true
		}
	}
	return false
}

// containsSequential 检查 query 是否作为子串连续出现在 content 中
func containsSequential(query, content string) bool {
	// 简单的子串匹配
	return len(query) >= 3 && len(content) >= len(query) &&
		indexOf(content, query) >= 0
}

// indexOf 子串第一次出现的位置，不存在返回 -1
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// splitWords 简单分词 (按空格和标点分割)
func splitWords(s string) []string {
	var words []string
	var current []byte

	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ' ' || c == ',' || c == '.' || c == '\n' || c == '\t' {
			if len(current) > 0 {
				words = append(words, string(current))
				current = nil
			}
		} else {
			current = append(current, c)
		}
	}

	if len(current) > 0 {
		words = append(words, string(current))
	}

	return words
}

// HybridReranker 混合重排序器 (结合多种策略)
type HybridReranker struct {
	rerankers []Reranker
	weights   []float64
}

// NewHybridReranker 创建混合重排序器
func NewHybridReranker(rerankers []Reranker, weights []float64) *HybridReranker {
	if len(rerankers) != len(weights) {
		// 默认等权重
		weights = make([]float64, len(rerankers))
		for i := range weights {
			weights[i] = 1.0
		}
	}
	return &HybridReranker{
		rerankers: rerankers,
		weights:   weights,
	}
}

// Rerank 对结果进行多重排序并融合
func (h *HybridReranker) Rerank(ctx context.Context, query string, results []*models.SearchResult, topK int) ([]*models.SearchResult, error) {
	if len(h.rerankers) == 0 {
		return results, nil
	}

	// 如果只有一个重排序器，直接使用
	if len(h.rerankers) == 1 {
		return h.rerankers[0].Rerank(ctx, query, results, topK)
	}

	// 收集所有重排序结果
	allScores := make([]map[string]float64, len(h.rerankers))

	for i, reranker := range h.rerankers {
		reranked, err := reranker.Rerank(ctx, query, results, len(results))
		if err != nil {
			continue
		}

		scores := make(map[string]float64)
		for rank, r := range reranked {
			// 倒数排名加权
			scores[r.Chunk.ID] = 1.0 / float64(rank+1)
		}
		allScores[i] = scores
	}

	// 融合多个分数
	finalScores := make(map[string]float64)
	for i, scores := range allScores {
		weight := h.weights[i]
		for id, score := range scores {
			finalScores[id] += score * weight
		}
	}

	// 构建最终结果
	idToResult := make(map[string]*models.SearchResult)
	for _, r := range results {
		idToResult[r.Chunk.ID] = r
	}

	type resultWithScore struct {
		result *models.SearchResult
		score  float64
	}

	var finalResults []*resultWithScore
	for id, score := range finalScores {
		if result, ok := idToResult[id]; ok {
			finalResults = append(finalResults, &resultWithScore{
				result: result,
				score:  score,
			})
		}
	}

	// 排序
	sort.Slice(finalResults, func(i, j int) bool {
		return finalResults[i].score > finalResults[j].score
	})

	// 截取 topK
	if len(finalResults) > topK {
		finalResults = finalResults[:topK]
	}

	// 构建返回结果
	reranked := make([]*models.SearchResult, len(finalResults))
	for i, rs := range finalResults {
		rs.result.Score = rs.score
		rs.result.Rank = i + 1
		reranked[i] = rs.result
	}

	return reranked, nil
}

// BM25Reranker 基于 BM25 的重排序
type BM25Reranker struct {
	k1 float64
	b  float64
}

// NewBM25Reranker 创建 BM25 重排序器
func NewBM25Reranker() *BM25Reranker {
	return &BM25Reranker{
		k1: 1.5, // term frequency saturation
		b:  0.75, // document length normalization
	}
}

// Rerank 使用 BM25 算法重排序
func (r *BM25Reranker) Rerank(ctx context.Context, query string, results []*models.SearchResult, topK int) ([]*models.SearchResult, error) {
	if len(results) == 0 {
		return results, nil
	}

	queryWords := splitWords(toLower(query))
	avgLen := 0.0
	for _, res := range results {
		avgLen += float64(len(res.Chunk.ContentRaw))
	}
	if len(results) > 0 {
		avgLen /= float64(len(results))
	}
	if avgLen == 0 {
		avgLen = 1.0
	}

	scored := make([]*scoredResult, len(results))
	for i, res := range results {
		content := toLower(res.Chunk.ContentRaw)
		contentWords := splitWords(content)
		contentLen := float64(len(content))

		var bm25Score float64
		wordFreq := make(map[string]int)
		for _, w := range contentWords {
			wordFreq[w]++
		}

		for _, qw := range queryWords {
			if len(qw) < 2 {
				continue
			}
			freq := float64(wordFreq[qw])
			if freq > 0 {
				// 简化的 BM25 计算
				idf := 1.0 // 简化，实际应该根据语料库计算
				tf := freq / (freq + r.k1*(1-r.b+r.b*(contentLen/avgLen)))
				bm25Score += idf * tf
			}
		}

		// 结合原始分数
		combinedScore := bm25Score*0.5 + res.Score*0.5

		scored[i] = &scoredResult{
			result: res,
			score:  combinedScore,
		}
	}

	// 排序
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// 截取 topK
	if len(scored) > topK {
		scored = scored[:topK]
	}

	reranked := make([]*models.SearchResult, len(scored))
	for i, s := range scored {
		s.result.Score = s.score
		s.result.Rank = i + 1
		reranked[i] = s.result
	}

	return reranked, nil
}