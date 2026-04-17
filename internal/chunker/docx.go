package chunker

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/lh123aa/cortex/internal/models"
	"github.com/pkoukk/tiktoken-go"
)

// DocxChunker 实现对 Word (.docx) 格式的解析与处理
type DocxChunker struct {
	config    ChunkConfig
	tokenizer *tiktoken.Tiktoken
}

// NewDocxChunker 初始化 docx 分块器
func NewDocxChunker(config ChunkConfig) (*DocxChunker, error) {
	tk := "cl100k_base"
	if config.Tokenizer != "" {
		tk = config.Tokenizer
	}
	tokenizer, err := tiktoken.GetEncoding(tk)
	if err != nil {
		return nil, fmt.Errorf("failed to init tiktoken for docx: %w", err)
	}

	return &DocxChunker{
		config:    config,
		tokenizer: tokenizer,
	}, nil
}

// Chunk 解析 docx 文件内容并分块
// content 参数接收 docx 文件内容（zip 格式字节）
func (c *DocxChunker) Chunk(content string, path string) ([]*models.Chunk, error) {
	docIDHash := generateID(path, "")

	// docx 本质是 zip，content 是 zip 文件内容
	r, err := zip.NewReader(bytes.NewReader([]byte(content)), int64(len(content)))
	if err != nil {
		return nil, fmt.Errorf("failed to read docx as zip: %w", err)
	}

	var mainDoc []byte
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open document.xml: %w", err)
			}
			mainDoc, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to read document.xml: %w", err)
			}
			break
		}
	}

	if len(mainDoc) == 0 {
		return nil, fmt.Errorf("word/document.xml not found in docx")
	}

	// 解析 XML 提取纯文本
	fullText := extractDocxText(mainDoc)
	if fullText == "" {
		return nil, fmt.Errorf("no text extracted from docx")
	}

	return c.splitByParagraph(fullText, path, docIDHash)
}

// extractDocxText 从 document.xml 中提取文本
func extractDocxText(xmlData []byte) string {
	type Text struct {
		Content string `xml:",chardata"`
	}
	type Run struct {
		Texts []Text `xml:"r t"`
	}
	type Paragraph struct {
		Runs []Run `xml:"p r"`
	}
	type Body struct {
		Paragraphs []Paragraph `xml:"p"`
	}

	var body Body
	if err := xml.Unmarshal(xmlData, &body); err != nil {
		// fallback: 简单提取所有文本节点
		return extractTextSimple(xmlData)
	}

	var sb strings.Builder
	for _, para := range body.Paragraphs {
		for _, run := range para.Runs {
			for _, t := range run.Texts {
				sb.WriteString(t.Content)
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// extractTextSimple 简单文本提取（兜底方案）
func extractTextSimple(data []byte) string {
	start := 0
	var result strings.Builder
	for {
		idx := bytes.Index(data[start:], []byte("<w:t"))
		if idx == -1 {
			break
		}
		idx += start
		closeIdx := bytes.Index(data[idx:], []byte("</w:t>"))
		if closeIdx == -1 {
			break
		}
		closeIdx += idx
		textContent := data[idx:closeIdx]
		// 找到 > 后的内容
		gtIdx := bytes.Index(textContent, []byte(">"))
		if gtIdx != -1 {
			result.Write(textContent[gtIdx+1:])
		}
		start = closeIdx + len("</w:t>")
	}
	return result.String()
}

// splitByParagraph 按段落和 Token Budget 分块
func (c *DocxChunker) splitByParagraph(fullText string, path, docIDHash string) ([]*models.Chunk, error) {
	paragraphs := strings.Split(fullText, "\n")
	var chunks []*models.Chunk

	var currentPara strings.Builder
	var currentTokens int
	currentHeading := "Document Start"
	currentLevel := 0

	flushChunk := func() {
		if currentPara.Len() <= c.config.MinChars {
			currentPara.Reset()
			currentTokens = 0
			return
		}
		text := strings.TrimSpace(currentPara.String())
		tokens := len(c.tokenizer.Encode(text, nil, nil))
		chID := generateID(path, text[:min(50, len(text))])

		content := text
		if c.config.IncludeBreadcrumb {
			content = fmt.Sprintf("Section: [%s] > %s\n\n%s", path, currentHeading, text)
		}

		chunks = append(chunks, &models.Chunk{
			ID:           chID,
			DocumentID:   docIDHash,
			HeadingPath:  currentHeading,
			HeadingLevel: currentLevel,
			Content:      content,
			ContentRaw:   text,
			TokenCount:   tokens,
		})
		currentPara.Reset()
		currentTokens = 0
	}

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		tokens := len(c.tokenizer.Encode(para, nil, nil))

		if currentTokens+tokens > c.config.MaxTokens && currentPara.Len() > c.config.MinChars {
			flushChunk()
		}

		currentPara.WriteString(para)
		currentPara.WriteString("\n")
		currentTokens += tokens
	}

	if currentPara.Len() > c.config.MinChars {
		flushChunk()
	}

	return chunks, nil
}

func (c *DocxChunker) Name() string {
	return "docx"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
