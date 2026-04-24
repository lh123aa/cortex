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

// IndexDirectoryWithCheckpoint 遍历执行整个文件夹（支持断点恢复，用户隔离）
func (idx *Indexer) IndexDirectoryWithCheckpoint(rootPath string, userID string) (*IndexResult, error) {
	start := time.Now()
	result := &IndexResult{}

	// 尝试获取已有进度
	progress, err := idx.storage.GetIndexProgress(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get index progress: %w", err)
	}

	// 初始化或恢复进度
	if progress == nil {
		progress = &models.IndexProgress{
			RootPath:  rootPath,
			Status:    "running",
			StartedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	// 第一阶段 — 收集所有文件路径
	var allFiles []string
	err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		allFiles = append(allFiles, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	result.Total = len(allFiles)
	progress.TotalFiles = len(allFiles)

	// 从上次中断的位置继续
	startIndex := progress.LastFileIndex
	if startIndex >= len(allFiles) {
		// 已经全部处理完成
		result.Indexed = progress.IndexedFiles
		result.Skipped = progress.Skipped
		result.Failed = progress.FailedFiles
		idx.storage.CompleteIndexProgress(rootPath)
		return result, nil
	}

	// 处理剩余文件
	filesToProcess := allFiles[startIndex:]
	resultCh := make(chan fileResult, len(filesToProcess))
	var wg sync.WaitGroup

	for i, file := range filesToProcess {
		wg.Add(1)
		// 闭包捕获 userID
		currentUserID := userID
		err := idx.pool.Submit(func() {
			defer wg.Done()
			indexed, skipped, err := idx.indexFileInternalWithUser(file, currentUserID)
			resultCh <- fileResult{indexed: indexed, skipped: skipped, err: err}
		})
		if err != nil {
			wg.Done()
			resultCh <- fileResult{err: fmt.Errorf("pool submit error: %w", err)}
		}

		// 每 10 个文件保存一次进度
		if (i+1)%10 == 0 || i == len(filesToProcess)-1 {
			progress.LastFileIndex = startIndex + i + 1
			progress.LastFilePath = file
			progress.UpdatedAt = time.Now()

			// 汇总当前结果
			for j := 0; j < len(resultCh); j++ {
				select {
				case res := <-resultCh:
					if res.indexed {
						result.Indexed++
						progress.IndexedFiles++
					}
					if res.skipped {
						result.Skipped++
					}
					if res.err != nil {
						result.Failed++
						progress.FailedFiles++
					}
				default:
					break
				}
			}

			if err := idx.storage.SaveIndexProgress(progress); err != nil {
				log.Printf("Warning: failed to save index progress: %v", err)
			}
		}
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	for res := range resultCh {
		if res.indexed {
			result.Indexed++
			progress.IndexedFiles++
		}
		if res.skipped {
			result.Skipped++
		}
		if res.err != nil {
			result.Failed++
			progress.FailedFiles++
		}
		metrics.IndexTotal.Inc()
	}

	// 标记完成
	progress.Status = "completed"
	progress.CompletedAt = time.Now()
	progress.UpdatedAt = time.Now()
	idx.storage.SaveIndexProgress(progress)

	result.Duration = time.Since(start).Milliseconds()
	return result, nil
}

// IndexDirectory 遍历执行整个文件夹（并发优化，不支持断点恢复，用户隔离）
func (idx *Indexer) IndexDirectory(rootPath string, userID string) (*IndexResult, error) {
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
		currentUserID := userID // 闭包捕获
		err := idx.pool.Submit(func() {
			defer wg.Done()
			indexed, skipped, err := idx.indexFileInternalWithUser(file, currentUserID)
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

// IndexFile 解析单一文件（暴露给Watcher使用，内部调用，用户隔离）
func (idx *Indexer) IndexFile(path string, userID string) (bool, bool, error) {
	return idx.indexFileInternalWithUser(path, userID)
}

// indexFileInternal 实际索引逻辑（无用户隔离，用于向后兼容）
func (idx *Indexer) indexFileInternal(path string) (bool, bool, error) {
	return idx.indexFileInternalWithUser(path, "")
}

// indexFileInternalWithUser 实际索引逻辑（用户隔离）
func (idx *Indexer) indexFileInternalWithUser(path string, userID string) (bool, bool, error) {
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
	} else if strings.HasSuffix(pathLower, ".go") {
		fileType = "go"
	} else if strings.HasSuffix(pathLower, ".yaml") || strings.HasSuffix(pathLower, ".yml") {
		fileType = "yaml"
	} else if strings.HasSuffix(pathLower, ".json") {
		fileType = "json"
	} else if strings.HasSuffix(pathLower, ".txt") {
		fileType = "txt"
	} else if strings.HasSuffix(pathLower, ".toml") {
		fileType = "toml"
	} else if strings.HasSuffix(pathLower, ".ini") {
		fileType = "ini"
	} else if strings.HasSuffix(pathLower, ".cfg") || strings.HasSuffix(pathLower, ".conf") {
		fileType = "cfg"
	} else if strings.HasSuffix(pathLower, ".hcl") {
		fileType = "hcl"
	} else if strings.HasSuffix(pathLower, ".env") {
		fileType = "env"
	} else if strings.HasSuffix(pathLower, ".properties") {
		fileType = "properties"
	} else if strings.HasSuffix(pathLower, ".xml") {
		fileType = "xml"
	} else if strings.HasSuffix(pathLower, ".html") || strings.HasSuffix(pathLower, ".htm") {
		fileType = "html"
	} else if strings.HasSuffix(pathLower, ".css") {
		fileType = "css"
	} else if strings.HasSuffix(pathLower, ".js") {
		fileType = "js"
	} else if strings.HasSuffix(pathLower, ".ts") {
		fileType = "ts"
	} else if strings.HasSuffix(pathLower, ".py") {
		fileType = "py"
	} else if strings.HasSuffix(pathLower, ".rb") {
		fileType = "rb"
	} else if strings.HasSuffix(pathLower, ".java") {
		fileType = "java"
	} else if strings.HasSuffix(pathLower, ".cpp") || strings.HasSuffix(pathLower, ".cc") || strings.HasSuffix(pathLower, ".cxx") {
		fileType = "cpp"
	} else if strings.HasSuffix(pathLower, ".c") || strings.HasSuffix(pathLower, ".h") {
		fileType = "c"
	} else if strings.HasSuffix(pathLower, ".sh") || strings.HasSuffix(pathLower, ".bash") || strings.HasSuffix(pathLower, ".zsh") {
		fileType = "sh"
	} else if strings.HasSuffix(pathLower, ".ps1") {
		fileType = "ps1"
	}

	ck, ok := idx.chunkers[fileType]
	if !ok {
		return false, true, fmt.Errorf("unsupported file type: %s", path)
	}

	hashBytes := sha256.Sum256(content)
	hashStr := hex.EncodeToString(hashBytes[:])
	docID := hashStr[:16]

	// 查询是否存在及比对Hash（用户隔离）
	doc, _ := idx.storage.GetDocumentByPath(path, userID)
	if doc != nil && doc.ContentHash == hashStr {
		// 跳过重复索引
		return false, true, nil
	}

	if doc != nil {
		idx.storage.DeleteChunksByDocument(doc.ID, userID)
	}

	// 开始文本切块解析
	chunks, err := ck.Chunk(string(content), path)
	if err != nil || len(chunks) == 0 {
		return false, false, err
	}

	// 设置 userID 和 documentID
	for i, c := range chunks {
		c.UserID = userID
		c.DocumentID = docID
	}

	// 转换为向量
	texts := make([]string, len(chunks))
	for i, c := range chunks {
		texts[i] = c.ContentRaw
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

	// 保存 Document（用户隔离）
	newDoc := &models.Document{
		ID:          docID,
		UserID:      userID,
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

// GetIndexProgress 获取当前索引进度
func (idx *Indexer) GetIndexProgress(rootPath string) (*models.IndexProgress, error) {
	return idx.storage.GetIndexProgress(rootPath)
}

// ==============================================
// 增量索引器 (IncrementalIndexer)
// ==============================================

// IncrementalIndexer 增量索引器 - 用于定期增量同步
type IncrementalIndexer struct {
	indexer  *Indexer
	states   map[string]*FileState // path -> state
	mu       sync.RWMutex
	rootPath string
	userID   string
}

// FileState 文件状态（用于增量比对）
type FileState struct {
	ModTime     time.Time
	ContentHash string
	IndexedAt   time.Time
}

// NewIncrementalIndexer 创建增量索引器
func NewIncrementalIndexer(idx *Indexer, rootPath string, userID string) *IncrementalIndexer {
	return &IncrementalIndexer{
		indexer:  idx,
		states:   make(map[string]*FileState),
		rootPath: rootPath,
		userID:   userID,
	}
}

// ScanDirectory 扫描目录，返回需要索引的文件列表
func (ii *IncrementalIndexer) ScanDirectory() ([]string, error) {
	var files []string
	err := filepath.Walk(ii.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files, err
}

// Sync 执行增量同步
// 返回: added/updated/removed/total
func (ii *IncrementalIndexer) Sync() (added, updated, removed, total int, err error) {
	files, err := ii.ScanDirectory()
	if err != nil {
		return 0, 0, 0, 0, err
	}

	currentFiles := make(map[string]bool)
	var changed bool

	for _, path := range files {
		currentFiles[path] = true

		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		hashBytes := sha256.Sum256(content)
		hashStr := hex.EncodeToString(hashBytes[:])

		ii.mu.Lock()
		oldState, exists := ii.states[path]
		ii.mu.Unlock()

		if !exists || oldState.ContentHash != hashStr {
			// 新文件或内容已更改
			indexed, _, err := ii.indexer.IndexFile(path, ii.userID)
			if err != nil {
				continue
			}
			if indexed {
				if exists {
					updated++
				} else {
					added++
				}
				changed = true

				ii.mu.Lock()
				ii.states[path] = &FileState{
					ModTime:     time.Now(),
					ContentHash: hashStr,
					IndexedAt:   time.Now(),
				}
				ii.mu.Unlock()
			}
		}
	}

	// 检测已删除的文件
	ii.mu.Lock()
	for path := range ii.states {
		if !currentFiles[path] {
			// 文件已删除
			err := ii.indexer.storage.DeleteDocumentByPath(path, ii.userID)
			if err == nil {
				removed++
				changed = true
				delete(ii.states, path)
			}
		}
	}
	ii.mu.Unlock()

	total = len(files)
	return added, updated, removed, total, nil
}

// GetStats 获取增量索引器状态
func (ii *IncrementalIndexer) GetStats() (tracked int) {
	ii.mu.RLock()
	defer ii.mu.RUnlock()
	return len(ii.states)
}