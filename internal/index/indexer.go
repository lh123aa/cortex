package index

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lh123aa/cortex/internal/chunker"
	"github.com/lh123aa/cortex/internal/embedding"
	"github.com/lh123aa/cortex/internal/models"
	"github.com/lh123aa/cortex/internal/storage"
)

// Indexer 统筹负责调度文件提取、分块、向量与存储
type Indexer struct {
	storage   storage.Storage
	chunkers  map[string]chunker.Chunker
	embedding embedding.EmbeddingProvider
}

type IndexResult struct {
	Total    int
	Indexed  int
	Skipped  int
	Failed   int
	Duration int64
}

func NewIndexer(s storage.Storage, em embedding.EmbeddingProvider) *Indexer {
	// 配置并挂载具体格式分块器
	ckMap := make(map[string]chunker.Chunker)
// ... in NewIndexer initialization:
	mk, _ := chunker.NewMarkdownChunker(chunker.ChunkConfig{
		MinChars:         50,
		MaxTokens:        512,
		IncludeBreadcrumb: true,
	})
	ckMap["md"] = mk
	
	// v1.1 注册 PDF 引擎
	pk, _ := chunker.NewPDFChunker(chunker.ChunkConfig{
		MinChars:          50,
		MaxTokens:         512,
		IncludeBreadcrumb: true,
	})
	ckMap["pdf"] = pk

	return &Indexer{
		storage:   s,
		chunkers:  ckMap,
		embedding: em,
	}
}

// IndexDirectory 遍历执行整个文件夹
func (idx *Indexer) IndexDirectory(rootPath string) (*IndexResult, error) {
	result := &IndexResult{}

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		result.Total++

		// 选择 Chunker
		fileType := "unknown"
		if strings.HasSuffix(strings.ToLower(path), ".md") {
			fileType = "md"
		}

		// 暴露并执行单一文件的解析
		indexed, skipped, err := idx.IndexFile(path)
		if err != nil {
			result.Failed++
			return nil // 不阻断继续跑
		}

		if indexed {
			result.Indexed++
		}
		if skipped {
			result.Skipped++
		}
		return nil
	})

	return result, err
}

// IndexFile 解析单一文件（暴露给Watcher使用）
func (idx *Indexer) IndexFile(path string) (bool, bool, error) {
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
