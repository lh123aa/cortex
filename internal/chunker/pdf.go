package chunker

import (
	"bytes"
	"fmt"
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

	// 读取 PDF 内容 (基于字节流提取)
	ctx, err := api.ReadContext(bytes.NewReader([]byte(content)), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read pdf context: %w", err)
	}

	pageCount, err := api.PageCount(ctx)
	if err != nil {
		return nil, err
	}

	var chunks []*models.Chunk

	for pageNum := 1; pageNum <= pageCount; pageNum++ {
		var buf bytes.Buffer
		// 提取单页文本
		if err := api.ExtractContent(ctx, []string{fmt.Sprintf("%d", pageNum)}, &buf); err != nil {
			continue // 忽略单页错误
		}

		text := strings.TrimSpace(buf.String())
		if len(text) < c.config.MinChars {
			continue
		}

		// 为简单起见，这里按纯文本换行粗略切割，再进行 Token 检查
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
