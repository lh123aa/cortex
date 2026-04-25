package index

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// IncrementalWatcher 增量索引监视器
type IncrementalWatcher struct {
	watcher     *fsnotify.Watcher
	indexer     *Indexer
	rootPath    string
	userID      string
	fileStates  map[string]*FileState // path -> state
	mu          sync.RWMutex
	debounceDur time.Duration
	extensions  map[string]bool // 支持的文件扩展名
	shutdownCh  chan struct{}
}

// NewIncrementalWatcher 创建增量监视器
func NewIncrementalWatcher(idx *Indexer, root string, userID string) (*IncrementalWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	iw := &IncrementalWatcher{
		watcher:     watcher,
		indexer:     idx,
		rootPath:    root,
		userID:      userID,
		fileStates:  make(map[string]*FileState),
		debounceDur: 2 * time.Second,
		extensions:  make(map[string]bool),
		shutdownCh:  make(chan struct{}),
	}

	// 初始化支持的文件扩展名
	supportedExts := []string{
		".md", ".pdf", ".docx", ".go", ".yaml", ".yml", ".json", ".txt",
		".toml", ".ini", ".cfg", ".conf", ".hcl", ".env", ".properties",
		".xml", ".html", ".htm", ".css", ".js", ".ts", ".py", ".rb",
		".java", ".cpp", ".cc", ".cxx", ".c", ".h", ".sh", ".bash",
		".zsh", ".ps1",
	}
	for _, ext := range supportedExts {
		iw.extensions[ext] = true
	}

	return iw, nil
}

// Start 开始监听
func (iw *IncrementalWatcher) Start() error {
	// 遍历并监听该级目录下所有子目录
	err := filepath.Walk(iw.rootPath, func(path string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() {
			// 过滤通常不需要监听的目录
			if strings.Contains(path, ".git") ||
				strings.Contains(path, "node_modules") ||
				strings.Contains(path, ".venv") ||
				strings.Contains(path, "vendor") {
				return filepath.SkipDir
			}
			return iw.watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	log.Printf("[IncrementalWatcher] Started on %s (user: %s)", iw.rootPath, iw.userID)

	// 启动事件处理循环
	go iw.runLoop()

	return nil
}

// runLoop 事件处理循环
func (iw *IncrementalWatcher) runLoop() {
	debounceMap := make(map[string]time.Time)

	for {
		select {
		case <-iw.shutdownCh:
			log.Printf("[IncrementalWatcher] Shutting down...")
			return

		case event, ok := <-iw.watcher.Events:
			if !ok {
				return
			}
			iw.handleEvent(event, debounceMap)

		case err, ok := <-iw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("[IncrementalWatcher] Error: %v", err)
		}
	}
}

// handleEvent 处理单个文件事件
func (iw *IncrementalWatcher) handleEvent(event fsnotify.Event, debounceMap map[string]time.Time) {
	// 获取文件扩展名
	ext := strings.ToLower(filepath.Ext(event.Name))

	// 检查是否是支持的文件类型
	if !iw.extensions[ext] {
		return
	}

	// 防抖处理
	lastAction, exists := debounceMap[event.Name]
	if exists && time.Since(lastAction) < iw.debounceDur {
		return
	}
	debounceMap[event.Name] = time.Now()

	// 处理不同事件类型
	switch {
	case event.Has(fsnotify.Write), event.Has(fsnotify.Create):
		iw.handleWriteOrCreate(event.Name)
	case event.Has(fsnotify.Remove), event.Has(fsnotify.Rename):
		iw.handleRemove(event.Name)
	}
}

// handleWriteOrCreate 处理写入或创建事件
func (iw *IncrementalWatcher) handleWriteOrCreate(path string) {
	log.Printf("[IncrementalWatcher] Detected change in %s", path)

	// 计算内容hash
	content, err := os.ReadFile(path)
	if err != nil {
		log.Printf("[IncrementalWatcher] Failed to read file %s: %v", path, err)
		return
	}

	hashBytes := sha256.Sum256(content)
	hashStr := hex.EncodeToString(hashBytes[:])

	// 检查是否有变化
	iw.mu.RLock()
	oldState, exists := iw.fileStates[path]
	iw.mu.RUnlock()

	if exists && oldState.ContentHash == hashStr {
		log.Printf("[IncrementalWatcher] File unchanged, skipping: %s", path)
		return
	}

	// 增量索引
	indexed, _, err := iw.indexer.IndexFile(path, iw.userID)
	if err != nil {
		log.Printf("[IncrementalWatcher] Failed to index %s: %v", path, err)
		return
	}

	if indexed {
		log.Printf("[IncrementalWatcher] Indexed: %s", path)

		// 更新状态
		iw.mu.Lock()
		iw.fileStates[path] = &FileState{
			ModTime:     time.Now(),
			ContentHash: hashStr,
			IndexedAt:   time.Now(),
		}
		iw.mu.Unlock()
	} else {
		log.Printf("[IncrementalWatcher] Skipped (unchanged): %s", path)
	}
}

// handleRemove 处理删除事件
func (iw *IncrementalWatcher) handleRemove(path string) {
	log.Printf("[IncrementalWatcher] Detected removal: %s", path)

	// 从存储中删除
	err := iw.indexer.storage.DeleteDocumentByPath(path, iw.userID)
	if err != nil {
		log.Printf("[IncrementalWatcher] Failed to delete %s: %v", path, err)
	}

	// 从状态缓存中移除
	iw.mu.Lock()
	delete(iw.fileStates, path)
	iw.mu.Unlock()
}

// Stop 停止监听
func (iw *IncrementalWatcher) Stop() error {
	close(iw.shutdownCh)
	return iw.watcher.Close()
}

// GetStats 获取监视器状态统计
func (iw *IncrementalWatcher) GetStats() (watching int, indexed int) {
	iw.mu.RLock()
	defer iw.mu.RUnlock()
	return len(iw.fileStates), len(iw.fileStates)
}

// SupportedExtensions 返回支持的文件扩展名
func (iw *IncrementalWatcher) SupportedExtensions() []string {
	exts := make([]string, 0, len(iw.extensions))
	for ext := range iw.extensions {
		exts = append(exts, ext)
	}
	return exts
}

// StartWatcher 启动增量索引守护线程（兼容旧接口）
func (idx *Indexer) StartWatcher(root string, userID string) error {
	iw, err := NewIncrementalWatcher(idx, root, userID)
	if err != nil {
		return err
	}
	return iw.Start()
}
