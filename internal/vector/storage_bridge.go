package vector

import (
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

// StorageBridge SQLite 和 HNSW 索引的桥接层
type StorageBridge struct {
	db      *sql.DB
	hnsw    *HNSW
	idToIdx map[string]int
	mu      sync.RWMutex
	dim     int
}

// NewStorageBridge 创建新的存储桥接器
func NewStorageBridge(db *sql.DB) *StorageBridge {
	return &StorageBridge{
		db:      db,
		hnsw:    NewHNSW(DefaultConfig()),
		idToIdx: make(map[string]int),
	}
}

// LoadFromDB 从数据库加载所有向量并构建 HNSW 索引
func (s *StorageBridge) LoadFromDB() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(`
		SELECT v.chunk_id, v.embedding, LENGTH(v.embedding) / 4 as dim
		FROM vectors v
	`)
	if err != nil {
		return fmt.Errorf("failed to query vectors: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var chunkID string
		var embedding []byte
		var dim int

		if err := rows.Scan(&chunkID, &embedding, &dim); err != nil {
			return fmt.Errorf("failed to scan vector: %w", err)
		}

		// 解码向量
		vec := make([]float32, dim)
		for i := 0; i < dim; i++ {
			vec[i] = Float32FromBytes(embedding[i*4 : (i+1)*4])
		}

		// 第一次添加时确定维度
		if count == 0 {
			s.dim = dim
		}

		// 添加到 HNSW
		s.hnsw.Add(chunkID, vec)
		s.idToIdx[chunkID] = count
		count++
	}

	return nil
}

// Add 添加向量到索引
func (s *StorageBridge) Add(chunkID string, embedding []float32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.hnsw.Count()
	s.idToIdx[chunkID] = idx
	s.hnsw.Add(chunkID, embedding)
}

// Search 搜索最近的 k 个向量
func (s *StorageBridge) Search(query []float32, k int) ([]string, []float64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if k > s.hnsw.Count() {
		k = s.hnsw.Count()
	}

	return s.hnsw.Search(query, k)
}

// Remove 从索引中移除向量
func (s *StorageBridge) Remove(chunkID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx, exists := s.idToIdx[chunkID]
	if !exists {
		return
	}

	// 将向量置零
	if idx < len(s.hnsw.vectors) {
		zero := make([]float32, s.dim)
		for i := range zero {
			zero[i] = 0
		}
		s.hnsw.vectors[idx] = zero
	}

	delete(s.idToIdx, chunkID)
}

// Count 返回索引中的向量数量
func (s *StorageBridge) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.hnsw.Count()
}

// Float32FromBytes 从字节切片读取 float32 (小端序)
func Float32FromBytes(b []byte) float32 {
	bits := uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
	return float32(bits)
}
