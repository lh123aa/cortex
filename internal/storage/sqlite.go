package storage

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"path/filepath"
	"strings"

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

	// 数据库迁移：检查并添加缺失的列（无警告）
	if !s.hasColumn("chunks", "content_hash") {
		if _, err := s.db.Exec(`ALTER TABLE chunks ADD COLUMN content_hash TEXT DEFAULT ''`); err != nil {
			s.logWarn("migration: failed to add content_hash column", zap.Error(err))
		}
	}
	if !s.hasColumn("chunks", "minhash_sig") {
		if _, err := s.db.Exec(`ALTER TABLE chunks ADD COLUMN minhash_sig BLOB DEFAULT NULL`); err != nil {
			s.logWarn("migration: failed to add minhash_sig column", zap.Error(err))
		}
	}
	if _, err := s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_chunks_hash ON chunks(content_hash)`); err != nil {
		s.logWarn("migration: create idx_chunks_hash index", zap.Error(err))
	}

	// 数据库迁移：users 表加 tier/storage 列
	if !s.hasColumn("users", "tier") {
		s.db.Exec(`ALTER TABLE users ADD COLUMN tier TEXT DEFAULT 'free'`)
		s.db.Exec(`ALTER TABLE users ADD COLUMN storage_used_bytes INTEGER DEFAULT 0`)
		s.db.Exec(`ALTER TABLE users ADD COLUMN storage_limit_bytes INTEGER DEFAULT 1073741824`)
		s.db.Exec(`ALTER TABLE users ADD COLUMN license_key TEXT DEFAULT ''`)
		s.db.Exec(`ALTER TABLE users ADD COLUMN last_login_at TIMESTAMP`)
		s.db.Exec(`ALTER TABLE users ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`)
	}
	// License 表由 schema.sql 自动创建（IF NOT EXISTS）

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

// hasColumn 检查 SQLite 表中是否存在指定列
func (s *SQLiteStorage) hasColumn(table, col string) bool {
	rows, err := s.db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false
		}
		if name == col {
			return true
		}
	}
	return false
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

// DedupByVector 基于向量相似度的去重
// 扫描所有带向量的 chunks，对每个 chunk 在 HNSW 中找最近邻
// 如果余弦相似度 >= threshold，则判定为语义重复并删除
func (s *SQLiteStorage) DedupByVector(threshold float64) (removed int, candidates int, err error) {
	if s.hnsw == nil {
		return 0, 0, fmt.Errorf("HNSW index not available, need to index documents first")
	}

	// 获取所有带向量的 chunk
	rows, err := s.db.Query(`
		SELECT c.id, v.embedding
		FROM chunks c
		JOIN vectors v ON v.chunk_id = c.id
		ORDER BY c.rowid ASC
	`)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to query chunks with vectors: %w", err)
	}
	defer rows.Close()

	type chunkVec struct {
		ID        string
		Embedding []float32
	}

	var allChunks []chunkVec
	for rows.Next() {
		var id string
		var embBytes []byte
		if err := rows.Scan(&id, &embBytes); err != nil {
			return removed, 0, fmt.Errorf("failed to scan: %w", err)
		}
		vec := BytesToFloat32Array(embBytes)
		if len(vec) == 0 {
			continue
		}
		allChunks = append(allChunks, chunkVec{ID: id, Embedding: vec})
	}

	// 对每个 chunk 搜索最近邻，跳过自己
	var deleted int
	var toDelete []string
	for i, c := range allChunks {
		// 搜索 2 个最近邻（第一个是自己，第二个是最近的邻居）
		ids, distances := s.hnsw.Search(c.Embedding, 2)
		if len(ids) < 2 {
			continue
		}

		// distances[0] 是 0（自己），distances[1] 是最近邻
		similarity := 1.0 - distances[1]

		// 如果相似度超过阈值且邻居的 ID 更大（保留更早的 chunk），则删除
		if similarity >= threshold && ids[1] > ids[0] {
			toDelete = append(toDelete, ids[1])
		}

		// 每处理 50 个 chunk，批量删除一次
		if (i+1)%50 == 0 && len(toDelete) > 0 {
			n, err := s.deleteChunksByIDs(toDelete)
			if err != nil {
				return removed, len(allChunks), fmt.Errorf("failed to delete at chunk %d: %w", i, err)
			}
			deleted += n
			toDelete = nil
		}
	}

	// 删除剩余的
	if len(toDelete) > 0 {
		n, err := s.deleteChunksByIDs(toDelete)
		if err != nil {
			return removed, len(allChunks), err
		}
		deleted += n
	}

	if deleted > 0 {
		s.InvalidateSearchCache()
		_ = s.BuildHNSWIndex()
	}

	return deleted, len(allChunks), nil
}

// deleteChunksByIDs 批量删除 chunks
func (s *SQLiteStorage) deleteChunksByIDs(ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	placeholders := strings.Repeat(",?", len(ids)-1)
	q := `DELETE FROM chunks WHERE id IN (?` + placeholders + `)`
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	res, err := s.db.Exec(q, args...)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// DedupByMinHash 基于 MinHash 签名的近似去重
// 扫描所有带 MinHash 签名的 chunks，Jaccard 相似度 >= threshold 则删除
func (s *SQLiteStorage) DedupByMinHash(threshold float64) (removed int, candidates int, err error) {
	rows, err := s.db.Query(`
		SELECT c.id, c.minhash_sig, c.rowid
		FROM chunks c
		WHERE c.minhash_sig IS NOT NULL
		ORDER BY c.rowid ASC
	`)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to query minhash sigs: %w", err)
	}
	defer rows.Close()

	type sigEntry struct {
		ID    string
		Sig   *vector.MinHash
		RowID int64
	}
	var entries []sigEntry
	for rows.Next() {
		var id string
		var data []byte
		var rowID int64
		if err := rows.Scan(&id, &data, &rowID); err != nil {
			return removed, 0, fmt.Errorf("failed to scan: %w", err)
		}
		mh := vector.MinHashFromBytes(data)
		if mh == nil {
			continue
		}
		entries = append(entries, sigEntry{ID: id, Sig: mh, RowID: rowID})
	}

	// 两两比对，发现高相似度的 chunk
	var toDelete []string
	for i := 0; i < len(entries); i++ {
		if entries[i].Sig == nil {
			continue
		}
		for j := i + 1; j < len(entries); j++ {
			if entries[j].Sig == nil {
				continue
			}
			jaccard := entries[i].Sig.Jaccard(entries[j].Sig)
			if jaccard >= threshold {
				// 保留较早的（rowID 小的），删除较晚的
				if entries[i].RowID < entries[j].RowID {
					toDelete = append(toDelete, entries[j].ID)
					entries[j].Sig = nil // 标记已处理
				} else {
					toDelete = append(toDelete, entries[i].ID)
					entries[i].Sig = nil // 标记已处理
					break
				}
			}
		}
	}

	if len(toDelete) > 0 {
		n, err := s.deleteChunksByIDs(toDelete)
		if err != nil {
			return 0, len(entries), fmt.Errorf("failed to delete: %w", err)
		}
		removed = n
		s.InvalidateSearchCache()
	}

	return removed, len(entries), nil
}
