package rag

import (
	"context"
	"strings"

	"github.com/lh123aa/cortex/internal/models"
)

type SearchEngine interface {
	Search(ctx context.Context, query string, opts models.SearchOptions) ([]*models.SearchResult, error)
}

type RAGBuilder struct {
	searchEngine SearchEngine
}

func NewRAGBuilder(se SearchEngine) *RAGBuilder {
	return &RAGBuilder{
		searchEngine: se,
	}
}

// RAGContext 表示拼接好的提词上下文反馈
type RAGContext struct {
	Context     string `json:"context"`
	TokenCount  int    `json:"token_count"`
	TokenBudget int    `json:"token_budget"`
	Truncated   bool   `json:"truncated"`
}

// BuildContext 控制 Token 预算，按智能标点回退截断块
func (b *RAGBuilder) BuildContext(ctx context.Context, query string, tokenBudget int, opts models.SearchOptions) (*RAGContext, error) {
	opts.TopK = 50

	// v1.1 若提供缓存，利用引擎包装好的 SearchWithCache，在此保持向后兼容暂利用 Search
	results, err := b.searchEngine.Search(ctx, query, opts)
	if err != nil {
		return nil, err
	}

	var sb strings.Builder
	totalTokens := 0
	truncated := false

	for _, result := range results {
		chunkTokens := result.Chunk.TokenCount
		sepTokens := 2 // \n\n--- 约占 2 token

		if sb.Len() > 0 {
			if totalTokens+sepTokens+chunkTokens > tokenBudget {
				// v1.1 触及边界：执行边界感知截断 (Smart Truncation)
				remainingTokens := tokenBudget - (totalTokens + sepTokens)
				if remainingTokens > 50 {
					// 还有足够空间容纳部分片段，按标点切断
					smartStr := smartTruncate(result.Chunk.Content, remainingTokens)
					sb.WriteString("\n\n---\n\n")
					sb.WriteString(smartStr)
					totalTokens += remainingTokens
				}
				truncated = true
				break
			}
			sb.WriteString("\n\n---\n\n")
			totalTokens += sepTokens
		} else if chunkTokens > tokenBudget {
			// 首块就超标时做强截断
			sb.WriteString(smartTruncate(result.Chunk.Content, tokenBudget))
			totalTokens = tokenBudget
			truncated = true
			break
		}

		sb.WriteString(result.Chunk.Content)
		totalTokens += chunkTokens
	}

	return &RAGContext{
		Context:     sb.String(),
		TokenCount:  totalTokens,
		TokenBudget: tokenBudget,
		Truncated:   truncated,
	}, nil
}

// smartTruncate 回退到最近的完整句号/感叹号/问号/换行
// 注意: 此处 remainingTokens 系近似对应到中英文字符比例, 通常 1 token 约等 1.5 - 2 汉字, 简化为 proportional cutoff
func smartTruncate(text string, remainingTokens int) string {
	maxChars := remainingTokens * 2
	if len(text) <= maxChars {
		return text
	}
	subStr := text[:maxChars]

	lastPunc := -1
	delims := []string{"。", ".", "！", "!", "？", "?", "\n"}
	for _, d := range delims {
		idx := strings.LastIndex(subStr, d)
		if idx > lastPunc {
			// 如果是多字节如中文句号，找齐它的长度
			lastPunc = idx + len(d)
		}
	}

	if lastPunc > 0 {
		return subStr[:lastPunc] + " [片段截断...]"
	}

	return subStr + "..."
}
