package models

import "time"

// IndexProgress 索引进度追踪（用于断点恢复）
type IndexProgress struct {
	ID            int       `json:"id"`
	RootPath      string    `json:"root_path"`
	LastFilePath  string    `json:"last_file_path"`  // 最后处理的文件路径
	LastFileIndex int       `json:"last_file_index"` // 已处理文件索引
	TotalFiles    int       `json:"total_files"`     // 总文件数
	IndexedFiles  int       `json:"indexed_files"`   // 已处理文件数
	IndexedChunks int       `json:"indexed_chunks"`  // 已处理 chunks 数
	FailedFiles   int       `json:"failed_files"`    // 失败文件数
	Status        string    `json:"status"`          // running/completed/failed
	StartedAt     time.Time `json:"started_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	CompletedAt   time.Time `json:"completed_at,omitempty"`
	ErrorMessage  string    `json:"error_message,omitempty"`
}

// IndexCheckpoint 索引检查点（用于增量索引）
type IndexCheckpoint struct {
	FilePath    string    `json:"file_path"`
	FileIndex   int       `json:"file_index"`
	IsDirectory bool      `json:"is_directory"`
	ProcessedAt time.Time `json:"processed_at"`
}
