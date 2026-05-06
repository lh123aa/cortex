package storage

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"path/filepath"

	"github.com/lh123aa/cortex/internal/vector"
	"go.uber.org/zap"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var embeddedSchemaSQL string

type SQLiteStorage struct {
	db       *sql.DB
	hnsw     *vector.StorageBridge
	useHNSW  bool
	vecIndex *vector.VectorIndex // 向量索引管理器（用于持久化）
	dbPath   string              // 数据库路径（用于计算向量索引路径）
	logger   *zap.Logger         // 结构化日志（可选，设日志记录，默认回退到 log.Printf）
}

// SetLogger 设置结构化日志记录器
func (s *SQLiteStorage) SetLogger(logger *zap.Logger) {
	s.logger = logger
}

// NewSQLiteStorage 初始化并打开 SQLite 数据库
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)") // 启动外键支持
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// 初始化内置Schema (去除外部文件路径依赖)
	if err := initSchema(db); err != nil {
		return nil, fmt.Errorf("failed to init schema: %w", err)
	}

	s := &SQLiteStorage{
		db:      db,
		useHNSW: false, // 默认关闭，等 BuildHNSWIndex 后开启
		dbPath:  dbPath,
	}

	// 初始化 index_progress 表
	if err := s.InitIndexProgressTable(); err != nil {
		s.logWarn("failed to init index_progress table", zap.Error(err))
	}

	// 数据库迁移：为旧表添加 content_hash 列 + 索引
	if _, err := s.db.Exec(`ALTER TABLE chunks ADD COLUMN content_hash TEXT DEFAULT ''`); err != nil {
		// 列已存在时忽略错误
		s.logWarn("migration: add content_hash column (may already exist)", zap.Error(err))
	}
	if _, err := s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_chunks_hash ON chunks(content_hash)`); err != nil {
		s.logWarn("migration: create idx_chunks_hash index", zap.Error(err))
	}

	return s, nil
}

// BuildHNSWIndex 从数据库加载向量构建 HNSW 索引
// v2.1: 优先从磁盘加载已持久化的索引，避免每次启动重建
func (s *SQLiteStorage) BuildHNSWIndex() error {
	// 先尝试从磁盘加载向量索引
	vecPath := s.getVectorIndexPath()
	idx := vector.NewVectorIndex(vector.DefaultConfig())
	if err := idx.Load(vecPath); err == nil {
		s.logInfo("loaded vector index from disk", zap.String("path", vecPath), zap.Int("count", idx.Count()))
		s.vecIndex = idx
	}

	// 加载或构建 HNSW
	bridge := vector.NewStorageBridge(s.db)
	if err := bridge.LoadFromDB(); err != nil {
		return fmt.Errorf("failed to load vectors from DB: %w", err)
	}

	s.hnsw = bridge
	s.useHNSW = true
	s.logInfo("HNSW index built", zap.Int("vectors", bridge.Count()))

	// 如果从磁盘加载了索引但向量数量不匹配，需要重新构建
	if s.vecIndex != nil && s.vecIndex.Count() != bridge.Count() {
		s.logWarn("vector index count mismatch, will update on next save",
			zap.Int("loaded", s.vecIndex.Count()),
			zap.Int("built", bridge.Count()))
	}

	return nil
}

// logWarn 结构化日志回退：优先使用 zap，否则使用标准 log
func (s *SQLiteStorage) logWarn(msg string, fields ...zap.Field) {
	if s.logger != nil {
		s.logger.Warn(msg, fields...)
	} else {
		log.Printf("Warning: %s %v", msg, fields)
	}
}

// logInfo 结构化日志回退
func (s *SQLiteStorage) logInfo(msg string, fields ...zap.Field) {
	if s.logger != nil {
		s.logger.Info(msg, fields...)
	} else {
		log.Printf("Info: %s %v", msg, fields)
	}
}

// SaveVectorIndex 将向量索引保存到磁盘
func (s *SQLiteStorage) SaveVectorIndex() error {
	if s.vecIndex == nil {
		return nil
	}
	path := s.getVectorIndexPath()
	return s.vecIndex.Save(path)
}

// getVectorIndexPath 获取向量索引文件路径
func (s *SQLiteStorage) getVectorIndexPath() string {
	dir := filepath.Dir(s.dbPath)
	name := filepath.Base(s.dbPath)
	return filepath.Join(dir, name+"_vector_idx.json")
}

// GetVectorIndex 获取向量索引管理器
func (s *SQLiteStorage) GetVectorIndex() *vector.VectorIndex {
	return s.vecIndex
}

// SetVectorIndex 设置向量索引管理器
func (s *SQLiteStorage) SetVectorIndex(idx *vector.VectorIndex) {
	s.vecIndex = idx
}

func initSchema(db *sql.DB) error {
	_, err := db.Exec(embeddedSchemaSQL)
	return err
}

// Close 关闭连接
func (s *SQLiteStorage) Close() error {
	// 关闭前保存向量索引
	if s.vecIndex != nil {
		if err := s.SaveVectorIndex(); err != nil {
			s.logWarn("failed to save vector index", zap.Error(err))
		}
	}
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// DedupChunks 查找并删除内容重复的 chunks（按 content_hash 分组）
func (s *SQLiteStorage) DedupChunks() (removed int, groups int, err error) {
	rows, err := s.db.Query(`
		SELECT content_hash, COUNT(*) as cnt, MIN(rowid) as keep_id
		FROM chunks
		WHERE content_hash != ''
		GROUP BY content_hash
		HAVING cnt > 1
	`)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to query duplicates: %w", err)
	}
	defer rows.Close()

	type dupGroup struct {
		Hash   string
		KeepID int64
	}
	var dups []dupGroup
	for rows.Next() {
		var hash string
		var cnt int
		var keepID int64
		if err := rows.Scan(&hash, &cnt, &keepID); err != nil {
			return 0, 0, fmt.Errorf("failed to scan: %w", err)
		}
		dups = append(dups, dupGroup{Hash: hash, KeepID: keepID})
	}

	for _, d := range dups {
		res, err := s.db.Exec(`DELETE FROM chunks WHERE content_hash = ? AND rowid != ?`, d.Hash, d.KeepID)
		if err != nil {
			return removed, len(dups), fmt.Errorf("failed to delete for hash %s: %w", d.Hash, err)
		}
		n, _ := res.RowsAffected()
		removed += int(n)
	}

	// 使缓存失效
	if removed > 0 {
		s.InvalidateSearchCache()
		_ = s.BuildHNSWIndex()
	}

	return removed, len(dups), nil
}
