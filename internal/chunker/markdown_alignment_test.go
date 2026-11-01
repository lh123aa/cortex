package chunker

import (
	"strings"
	"testing"
)

// TestMarkdownChunkHeadingContentAlignment 回归测试：
// 早期实现遇到新标题时先更新 currentHeading 再为已收集内容建块，
// 导致「数据合规」小节的内容被标成「配额与套餐」标题。
// 正确行为：每个 chunk 的 HeadingPath 必须是它自身内容的那个标题。
func TestMarkdownChunkHeadingContentAlignment(t *testing.T) {
	config := ChunkConfig{
		MinChars:          5,
		MaxTokens:         500,
		IncludeBreadcrumb: true,
	}
	c, err := NewMarkdownChunker(config)
	if err != nil {
		t.Fatalf("failed to create chunker: %v", err)
	}

	md := `# 企业私有化知识库选型

## 数据合规
金融、医疗、法务团队要求数据不出内网，所有内容必须留在本地。

## 配额与套餐
Free 档位限制 5000 文档与 1GiB 存储，Pro 提升到 50 万文档。
`
	chunks, err := c.Chunk(md, "compliance.md")
	if err != nil {
		t.Fatalf("chunk failed: %v", err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}

	// 找到包含「数据不出内网」的 chunk，它的标题必须是「数据合规」
	foundCompliance := false
	for _, ch := range chunks {
		if !strings.Contains(ch.ContentRaw, "不 出 内 网") {
			continue
		}
		foundCompliance = true
		if ch.HeadingPath != "数据合规" {
			t.Errorf("chunk containing 「数据不出内网」 has heading %q, want 「数据合规」", ch.HeadingPath)
		}
	}
	if !foundCompliance {
		t.Fatalf("no chunk contains the 「数据合规」 section body")
	}

	// 包含「Free 档位」的 chunk，标题必须是「配额与套餐」
	foundQuota := false
	for _, ch := range chunks {
		if !strings.Contains(ch.ContentRaw, "Free") {
			continue
		}
		foundQuota = true
		if ch.HeadingPath != "配额与套餐" {
			t.Errorf("chunk containing 「Free 档位」 has heading %q, want 「配额与套餐」", ch.HeadingPath)
		}
	}
	if !foundQuota {
		t.Fatalf("no chunk contains the 「配额与套餐」 section body")
	}
}
