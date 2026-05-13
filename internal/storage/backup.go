package storage

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// BackupManager 负责安全的热机拷贝
type BackupManager struct {
	dbPath    string
	backupDir string
	maxKeep   int       // 保留的最大备份数，0=不限制
	stopCh    chan struct{}
}

func NewBackupManager(dbPath string) *BackupManager {
	dir := filepath.Dir(dbPath)
	return &BackupManager{
		dbPath:    dbPath,
		backupDir: filepath.Join(dir, "backups"),
		maxKeep:   10,
		stopCh:    make(chan struct{}),
	}
}

// SetMaxBackups 设置保留的最大备份数
func (b *BackupManager) SetMaxBackups(n int) {
	if n > 0 {
		b.maxKeep = n
	}
}

// CreateBackup 将数据库复制一份带有时间戳的备份
func (b *BackupManager) CreateBackup() (string, error) {
	if err := os.MkdirAll(b.backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup dir: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	backupPath := filepath.Join(b.backupDir, fmt.Sprintf("cortex_%s.db", timestamp))

	if err := copyFile(b.dbPath, backupPath); err != nil {
		return "", err
	}

	if walInfo, err := os.Stat(b.dbPath + "-wal"); err == nil && !walInfo.IsDir() {
		if err := copyFile(b.dbPath+"-wal", backupPath+"-wal"); err != nil {
			log.Printf("Warning: failed to copy WAL file during backup: %v", err)
		}
	}

	// 清理过期备份
	b.cleanupOldBackups()

	return backupPath, nil
}

// StartAutoBackup 启动定时自动备份（每 interval 执行一次）
func (b *BackupManager) StartAutoBackup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		log.Printf("[AutoBackup] started, interval=%v, dir=%s", interval, b.backupDir)

		for {
			select {
			case <-ticker.C:
				path, err := b.CreateBackup()
				if err != nil {
					log.Printf("[AutoBackup] failed: %v", err)
				} else {
					log.Printf("[AutoBackup] created: %s", path)
				}
			case <-b.stopCh:
				log.Printf("[AutoBackup] stopped")
				return
			}
		}
	}()
}

// StopAutoBackup 停止定时自动备份
func (b *BackupManager) StopAutoBackup() {
	close(b.stopCh)
}

// cleanupOldBackups 清理超过 maxKeep 的旧备份
func (b *BackupManager) cleanupOldBackups() {
	if b.maxKeep <= 0 {
		return
	}

	entries, err := os.ReadDir(b.backupDir)
	if err != nil {
		return
	}

	var backups []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".db" {
			backups = append(backups, filepath.Join(b.backupDir, e.Name()))
		}
	}

	if len(backups) <= b.maxKeep {
		return
	}

	// 按修改时间排序，删除最旧的
	for i := 0; i < len(backups)-b.maxKeep; i++ {
		os.Remove(backups[i])
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
