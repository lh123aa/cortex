package storage

import (
	"database/sql"
	"fmt"
	_ "embed"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var embeddedSchemaSQL string

type SQLiteStorage struct {
	db *sql.DB
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

	return &SQLiteStorage{db: db}, nil
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

