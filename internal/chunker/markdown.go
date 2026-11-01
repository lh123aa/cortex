package chunker

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/lh123aa/cortex/internal/models"

	// v1.1 恢复并实际接入真正的第三方解析库
	"github.com/pkoukk/tiktoken-go"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// MarkdownChunker 实现对 markdown 的 AST 解析与处理
type MarkdownChunker struct {
	config    ChunkConfig
	tokenizer *tiktoken.Tiktoken
	md        goldmark.Markdown
}

// NewMarkdownChunker 初始化 markdown 分块器
func NewMarkdownChunker(config ChunkConfig) (*MarkdownChunker, error) {
	// Tiktoken 初始化 (首次加载可能会较慢，视词表下载情况)
	tk := "cl100k_base"
	if config.Tokenizer != "" {
		tk = config.Tokenizer
	}
	tokenizer, err := tiktoken.GetEncoding(tk)
	if err != nil {
		return nil, fmt.Errorf("failed to init tiktoken: %w", err)
	}

	return &MarkdownChunker{
		config:    config,
		tokenizer: tokenizer,
		md:        goldmark.New(),
	}, nil
}

// Chunk AST提取重组
func (c *MarkdownChunker) Chunk(content string, path string) ([]*models.Chunk, error) {
	reader := text.NewReader([]byte(content))
	doc := c.md.Parser().Parse(reader)
	if doc == nil {
		return nil, fmt.Errorf("failed to parse markdown AST: result is nil")
	}

	docIDHash := generateID(path, "")

	var chunks []*models.Chunk
	var currentPara strings.Builder
	var currentTokens int
	var currentHeading string = "Document Start"
	var currentLevel int = 0

	// 遍历 AST 叶子节点将其合并到 Chunk 中，不超 Token Budget
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch n.Kind() {
		case ast.KindHeading:
			heading := n.(*ast.Heading)
			// 先用「上一个标题」为已收集的内容建块，再切换标题。
			// 早期实现顺序反了：先更新 currentHeading 再 buildChunk，
			// 结果「数据合规」小节的内容被标成了下一个标题「配额与套餐」，
			// 按标题检索、breadcrumb 上下文、引用来源全部错位。
			if currentPara.Len() > c.config.MinChars {
				chunk := c.buildChunk(currentPara.String(), currentTokens, path, docIDHash, currentHeading, currentLevel)
				chunks = append(chunks, chunk)
				currentPara.Reset()
				currentTokens = 0
			}
			currentHeading = string(heading.Text([]byte(content)))
			currentLevel = heading.Level
		case ast.KindParagraph, ast.KindCodeBlock, ast.KindFencedCodeBlock, ast.KindList, ast.KindBlockquote:
			// 获取改节点下的文本
			var blockText string
			if n.Kind() == ast.KindCodeBlock || n.Kind() == ast.KindFencedCodeBlock {
				// 获取原始代码文本
				lines := n.Lines()
				for i := 0; i < lines.Len(); i++ {
					line := lines.At(i)
					blockText += string(line.Value([]byte(content)))
				}
			} else {
				blockText = string(n.Text([]byte(content)))
			}

			blockText = strings.TrimSpace(blockText)
			if blockText == "" {
				return ast.WalkContinue, nil
			}

			// 精确计算 Tokens
			tokens := len(c.tokenizer.Encode(blockText, nil, nil))

			if currentTokens+tokens > c.config.MaxTokens && currentPara.Len() > c.config.MinChars {
				// 如果合并后超标，先吐出以前收集的
				chunk := c.buildChunk(currentPara.String(), currentTokens, path, docIDHash, currentHeading, currentLevel)
				chunks = append(chunks, chunk)
				currentPara.Reset()
				currentTokens = 0
			}

			currentPara.WriteString(blockText)
			currentPara.WriteString("\n\n")
			currentTokens += tokens
		}
		return ast.WalkContinue, nil
	})

	// 循环结束后打扫剩余节点
	if currentPara.Len() > c.config.MinChars {
		chunk := c.buildChunk(currentPara.String(), currentTokens, path, docIDHash, currentHeading, currentLevel)
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

func (c *MarkdownChunker) buildChunk(rawText string, tokenCount int, path string, docIDHash string, heading string, level int) *models.Chunk {
	trimmed := strings.TrimSpace(rawText)
	// 中文分词优化：正文与标题都要分词。
	// 标题（例如「配额与套餐」）如果原样入库，FTS5 会把它当成一个整词，
	// 用户搜「配额」时无法命中 —— 只有正文分词是不够的。
	contentForSearch := OptimizeForChineseSearch(trimmed)
	headingForSearch := OptimizeForChineseSearch(heading)
	contentWrapped := trimmed
	if c.config.IncludeBreadcrumb {
		breadcrumb := fmt.Sprintf("Section: [%s] > %s\n\n", path, heading)
		contentWrapped = breadcrumb + trimmed
	}
	// 标题分词结果并入 content_raw，使标题关键词可被全文检索命中。
	if headingForSearch != "" && headingForSearch != heading {
		contentForSearch = headingForSearch + "\n" + contentForSearch
	}

	chID := generateID(path, trimmed)
	return &models.Chunk{
		ID:           chID,
		DocumentID:   docIDHash,
		HeadingPath:  heading,
		HeadingLevel: level,
		Content:      contentWrapped,
		ContentRaw:   contentForSearch,
		TokenCount:   tokenCount,
	}
}

func (c *MarkdownChunker) Name() string {
	return "markdown"
}

func generateID(path, mix string) string {
	h := sha256.New()
	h.Write([]byte(path + mix))
	return hex.EncodeToString(h.Sum(nil))[:16]
}
