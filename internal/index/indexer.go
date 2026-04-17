package index

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lh123aa/cortex/internal/chunker"
	"github.com/lh123aa/cortex/internal/embedding"
	"github.com/lh123aa/cortex/internal/metrics"
	"github.com/lh123aa/cortex/internal/models"
	"github.com/lh123aa/cortex/internal/storage"
	"github.com/panjf2000/ants/v2"
)

// Indexer 统筹负责调度文件提取、分块、向量与存储
type Indexer struct {
	storage   storage.Storage
	chunkers  map[string]chunker.Chunker
	embedding embedding.EmbeddingProvider
	pool      *ants.Pool
}

// NewIndexer 初始化索引器
func NewIndexer(s storage.Storage, em embedding.EmbeddingProvider) (*Indexer, error) {
	ckMap := make(map[string]chunker.Chunker)
	mk, _ := chunker.NewMarkdownChunker(chunker.ChunkConfig{
		MinChars:         50,
		MaxTokens:        512,
		IncludeBreadcrumb: true,
	})
	ckMap["md"] = mk

	pk, _ := chunker.NewPDFChunker(chunker.ChunkConfig{
		MinChars:          50,
		MaxTokens:         512,
		IncludeBreadcrumb: true,
	})
	ckMap["pdf"] = pk

	dk, _ := chunker.NewDocxChunker(chunker.ChunkConfig{
		MinChars:          50,
		MaxTokens:         512,
		IncludeBreadcrumb: true,
	})
	ckMap["docx"] = dk

	gk, _ := chunker.NewGoChunker(chunker.ChunkConfig{
		MinChars:          50,
		MaxTokens:         512,
		IncludeBreadcrumb: true,
	})
	ckMap["go"] = gk

	// 通用文本 chunker（用于 yaml, yml, json, txt, toml, ini, cfg, conf, hcl, env, properties, xml, html, css, js, ts, py, rb, java, cpp, c, h, sh, bash, zsh, ps1 等）
	tk, _ := chunker.NewTextChunker(chunker.ChunkConfig{
		MinChars:          50,
		MaxTokens:         512,
		IncludeBreadcrumb: true,
	})
	ckMap["yaml"] = tk
	ckMap["yml"] = tk
	ckMap["json"] = tk
	ckMap["txt"] = tk
	ckMap["toml"] = tk
	ckMap["ini"] = tk
	ckMap["cfg"] = tk
	ckMap["conf"] = tk
	ckMap["hcl"] = tk
	ckMap["env"] = tk
	ckMap["properties"] = tk
	ckMap["xml"] = tk
	ckMap["html"] = tk
	ckMap["css"] = tk
	ckMap["js"] = tk
	ckMap["ts"] = tk
	ckMap["py"] = tk
	ckMap["rb"] = tk
	ckMap["java"] = tk
	ckMap["cpp"] = tk
	ckMap["c"] = tk
	ckMap["h"] = tk
	ckMap["sh"] = tk
	ckMap["bash"] = tk
	ckMap["zsh"] = tk
	ckMap["ps1"] = tk
	// md 保持使用 MarkdownChunker（更好的 AST 解析）

	// P2-2: 初始化 goroutine pool（默认 4 个 worker）
	p, err := ants.NewPool(4, ants.WithPreAlloc(false))
	if err != nil {
		return nil, fmt.Errorf("failed to create worker pool: %w", err)
	}

	return &Indexer{
		storage:   s,
		chunkers:  ckMap,
		embedding: em,
		pool:      p,
	}, nil
}

type IndexResult struct {
	Total    int
	Indexed  int
	Skipped  int
	Failed   int
	Duration int64
}

// IndexDirectory 遍历执行整个文件夹（并发优化）
func (idx *Indexer) IndexDirectory(rootPath string) (*IndexResult, error) {
	start := time.Now()
	result := &IndexResult{}

	// P2-2: 第一阶段 — 收集所有文件路径
	var files []string
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	result.Total = len(files)

	// 第二阶段 — 使用 goroutine pool 并发处理
	type fileResult struct {
		indexed bool
		skipped bool
		err     error
	}
	resultCh := make(chan fileResult, len(files))
	var wg sync.WaitGroup

	for _, file := range files {
		wg.Add(1)
		err := idx.pool.Submit(func() {
			defer wg.Done()
			indexed, skipped, err := idx.indexFileInternal(file)
			resultCh <- fileResult{indexed: indexed, skipped: skipped, err: err}
		})
		if err != nil {
			wg.Done()
			resultCh <- fileResult{err: fmt.Errorf("pool submit error: %w", err)}
		}
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	for res := range resultCh {
		if res.indexed {
			result.Indexed++
		}
		if res.skipped {
			result.Skipped++
		}
		if res.err != nil {
			result.Failed++
		}
		metrics.IndexTotal.Inc()
	}

	result.Duration = time.Since(start).Milliseconds()
	return result, nil
}

// IndexFile 解析单一文件（暴露给Watcher使用，内部调用）
func (idx *Indexer) IndexFile(path string) (bool, bool, error) {
	return idx.indexFileInternal(path)
}

// indexFileInternal 实际索引逻辑
func (idx *Indexer) indexFileInternal(path string) (bool, bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, false, err
	}

	// 解析文件类型
	fileType := "unknown"
	pathLower := strings.ToLower(path)
	if strings.HasSuffix(pathLower, ".md") {
		fileType = "md"
	} else if strings.HasSuffix(pathLower, ".pdf") {
		fileType = "pdf"
	} else if strings.HasSuffix(pathLower, ".docx") {
		fileType = "docx"
	}
	
	ck, ok := idx.chunkers[fileType]
	if !ok {
		return false, true, fmt.Errorf("unsupported file type: %s", path)
	}

	hashBytes := sha256.Sum256(content)
	hashStr := hex.EncodeToString(hashBytes[:])
	docID := hashStr[:16]

	// 查询是否存在及比对Hash
	doc, _ := idx.storage.GetDocumentByPath(path)
	if doc != nil && doc.ContentHash == hashStr {
		// 跳过重复索引
		return false, true, nil
	}


	if doc != nil {
		idx.storage.DeleteChunksByDocument(doc.ID)
	}

	// 开始文本切块解析
	chunks, err := ck.Chunk(string(content), path)
	if err != nil || len(chunks) == 0 {
		return false, false, err
	}

	// 转换为向量
	texts := make([]string, len(chunks))
	for i, c := range chunks {
		texts[i] = c.ContentRaw
		c.DocumentID = docID
	}

	// 进行Embedding
	if idx.embedding != nil {
		embeddings, err := idx.embedding.EmbedBatch(texts)
		if err == nil {
			for i, c := range chunks {
				c.Embedding = embeddings[i]
				c.EmbeddingModel = idx.embedding.Name()
			}
		}
	}

	// 保存 Document
	newDoc := &models.Document{
		ID:          docID,
		Path:        path,
		FileType:    fileType,
		ContentHash: hashStr,
		FileSize:    int64(len(content)),
		ChunkCount:  len(chunks),
		Status:      "indexed",
	}

	if err := idx.storage.SaveDocument(newDoc); err != nil {
		return false, false, err
	}

	// 保存 Chunks
	if err := idx.storage.SaveChunks(chunks); err != nil {
		return false, false, fmt.Errorf("saving chunks failed: %w", err)
	}

	return true, false, nil
}
