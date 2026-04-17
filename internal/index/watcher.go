package index

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// StartWatcher 启动增量索引守护线程
func (idx *Indexer) StartWatcher(root string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	// 遍历并监听该级目录下所有子目录
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() {
			
			// 过滤通常不需要监听的目录
			if strings.Contains(path, ".git") || strings.Contains(path, "node_modules") {
				return filepath.SkipDir
			}
			
			return watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	log.Printf("Watcher started on %s. Waiting for file changes...", root)

	// 简单防抖(Debounce)字典
	debounceMap := make(map[string]time.Time)
	debounceDur := 2 * time.Second

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			// 只对写入或创建事件响应
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				// 文件过滤器
				if !strings.HasSuffix(strings.ToLower(event.Name), ".md") {
					continue
				}

				lastAction, exists := debounceMap[event.Name]
				if exists && time.Since(lastAction) < debounceDur {
					continue
				}
				debounceMap[event.Name] = time.Now()

				log.Printf("[Watcher] Detected change in %s. Re-indexing...", event.Name)
				indexed, _, err := idx.IndexFile(event.Name)
				if err != nil {
					log.Printf("[Watcher Error] Failed to index %s: %v", event.Name, err)
				} else if indexed {
					log.Printf("[Watcher Success] Updated index for %s", event.Name)
				}
			}

			if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				if strings.HasSuffix(strings.ToLower(event.Name), ".md") {
					log.Printf("[Watcher] Detected removal of %s. Deleting from index...", event.Name)
					idx.storage.DeleteDocumentByPath(event.Name)
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Printf("[Watcher Error] fsnotify issue: %v", err)
		}
	}
}
