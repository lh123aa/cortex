package chunker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lh123aa/cortex/internal/models"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pkoukk/tiktoken-go"
)

type PDFChunker struct {
	config    ChunkConfig
	tokenizer *tiktoken.Tiktoken
}

func NewPDFChunker(config ChunkConfig) (*PDFChunker, error) {
	tk := "cl100k_base"
	if config.Tokenizer != "" {
		tk = config.Tokenizer
	}
	tokenizer, err := tiktoken.GetEncoding(tk)
	if err != nil {
		return nil, fmt.Errorf("failed to init tiktoken: %w", err)
	}

	return &PDFChunker{
		config:    config,
		tokenizer: tokenizer,
	}, nil
}

func (c *PDFChunker) Chunk(content string, path string) ([]*models.Chunk, error) {
	docIDHash := generateID(path, "")

	// 创建临时目录用于提取
	tmpDir, err := os.MkdirTemp("", "cortex-pdf-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// 写入临时 PDF
	tmpPdf := filepath.Join(tmpDir, "input.pdf")
	if err := os.WriteFile(tmpPdf, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write temp pdf: %w", err)
	}

	// 获取页数
	reader, err := os.Open(tmpPdf)
	if err != nil {
		return nil, fmt.Errorf("failed to open temp pdf: %w", err)
	}

	pageCount, err := api.PageCount(reader, nil)
	reader.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to get page count: %w", err)
	}

	var chunks []*models.Chunk

	// 逐页提取文本
	for pageNum := 1; pageNum <= pageCount; pageNum++ {
		// 打开文件用于提取
		f, err := os.Open(tmpPdf)
		if err != nil {
			continue
		}

		// v0.8 API: ExtractContent(rs io.ReadSeeker, outDir, fileName string, selectedPages []string, conf *model.Configuration) error
		pageDir := filepath.Join(tmpDir, fmt.Sprintf("page%d", pageNum))
		os.MkdirAll(pageDir, 0755)

		err = api.ExtractContent(f, pageDir, "", []string{fmt.Sprintf("%d", pageNum)}, nil)
		f.Close()
		if err != nil {
			continue
		}

		// 读取提取的文本
		textFiles, _ := filepath.Glob(filepath.Join(pageDir, "*.txt"))
		var pageText strings.Builder
		for _, tf := range textFiles {
			data, _ := os.ReadFile(tf)
			pageText.Write(data)
		}

		text := strings.TrimSpace(pageText.String())
		if len(text) < c.config.MinChars {
			continue
		}

		// 按段落分割
		paras := strings.Split(text, "\n\n")
		var currPara strings.Builder
		var currTokens int

		for _, p := range paras {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			tks := len(c.tokenizer.Encode(p, nil, nil))

			if currTokens+tks > c.config.MaxTokens && currPara.Len() > c.config.MinChars {
				chunk := c.buildChunk(currPara.String(), currTokens, path, docIDHash, pageNum)
				chunks = append(chunks, chunk)
				currPara.Reset()
				currTokens = 0
			}

			currPara.WriteString(p)
			currPara.WriteString("\n")
			currTokens += tks
		}

		if currPara.Len() > c.config.MinChars {
			chunk := c.buildChunk(currPara.String(), currTokens, path, docIDHash, pageNum)
			chunks = append(chunks, chunk)
		}
	}

	return chunks, nil
}

func (c *PDFChunker) buildChunk(rawText string, tokenCount int, path string, docIDHash string, pageNum int) *models.Chunk {
	trimmed := strings.TrimSpace(rawText)
	contentWrapped := trimmed
	if c.config.IncludeBreadcrumb {
		breadcrumb := fmt.Sprintf("Section: [%s] > Page %d\n\n", path, pageNum)
		contentWrapped = breadcrumb + trimmed
	}

	chID := generateID(path, fmt.Sprintf("p%d_%s", pageNum, trimmed))
	return &models.Chunk{
		ID:           chID,
		DocumentID:   docIDHash,
		HeadingPath:  fmt.Sprintf("Page %d", pageNum),
		HeadingLevel: 1,
		Content:      contentWrapped,
		ContentRaw:   trimmed,
		TokenCount:   tokenCount,
	}
}

func (c *PDFChunker) Name() string {
	return "pdf"
}
