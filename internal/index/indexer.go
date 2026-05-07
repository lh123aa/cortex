package index

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lh123aa/cortex/internal/chunker"
	"github.com/lh123aa/cortex/internal/embedding"
	"github.com/lh123aa/cortex/internal/metrics"
	"github.com/lh123aa/cortex/internal/models"
	"github.com/lh123aa/cortex/internal/storage"
	"github.com/panjf2000/ants/v2"
)

// perFileEmbeddingTimeout 单个文件的 embedding 超时时间（防止卡死）
const perFileEmbeddingTimeout = 5 * time.Minute

// Indexer 统筹负责调度文件提取、分块、向量与存储
type Indexer struct {
	storage   storage.Storage
	chunkers  map[string]chunker.Chunker
	embedding embedding.EmbeddingProvider
	pool      *ants.Pool
	logger    *log.Logger // 结构化日志（可选）

	// OnProgress 可选回调，每个文件处理完成后触发（用于实时进度展示）
	OnProgress func(evt models.IndexProgressEvent)
}

// SetLogger 设置日志记录器
func (idx *Indexer) SetLogger(l *log.Logger) {
	idx.logger = l
}

// logWarn 日志回退：优先使用结构日志，否则标准 log
func (idx *Indexer) logWarn(msg string, args ...interface{}) {
	if idx.logger != nil {
		idx.logger.Printf(msg, args...)
	} else {
		log.Printf(msg, args...)
	}
}

// NewIndexer 初始化索引器（workers 从配置读取，默认 16）
func NewIndexer(s storage.Storage, em embedding.EmbeddingProvider, workers int) (*Indexer, error) {
	if workers <= 0 {
		workers = 16 // 默认值（I/O 密集型，更多 workers 提升吞吐量）
	}
	ckMap := make(map[string]chunker.Chunker)
	mk, _ := chunker.NewMarkdownChunker(chunker.ChunkConfig{
		MinChars:          50,
		MaxTokens:         512,
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

	// P2-2: 初始化 goroutine pool（默认 8 个 worker，提升吞吐量）
	p, err := ants.NewPool(workers, ants.WithPreAlloc(false))
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

type fileResult struct {
	path    string
	indexed bool
	skipped bool
	err     error
}

// 默认排除目录：这些目录的内容通常不产生有意义的搜索语义
var defaultExcludeDirs = map[string]bool{
	"node_modules": true,
	".git":         true,
	".opencode":    true,
	".svn":         true,
	"__pycache__":  true,
	".cache":       true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	".next":        true,
	"coverage":     true,
	".idea":        true,
	".vscode":      true,
	"WeChat Files": true,
	"Applet":       true,
	"FileStorage":  true,
	"__GAME_FILE_CACHE": true,
}

// isExcludedDir 检查目录是否应被排除
func isExcludedDir(name string) bool {
	return defaultExcludeDirs[name]
}

// IndexDirectoryWithCheckpoint 遍历执行整个文件夹（支持断点恢复，用户隔离）
// ctx 用于超时控制和优雅退出，传入 context.Background() 则不限制
func (idx *Indexer) IndexDirectoryWithCheckpoint(ctx context.Context, rootPath string, userID string) (*IndexResult, error) {
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

	// 路径规范化，防止路径遍历攻击
	rootPath = filepath.Clean(rootPath)

	// 第一阶段 — 收集所有文件路径
	var allFiles []string
	err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if isExcludedDir(info.Name()) {
				return filepath.SkipDir
			}
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
		result.Skipped = 0 // IndexProgress 不跟踪 Skipped
		result.Failed = progress.FailedFiles
		idx.storage.CompleteIndexProgress(rootPath)
		return result, nil
	}

	// 处理剩余文件
	filesToProcess := allFiles[startIndex:]
	resultCh := make(chan fileResult, len(filesToProcess))
	var wg sync.WaitGroup

	for _, file := range filesToProcess {
		// 检查全局是否已取消
		select {
		case <-ctx.Done():
			goto collectResultsWithCheckpoint
		default:
		}

		wg.Add(1)
		currentUserID := userID
		filePath := file
		err := idx.pool.Submit(func() {
			defer wg.Done()
			// goroutine 内部再次检查取消
			select {
			case <-ctx.Done():
				resultCh <- fileResult{path: filePath, err: ctx.Err()}
				return
			default:
			}
			indexed, skipped, err := idx.indexFileInternalWithUser(ctx, filePath, currentUserID)
			resultCh <- fileResult{path: filePath, indexed: indexed, skipped: skipped, err: err}
		})
		if err != nil {
			wg.Done()
			resultCh <- fileResult{path: filePath, err: fmt.Errorf("pool submit error: %w", err)}
		}
	}

collectResultsWithCheckpoint:
	// 等所有 goroutine 完成后收集结果
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// 使用原子计数器统计已完成文件
	var atomicIndexed atomic.Int32
	var atomicSkipped atomic.Int32
	var atomicFailed atomic.Int32

	// 记录 baseline，防止 checkpoint 累计计数 bug
	baselineIndexed := progress.IndexedFiles
	baselineFailed := progress.FailedFiles

	completedCount := 0
	checkpointTicker := time.NewTicker(5 * time.Second) // 每 5 秒保存一次 checkpoint
	defer checkpointTicker.Stop()

	progressStart := start
	for res := range resultCh {
		if res.indexed {
			atomicIndexed.Add(1)
		}
		if res.skipped {
			atomicSkipped.Add(1)
		}
		if res.err != nil {
			atomicFailed.Add(1)
		}
		metrics.IndexTotal.Inc()
		completedCount++

		// 发送实时进度事件
		if idx.OnProgress != nil {
			indexed := int(atomicIndexed.Load())
			skipped := int(atomicSkipped.Load())
			failed := int(atomicFailed.Load())
			elapsed := time.Since(progressStart)
			speed := float64(completedCount) / elapsed.Seconds()
			var eta time.Duration
			if speed > 0.01 && completedCount < len(filesToProcess) {
				remaining := float64(len(filesToProcess) - completedCount)
				eta = time.Duration(remaining/speed) * time.Second
			}
			idx.OnProgress(models.IndexProgressEvent{
				Total:       result.Total,
				Indexed:     indexed,
				Skipped:     skipped,
				Failed:      failed,
				CurrentFile: res.path,
				Speed:       speed,
				Elapsed:     elapsed,
				ETA:         eta,
				Done:        completedCount >= len(filesToProcess),
			})
		}

		// 定时保存 checkpoint（不阻塞每次循环）
		select {
		case <-checkpointTicker.C:
			progress.LastFileIndex = startIndex + completedCount
			progress.LastFilePath = res.path
			progress.UpdatedAt = time.Now()
			progress.IndexedFiles = baselineIndexed + int(atomicIndexed.Load())
			progress.FailedFiles = baselineFailed + int(atomicFailed.Load())
			if err := idx.storage.SaveIndexProgress(progress); err != nil {
				idx.logWarn("Warning: failed to save index progress: %v", err)
			}
		default:
		}
	}

	// 汇总最终结果
	result.Indexed = int(atomicIndexed.Load())
	result.Skipped = int(atomicSkipped.Load())
	result.Failed = int(atomicFailed.Load())
	progress.IndexedFiles = baselineIndexed + result.Indexed
	progress.FailedFiles = baselineFailed + result.Failed

	// 标记完成
	progress.Status = "completed"
	progress.CompletedAt = time.Now()
	progress.UpdatedAt = time.Now()
	idx.storage.SaveIndexProgress(progress)

	result.Duration = time.Since(start).Milliseconds()
	return result, nil
}

// IndexDirectory 遍历执行整个文件夹（并发优化，不支持断点恢复，用户隔离）
// ctx 用于超时控制和优雅退出，传入 context.Background() 则不限制
func (idx *Indexer) IndexDirectory(ctx context.Context, rootPath string, userID string) (*IndexResult, error) {
	start := time.Now()
	result := &IndexResult{}

	// 路径规范化，防止路径遍历攻击
	rootPath = filepath.Clean(rootPath)

	// P2-2: 第一阶段 — 收集所有文件路径
	var files []string
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if isExcludedDir(info.Name()) {
				return filepath.SkipDir
			}
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
	resultCh := make(chan fileResult, len(files))
	var wg sync.WaitGroup

	for _, file := range files {
		// 检查全局是否已取消
		select {
		case <-ctx.Done():
			// 跳过剩余文件，直接进入收集阶段
			goto collectResults
		default:
		}

		wg.Add(1)
		currentUserID := userID // 闭包捕获
		filePath := file
		err := idx.pool.Submit(func() {
			defer wg.Done()
			// goroutine 内部再次检查取消
			select {
			case <-ctx.Done():
				resultCh <- fileResult{path: filePath, err: ctx.Err()}
				return
			default:
			}
			indexed, skipped, err := idx.indexFileInternalWithUser(ctx, filePath, currentUserID)
			resultCh <- fileResult{path: filePath, indexed: indexed, skipped: skipped, err: err}
		})
		if err != nil {
			wg.Done()
			resultCh <- fileResult{path: filePath, err: fmt.Errorf("pool submit error: %w", err)}
		}
	}

collectResults:
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	progStart := start
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

		// 发送进度事件
		if idx.OnProgress != nil {
			completed := result.Indexed + result.Skipped + result.Failed
			elapsed := time.Since(progStart)
			speed := float64(completed) / elapsed.Seconds()
			var eta time.Duration
			if speed > 0.01 && completed < result.Total {
				remaining := float64(result.Total - completed)
				eta = time.Duration(remaining/speed) * time.Second
			}
			idx.OnProgress(models.IndexProgressEvent{
				Total:       result.Total,
				Indexed:     result.Indexed,
				Skipped:     result.Skipped,
				Failed:      result.Failed,
				CurrentFile: res.path,
				Speed:       speed,
				Elapsed:     elapsed,
				ETA:         eta,
				Done:        completed >= result.Total,
			})
		}
	}

	result.Duration = time.Since(start).Milliseconds()
	return result, nil
}

// IndexFile 解析单一文件（暴露给Watcher使用，向后兼容）
func (idx *Indexer) IndexFile(path string, userID string) (bool, bool, error) {
	return idx.indexFileInternalWithUser(context.Background(), path, userID)
}

// indexFileInternal 实际索引逻辑（无用户隔离，用于向后兼容）
func (idx *Indexer) indexFileInternal(path string) (bool, bool, error) {
	return idx.indexFileInternalWithUser(context.Background(), path, "")
}

// indexFileInternalWithUser 实际索引逻辑（用户隔离，带 Context 超时控制）
func (idx *Indexer) indexFileInternalWithUser(ctx context.Context, path string, userID string) (bool, bool, error) {
	// 检查文件大小，防止 OOM（限制 100MB）
	info, err := os.Stat(path)
	if err != nil {
		return false, false, fmt.Errorf("failed to stat file %s: %w", path, err)
	}
	const maxFileSize int64 = 100 * 1024 * 1024 // 100MB
	if info.Size() > maxFileSize {
		return false, true, fmt.Errorf("file too large (%d bytes, max %d bytes): %s", info.Size(), maxFileSize, path)
	}

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
	for _, c := range chunks {
		c.UserID = userID
		c.DocumentID = docID
	}

	// 去重：计算每个 chunk 的内容哈希，跳过已存在的重复 chunk
	var deduped []*models.Chunk
	for _, c := range chunks {
		hashBytes := sha256.Sum256([]byte(c.ContentRaw))
		c.ContentHash = hex.EncodeToString(hashBytes[:])

		existing, err := idx.storage.GetChunkByHash(c.ContentHash, userID)
		if err == nil && existing != nil {
			// 已存在完全相同的 chunk，跳过
			continue
		}
		deduped = append(deduped, c)
	}
	chunks = deduped

	if len(chunks) == 0 {
		// 所有 chunk 都是重复的，但文档本身可能已存在，标记为已索引
		return true, false, nil
	}

	// 转换为向量
	texts := make([]string, len(chunks))
	for i, c := range chunks {
		texts[i] = c.ContentRaw
	}

	// 进行Embedding（带 per-file 超时保护，防止卡死）
	if idx.embedding != nil {
		embedCtx, embedCancel := context.WithTimeout(ctx, perFileEmbeddingTimeout)
		done := make(chan struct{}, 1)
		var embeddings [][]float32
		var embedErr error

		go func() {
			embeddings, embedErr = idx.embedding.EmbedBatch(texts)
			close(done)
		}()

		select {
		case <-done:
			embedCancel()
			if embedErr != nil {
				idx.logWarn("Warning: embedding failed for %s (indexing continues without vectors): %v", path, embedErr)
			} else {
				for j, c := range chunks {
					c.Embedding = embeddings[j]
					c.EmbeddingModel = idx.embedding.Name()
				}
			}
		case <-embedCtx.Done():
			embedCancel()
			idx.logWarn("Warning: embedding timed out for %s (indexing continues without vectors)", path)
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
	// 路径规范化
	ii.rootPath = filepath.Clean(ii.rootPath)
	err := filepath.Walk(ii.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if isExcludedDir(info.Name()) {
				return filepath.SkipDir
			}
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
