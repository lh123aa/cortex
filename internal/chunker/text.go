package chunker

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/lh123aa/cortex/internal/models"
)

// TextChunker 实现对通用文本/配置文件的解析与分块
type TextChunker struct {
	config     ChunkConfig
	splitRegex *regexp.Regexp
}

// NewTextChunker 初始化通用文本分块器
func NewTextChunker(config ChunkConfig) (*TextChunker, error) {
	// 按空行或特定模式分割
	splitRegex := regexp.MustCompile(`(?m)^\n|## |\n\n+`)
	return &TextChunker{
		config:     config,
		splitRegex: splitRegex,
	}, nil
}

// Chunk 解析文本内容
func (c *TextChunker) Chunk(content string, path string) ([]*models.Chunk, error) {
	docIDHash := generateID(path, "")

	// 先尝试按标题分割（## 开头）
	sections := c.splitByHeaders(content)

	var chunks []*models.Chunk
	var currentBlock strings.Builder
	var currentTokens int
	var currentHeading string = "Document"
	var currentLevel int = 0

	for _, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}

		// 估算 token 数
		sectionTokens := len(section) / 4

		// 检测是否为标题
		heading, level := c.detectHeading(section)

		// 如果是标题变化，先输出当前块
		if level > 0 && currentHeading != heading && currentBlock.Len() > c.config.MinChars {
			chunk := c.buildChunk(currentBlock.String(), currentTokens, path, docIDHash, currentHeading, currentLevel)
			chunks = append(chunks, chunk)
			currentBlock.Reset()
			currentTokens = 0
		}

		// 如果合并后超标，先输出当前块
		if currentTokens+sectionTokens > c.config.MaxTokens && currentBlock.Len() > c.config.MinChars {
			chunk := c.buildChunk(currentBlock.String(), currentTokens, path, docIDHash, currentHeading, currentLevel)
			chunks = append(chunks, chunk)
			currentBlock.Reset()
			currentTokens = 0
		}

		currentHeading = heading
		currentLevel = level
		currentBlock.WriteString(section)
		currentBlock.WriteString("\n\n")
		currentTokens += sectionTokens
	}

	// 处理剩余内容
	if currentBlock.Len() > c.config.MinChars {
		chunk := c.buildChunk(currentBlock.String(), currentTokens, path, docIDHash, currentHeading, currentLevel)
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// splitByHeaders 按 Markdown 标题分割
func (c *TextChunker) splitByHeaders(content string) []string {
	headerPattern := regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)$`)

	indices := headerPattern.FindAllStringIndex(content, -1)

	if len(indices) == 0 {
		// 没有找到标题，整个内容作为一个块
		return []string{content}
	}

	var sections []string
	start := 0

	for _, match := range indices {
		pos := match[0]
		if pos > start {
			section := content[start:pos]
			if strings.TrimSpace(section) != "" {
				sections = append(sections, section)
			}
		}
		start = pos
	}

	// 添加最后一部分
	if start < len(content) {
		section := content[start:]
		if strings.TrimSpace(section) != "" {
			sections = append(sections, section)
		}
	}

	return sections
}

// detectHeading 检测标题
func (c *TextChunker) detectHeading(section string) (string, int) {
	lines := strings.Split(section, "\n")
	if len(lines) == 0 {
		return "unknown", 0
	}

	firstLine := strings.TrimSpace(lines[0])

	// Markdown 标题
	if strings.HasPrefix(firstLine, "# ") {
		return strings.TrimPrefix(firstLine, "# "), 1
	}
	if strings.HasPrefix(firstLine, "## ") {
		return strings.TrimPrefix(firstLine, "## "), 2
	}
	if strings.HasPrefix(firstLine, "### ") {
		return strings.TrimPrefix(firstLine, "### "), 3
	}

	// JSON/YAML key-value (第一行作为标题)
	if strings.Contains(firstLine, ":") && !strings.Contains(firstLine, " ") {
		return firstLine, 1
	}

	// 文件名作为标题
	return "section", 0
}

// buildChunk 创建 Chunk
func (c *TextChunker) buildChunk(rawText string, tokenCount int, path string, docIDHash string, heading string, level int) *models.Chunk {
	trimmed := strings.TrimSpace(rawText)
	contentWrapped := trimmed
	if c.config.IncludeBreadcrumb {
		breadcrumb := fmt.Sprintf("File: %s | %s\n\n", path, heading)
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

// Name 返回 chunker 名称
func (c *TextChunker) Name() string {
	return "text"
}
