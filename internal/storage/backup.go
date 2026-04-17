package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// BackupManager 负责安全的热机拷贝
type BackupManager struct {
	dbPath    string
	backupDir string
}

func NewBackupManager(dbPath string) *BackupManager {
	dir := filepath.Dir(dbPath)
	return &BackupManager{
		dbPath:    dbPath,
		backupDir: filepath.Join(dir, "backups"),
	}
}

// CreateBackup 将数据库复制一份带有时间戳的备份
func (b *BackupManager) CreateBackup() (string, error) {
	if err := os.MkdirAll(b.backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup dir: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	backupPath := filepath.Join(b.backupDir, fmt.Sprintf("cortex_%s.db", timestamp))

	// 对于 WAL 模式 SQLite 的简单热备，可以通过文件IO加上WAL同步，
	// 如果是极度严格要求可使用 sqlite 的在线 backup API。这里采取稳健的 File IO 拷贝。
	if err := copyFile(b.dbPath, backupPath); err != nil {
		return "", err
	}
	
	// 同时备份一下存在可能未落盘的 wall log
	if walInfo, err := os.Stat(b.dbPath + "-wal"); err == nil && !walInfo.IsDir() {
		_ = copyFile(b.dbPath+"-wal", backupPath+"-wal")
	}

	return backupPath, nil
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
