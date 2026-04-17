package chunker

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/lh123aa/cortex/internal/models"
)

// GoChunker 实现对 Go 源代码的解析与分块
type GoChunker struct {
	config ChunkConfig
}

// NewGoChunker 初始化 Go 分块器
func NewGoChunker(config ChunkConfig) (*GoChunker, error) {
	return &GoChunker{
		config: config,
	}, nil
}

// Chunk 解析 Go 代码，按函数/类型/包声明分割
func (c *GoChunker) Chunk(content string, path string) ([]*models.Chunk, error) {
	docIDHash := generateID(path, "")

	// 按包级别声明分割 Go 代码
	sections := c.splitByDeclaration(content)

	var chunks []*models.Chunk
	var currentBlock strings.Builder
	var currentTokens int
	var currentHeading string = "package"
	var currentLevel int = 1

	for _, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}

		// 估算 token 数（简单按单词和符号数估算，1 token ≈ 4 chars）
		sectionTokens := len(section) / 4

		// 判断当前 section 的类型
		sectionHeading := c.detectSectionHeading(section)
		sectionLevel := c.detectSectionLevel(section)

		// 如果新 section 标题变化，且当前块有内容，先输出
		if currentHeading != sectionHeading && currentBlock.Len() > c.config.MinChars {
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

		currentHeading = sectionHeading
		currentLevel = sectionLevel
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

// splitByDeclaration 按 Go 代码声明分割
func (c *GoChunker) splitByDeclaration(content string) []string {
	// 匹配顶级声明: func, type, const, var, package
	// 使用正则表达式找到所有声明行的位置
	declarationPattern := regexp.MustCompile(`(?m)^(func|package|type|const|var)\s+`)

	indices := declarationPattern.FindAllStringIndex(content, -1)

	if len(indices) == 0 {
		// 没有找到声明，整个内容作为一个块
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

// detectSectionHeading 检测代码段的标题（函数/类型名等）
func (c *GoChunker) detectSectionHeading(section string) string {
	lines := strings.Split(section, "\n")
	if len(lines) == 0 {
		return "unknown"
	}

	firstLine := strings.TrimSpace(lines[0])

	// func 函数名
	if strings.HasPrefix(firstLine, "func ") {
		// 可能是 func() 或 func name()
		parts := strings.SplitN(strings.TrimPrefix(firstLine, "func "), "(", 2)
		if len(parts) > 0 {
			funcName := strings.TrimSpace(parts[0])
			if funcName == "" {
				return "func (anonymous)"
			}
			return "func " + funcName
		}
	}

	// type 类型名
	if strings.HasPrefix(firstLine, "type ") {
		parts := strings.SplitN(strings.TrimPrefix(firstLine, "type "), " ", 2)
		if len(parts) > 0 {
			return "type " + strings.TrimSpace(parts[0])
		}
	}

	// package 包名
	if strings.HasPrefix(firstLine, "package ") {
		return "package " + strings.TrimPrefix(firstLine, "package ")
	}

	// const, var
	if strings.HasPrefix(firstLine, "const ") || strings.HasPrefix(firstLine, "var ") {
		return firstLine
	}

	// 默认返回第一行作为标题
	if len(lines) > 0 {
		return truncateString(firstLine, 50)
	}
	return "code"
}

// detectSectionLevel 检测代码段的层级
func (c *GoChunker) detectSectionLevel(section string) int {
	firstLine := strings.TrimSpace(strings.Split(section, "\n")[0])

	if strings.HasPrefix(firstLine, "package ") {
		return 0 // 包级别最高
	}
	if strings.HasPrefix(firstLine, "func ") && !strings.Contains(firstLine, ".") {
		return 1 // 顶级函数
	}
	if strings.HasPrefix(firstLine, "type ") {
		return 1 // 类型定义
	}
	if strings.HasPrefix(firstLine, "func ") {
		return 2 // 方法
	}
	if strings.HasPrefix(firstLine, "const ") || strings.HasPrefix(firstLine, "var ") {
		return 2 // 变量声明
	}
	return 1
}

// buildChunk 创建 Chunk
func (c *GoChunker) buildChunk(rawText string, tokenCount int, path string, docIDHash string, heading string, level int) *models.Chunk {
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
func (c *GoChunker) Name() string {
	return "go"
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
