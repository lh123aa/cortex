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
	doc := ast.NewDocument()
	if err := c.md.Parser().Parse(reader, doc); err != nil {
		return nil, fmt.Errorf("failed to parse markdown AST: %w", err)
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
			currentHeading = string(heading.Text([]byte(content)))
			currentLevel = heading.Level
			// 当遇到标题时强制打断前面收集的内容作为一个新块返回
			if currentPara.Len() > c.config.MinChars {
				chunk := c.buildChunk(currentPara.String(), currentTokens, path, docIDHash, currentHeading, currentLevel)
				chunks = append(chunks, chunk)
				currentPara.Reset()
				currentTokens = 0
			}
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
	contentWrapped := trimmed
	if c.config.IncludeBreadcrumb {
		breadcrumb := fmt.Sprintf("Section: [%s] > %s\n\n", path, heading)
		contentWrapped = breadcrumb + trimmed
	}

	chID := generateID(path, trimmed)
	return &models.Chunk{
		ID:           chID,
		DocumentID:   docIDHash,
		HeadingPath:  heading,
		HeadingLevel: level,
		Content:      contentWrapped,
		ContentRaw:   trimmed,
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
