package storage

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"path/filepath"

	"github.com/lh123aa/cortex/internal/vector"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var embeddedSchemaSQL string

type SQLiteStorage struct {
	db       *sql.DB
	hnsw     *vector.StorageBridge
	useHNSW  bool
	vecIndex *vector.VectorIndex // 向量索引管理器（用于持久化）
	dbPath   string              // 数据库路径（用于计算向量索引路径）
}

// NewSQLiteStorage 初始化并打开 SQLite 数据库
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_fk=1&cache=shared") // 启动外键支持
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
		log.Printf("Warning: failed to init index_progress table: %v", err)
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
		log.Printf("Loaded vector index from %s with %d vectors", vecPath, idx.Count())
		s.vecIndex = idx
	}

	// 加载或构建 HNSW
	bridge := vector.NewStorageBridge(s.db)
	if err := bridge.LoadFromDB(); err != nil {
		return fmt.Errorf("failed to load vectors from DB: %w", err)
	}

	s.hnsw = bridge
	s.useHNSW = true
	log.Printf("HNSW index built with %d vectors", bridge.Count())

	// 如果从磁盘加载了索引但向量数量不匹配，需要重新构建
	if s.vecIndex != nil && s.vecIndex.Count() != bridge.Count() {
		log.Printf("Vector index count mismatch (%d vs %d), will update on next save", s.vecIndex.Count(), bridge.Count())
	}

	return nil
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
			log.Printf("Warning: failed to save vector index: %v", err)
		}
	}
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
