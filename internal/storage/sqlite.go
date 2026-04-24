package storage

import (
	"database/sql"
	"fmt"
	_ "embed"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"github.com/lh123aa/cortex/internal/vector"
)

//go:embed schema.sql
var embeddedSchemaSQL string

type SQLiteStorage struct {
	db      *sql.DB
	hnsw    *vector.StorageBridge
	useHNSW bool
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
	}

	// 初始化 index_progress 表
	if err := s.InitIndexProgressTable(); err != nil {
		log.Printf("Warning: failed to init index_progress table: %v", err)
	}

	return s, nil
}

// BuildHNSWIndex 从数据库加载向量构建 HNSW 索引
func (s *SQLiteStorage) BuildHNSWIndex() error {
	bridge := vector.NewStorageBridge(s.db)
	if err := bridge.LoadFromDB(); err != nil {
		return fmt.Errorf("failed to load vectors from DB: %w", err)
	}

	s.hnsw = bridge
	s.useHNSW = true
	log.Printf("HNSW index built with %d vectors", bridge.Count())
	return nil
}

func initSchema(db *sql.DB) error {
	_, err := db.Exec(embeddedSchemaSQL)
	return err
}

// Close 关闭连接
func (s *SQLiteStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

