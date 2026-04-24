package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/lh123aa/cortex/internal/models"
)

// IndexProgress CRUD

// SaveIndexProgress 保存或更新索引进度
func (s *SQLiteStorage) SaveIndexProgress(p *models.IndexProgress) error {
	query := `
		INSERT OR REPLACE INTO index_progress
		(id, root_path, last_file_path, last_file_index, total_files, indexed_files, indexed_chunks, failed_files, status, started_at, updated_at, completed_at, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	var completedAt interface{}
	if !p.CompletedAt.IsZero() {
		completedAt = p.CompletedAt
	}
	var errorMsg interface{}
	if p.ErrorMessage != "" {
		errorMsg = p.ErrorMessage
	}

	_, err := s.db.Exec(query,
		p.ID, p.RootPath, p.LastFilePath, p.LastFileIndex, p.TotalFiles,
		p.IndexedFiles, p.IndexedChunks, p.FailedFiles, p.Status,
		p.StartedAt, p.UpdatedAt, completedAt, errorMsg,
	)
	return err
}

// GetIndexProgress 获取索引进度（按 root_path 查询进行中的）
func (s *SQLiteStorage) GetIndexProgress(rootPath string) (*models.IndexProgress, error) {
	row := s.db.QueryRow(`
		SELECT id, root_path, last_file_path, last_file_index, total_files, indexed_files, indexed_chunks, failed_files, status, started_at, updated_at, completed_at, error_message
		FROM index_progress
		WHERE root_path = ? AND status = 'running'
		ORDER BY updated_at DESC LIMIT 1
	`, rootPath)

	var p models.IndexProgress
	var completedAt, errorMsg sql.NullTime
	var lastFilePath sql.NullString
	var lastFileIndex, totalFiles, indexedFiles, indexedChunks, failedFiles sql.NullInt64

	err := row.Scan(
		&p.ID, &p.RootPath, &lastFilePath, &lastFileIndex, &totalFiles,
		&indexedFiles, &indexedChunks, &failedFiles, &p.Status,
		&p.StartedAt, &p.UpdatedAt, &completedAt, &errorMsg,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if lastFilePath.Valid {
		p.LastFilePath = lastFilePath.String
	}
	if lastFileIndex.Valid {
		p.LastFileIndex = int(lastFileIndex.Int64)
	}
	if totalFiles.Valid {
		p.TotalFiles = int(totalFiles.Int64)
	}
	if indexedFiles.Valid {
		p.IndexedFiles = int(indexedFiles.Int64)
	}
	if indexedChunks.Valid {
		p.IndexedChunks = int(indexedChunks.Int64)
	}
	if failedFiles.Valid {
		p.FailedFiles = int(failedFiles.Int64)
	}
	if completedAt.Valid {
		p.CompletedAt = completedAt.Time
	}
	if errorMsg.Valid {
		p.ErrorMessage = errorMsg.String
	}

	return &p, nil
}

// ListIndexProgress 查询所有索引进度
func (s *SQLiteStorage) ListIndexProgress(limit, offset int) ([]*models.IndexProgress, error) {
	rows, err := s.db.Query(`
		SELECT id, root_path, last_file_path, last_file_index, total_files, indexed_files, indexed_chunks, failed_files, status, started_at, updated_at, completed_at, error_message
		FROM index_progress
		ORDER BY updated_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.IndexProgress
	for rows.Next() {
		var p models.IndexProgress
		var completedAt, errorMsg sql.NullTime
		var lastFilePath sql.NullString
		var lastFileIndex, totalFiles, indexedFiles, indexedChunks, failedFiles sql.NullInt64

		err := rows.Scan(
			&p.ID, &p.RootPath, &lastFilePath, &lastFileIndex, &totalFiles,
			&indexedFiles, &indexedChunks, &failedFiles, &p.Status,
			&p.StartedAt, &p.UpdatedAt, &completedAt, &errorMsg,
		)
		if err != nil {
			return nil, err
		}

		if lastFilePath.Valid {
			p.LastFilePath = lastFilePath.String
		}
		if lastFileIndex.Valid {
			p.LastFileIndex = int(lastFileIndex.Int64)
		}
		if totalFiles.Valid {
			p.TotalFiles = int(totalFiles.Int64)
		}
		if indexedFiles.Valid {
			p.IndexedFiles = int(indexedFiles.Int64)
		}
		if indexedChunks.Valid {
			p.IndexedChunks = int(indexedChunks.Int64)
		}
		if failedFiles.Valid {
			p.FailedFiles = int(failedFiles.Int64)
		}
		if completedAt.Valid {
			p.CompletedAt = completedAt.Time
		}
		if errorMsg.Valid {
			p.ErrorMessage = errorMsg.String
		}

		results = append(results, &p)
	}
	return results, nil
}

// DeleteIndexProgress 删除索引进度记录
func (s *SQLiteStorage) DeleteIndexProgress(id int) error {
	_, err := s.db.Exec(`DELETE FROM index_progress WHERE id = ?`, id)
	return err
}

// CompleteIndexProgress 标记索引完成
func (s *SQLiteStorage) CompleteIndexProgress(rootPath string) error {
	now := time.Now()
	_, err := s.db.Exec(`
		UPDATE index_progress
		SET status = 'completed', updated_at = ?, completed_at = ?
		WHERE root_path = ? AND status = 'running'
	`, now, now, rootPath)
	return err
}

// FailIndexProgress 标记索引失败
func (s *SQLiteStorage) FailIndexProgress(rootPath string, errMsg string) error {
	now := time.Now()
	_, err := s.db.Exec(`
		UPDATE index_progress
		SET status = 'failed', updated_at = ?, error_message = ?
		WHERE root_path = ? AND status = 'running'
	`, now, errMsg, rootPath)
	return err
}

// InitIndexProgressTable 初始化 index_progress 表
func (s *SQLiteStorage) InitIndexProgressTable() error {
	schema := `
	CREATE TABLE IF NOT EXISTS index_progress (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		root_path TEXT NOT NULL,
		last_file_path TEXT DEFAULT '',
		last_file_index INTEGER DEFAULT 0,
		total_files INTEGER DEFAULT 0,
		indexed_files INTEGER DEFAULT 0,
		indexed_chunks INTEGER DEFAULT 0,
		failed_files INTEGER DEFAULT 0,
		status TEXT DEFAULT 'running',
		started_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		completed_at TIMESTAMP,
		error_message TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_progress_root ON index_progress(root_path);
	CREATE INDEX IF NOT EXISTS idx_progress_status ON index_progress(status);
	`
	_, err := s.db.Exec(schema)
	return err
}