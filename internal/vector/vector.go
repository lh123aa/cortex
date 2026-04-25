package vector

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// VectorIndex 向量索引管理器，支持 HNSW 和持久化
type VectorIndex struct {
	hnsw      *HNSW
	chunkIDs  []string // chunk_id -> index 的映射
	idToIdx   map[string]int
	mu        sync.RWMutex
	indexPath string
}

// NewVectorIndex 创建新的向量索引管理器
func NewVectorIndex(cfg *Config) *VectorIndex {
	return &VectorIndex{
		hnsw:     NewHNSW(cfg),
		chunkIDs: make([]string, 0),
		idToIdx:  make(map[string]int),
	}
}

// NewVectorIndexFromStorage 从存储加载向量构建索引
func NewVectorIndexFromStorage(vectors map[string][]float32) *VectorIndex {
	cfg := DefaultConfig()
	idx := &VectorIndex{
		hnsw:     NewHNSW(cfg),
		chunkIDs: make([]string, 0, len(vectors)),
		idToIdx:  make(map[string]int, len(vectors)),
	}

	i := 0
	for id, vec := range vectors {
		idx.chunkIDs = append(idx.chunkIDs, id)
		idx.idToIdx[id] = i
		idx.hnsw.Add(id, vec)
		i++
	}

	return idx
}

// Add 添加向量到索引
func (v *VectorIndex) Add(chunkID string, embedding []float32) {
	v.mu.Lock()
	defer v.mu.Unlock()

	// 检查是否已存在
	if _, exists := v.idToIdx[chunkID]; exists {
		// 更新向量
		idx := v.idToIdx[chunkID]
		v.hnsw.vectors[idx] = embedding
		return
	}

	// 添加新向量
	idx := len(v.chunkIDs)
	v.chunkIDs = append(v.chunkIDs, chunkID)
	v.idToIdx[chunkID] = idx
	v.hnsw.Add(chunkID, embedding)
}

// Search 搜索最近的 k 个向量
func (v *VectorIndex) Search(query []float32, k int) ([]string, []float64) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if k > v.hnsw.Count() {
		k = v.hnsw.Count()
	}

	return v.hnsw.Search(query, k)
}

// Remove 从索引中移除向量
func (v *VectorIndex) Remove(chunkID string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	idx, exists := v.idToIdx[chunkID]
	if !exists {
		return
	}

	// 标记向量为已删除
	v.hnsw.vectors[idx] = make([]float32, v.hnsw.Dimension())

	// 从映射中移除
	delete(v.idToIdx, chunkID)
}

// Count 返回索引中的向量数量
func (v *VectorIndex) Count() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.hnsw.Count()
}

// Clear 清空索引
func (v *VectorIndex) Clear() {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.hnsw.Clear()
	v.chunkIDs = make([]string, 0)
	v.idToIdx = make(map[string]int)
}

// Save 将索引保存到磁盘
func (v *VectorIndex) Save(path string) error {
	v.mu.RLock()
	defer v.mu.RUnlock()

	data := struct {
		ChunkIDs []string    `json:"chunk_ids"`
		Vectors  [][]float32 `json:"vectors"`
	}{
		ChunkIDs: v.chunkIDs,
		Vectors:  v.hnsw.vectors,
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// Load 从磁盘加载索引
func (v *VectorIndex) Load(path string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var data struct {
		ChunkIDs []string    `json:"chunk_ids"`
		Vectors  [][]float32 `json:"vectors"`
	}

	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&data); err != nil {
		return err
	}

	v.chunkIDs = data.ChunkIDs
	v.idToIdx = make(map[string]int, len(data.ChunkIDs))
	for i, id := range data.ChunkIDs {
		v.idToIdx[id] = i
	}

	// 重建 HNSW
	v.hnsw = NewHNSW(DefaultConfig())
	for i, vec := range data.Vectors {
		v.hnsw.Add(data.ChunkIDs[i], vec)
	}

	return nil
}

// GetIndexPath 获取索引文件路径
func (v *VectorIndex) GetIndexPath(dbPath string) string {
	dir := filepath.Dir(dbPath)
	name := filepath.Base(dbPath)
	return filepath.Join(dir, name+"_vector_idx.json")
}
