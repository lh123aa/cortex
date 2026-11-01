package chunker

import (
	"strings"
	"testing"
)

// TestMarkdownChunkHeadingIsSegmented 回归测试：
// 中文标题（如「配额与套餐」）如果原样进 FTS，会被当成单个整词，
// 用户搜索「配额」时不命中。标题必须和正文一样做 CJK 单字分词，
// 并把分词结果并入 ContentRaw 以便全文检索。
func TestMarkdownChunkHeadingIsSegmented(t *testing.T) {
	config := ChunkConfig{
		MinChars:          5,
		MaxTokens:         500,
		IncludeBreadcrumb: true,
	}
	c, err := NewMarkdownChunker(config)
	if err != nil {
		t.Fatalf("failed to create chunker: %v", err)
	}

	md := "# 配额与套餐\n\nFree 档位限制 5000 文档与 1GiB 存储。\n"
	chunks, err := c.Chunk(md, "quota.md")
	if err != nil {
		t.Fatalf("chunk failed: %v", err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}

	// 至少有一个 chunk 的 ContentRaw 里包含被空格分隔的标题词项
	foundSegmented := false
	for _, ch := range chunks {
		if strings.Contains(ch.ContentRaw, "配 额") || strings.Contains(ch.ContentRaw, "套 餐") {
			foundSegmented = true
			break
		}
	}
	if !foundSegmented {
		for _, ch := range chunks {
			t.Logf("chunk heading=%q contentRaw=%q", ch.HeadingPath, ch.ContentRaw)
		}
		t.Fatal("heading was not CJK-segmented into ContentRaw; searching 「配额」 would miss")
	}
}
