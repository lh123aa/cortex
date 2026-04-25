package vector

import (
	"encoding/binary"
	"fmt"
	"math"
	"sync"
)

// PQConfig Product Quantization 配置
type PQConfig struct {
	Dimension     int // 原始向量维度 (如 768)
	CompressedDim int // 压缩后维度 (如 64)
	CodebookSize  int // 码本大小 (通常 256)
	Iterations    int // k-means 迭代次数
}

// DefaultPQConfig 默认 PQ 配置
func DefaultPQConfig() *PQConfig {
	return &PQConfig{
		Dimension:     768,
		CompressedDim: 64,
		CodebookSize:  256,
		Iterations:    20,
	}
}

// PQ Product Quantization 向量压缩
// 将高维向量压缩为低维向量码，大幅减少内存占用
type PQ struct {
	cfg        *PQConfig
	codebook   []float32 // 码本: codebookSize * compressedDim 个 float32
	subVectors int       // 子向量数量
	mu         sync.RWMutex
}

// NewPQ 创建新的 PQ 量化器
func NewPQ(cfg *PQConfig) *PQ {
	return &PQ{
		cfg:        cfg,
		codebook:   make([]float32, cfg.CodebookSize*cfg.CompressedDim),
		subVectors: cfg.Dimension / cfg.CompressedDim,
	}
}

// Train 从样本向量训练码本
// sampleVectors: 用于训练码本的样本向量（建议 > 10 * codebookSize）
func (pq *PQ) Train(sampleVectors [][]float32) error {
	if len(sampleVectors) == 0 {
		return fmt.Errorf("no sample vectors provided")
	}

	dim := len(sampleVectors[0])
	if dim != pq.cfg.Dimension {
		return fmt.Errorf("vector dimension mismatch: expected %d, got %d", pq.cfg.Dimension, dim)
	}

	if len(sampleVectors) < pq.cfg.CodebookSize {
		return fmt.Errorf("not enough samples: need at least %d, got %d", pq.cfg.CodebookSize, len(sampleVectors))
	}

	pq.subVectors = dim / pq.cfg.CompressedDim
	if pq.subVectors == 0 {
		pq.subVectors = 1
		pq.cfg.CompressedDim = dim
	}

	subDim := dim / pq.subVectors

	// 对每个子空间进行 k-means 聚类
	for s := 0; s < pq.subVectors; s++ {
		// 提取子向量
		subVectors := make([][]float32, len(sampleVectors))
		for i, vec := range sampleVectors {
			subVectors[i] = make([]float32, subDim)
			for j := 0; j < subDim; j++ {
				subVectors[i][j] = vec[s*subDim+j]
			}
		}

		// k-means 聚类
		centroids := pq.kmeans(subVectors, pq.cfg.CodebookSize, pq.cfg.Iterations)

		// 将质心复制到码本
		offset := s * pq.cfg.CodebookSize * subDim
		for c := 0; c < pq.cfg.CodebookSize; c++ {
			for j := 0; j < subDim; j++ {
				pq.codebook[offset+c*subDim+j] = centroids[c][j]
			}
		}
	}

	return nil
}

// kmeans k-means 聚类实现
func (pq *PQ) kmeans(vectors [][]float32, k, iterations int) (centroids [][]float32) {
	// 初始化质心（使用 k-means++）
	centroids = pq.initCentroids(vectors, k)

	subDim := len(vectors[0])

	for iter := 0; iter < iterations; iter++ {
		// 分配样本到最近的质心
		clusters := make([][][]float32, k)
		for _, vec := range vectors {
			closest := 0
			minDist := sqeuclidean(vec, centroids[0])
			for c := 1; c < k; c++ {
				dist := sqeuclidean(vec, centroids[c])
				if dist < minDist {
					minDist = dist
					closest = c
				}
			}
			clusters[closest] = append(clusters[closest], vec)
		}

		// 更新质心
		for c := 0; c < k; c++ {
			if len(clusters[c]) == 0 {
				continue
			}
			newCentroid := make([]float32, subDim)
			for _, vec := range clusters[c] {
				for j := 0; j < subDim; j++ {
					newCentroid[j] += vec[j]
				}
			}
			for j := 0; j < subDim; j++ {
				newCentroid[j] /= float32(len(clusters[c]))
			}
			centroids[c] = newCentroid
		}
	}

	return centroids
}

// initCentroids k-means++ 初始化
func (pq *PQ) initCentroids(vectors [][]float32, k int) (centroids [][]float32) {
	// 随机选择第一个质心
	centroids = append(centroids, vectors[0])

	for i := 1; i < k; i++ {
		// 计算每个点到最近质心的距离
		maxDist := 0.0
		var farthest []float32
		for _, vec := range vectors {
			minDist := sqeuclidean(vec, centroids[0])
			for c := 1; c < len(centroids); c++ {
				dist := sqeuclidean(vec, centroids[c])
				if dist < minDist {
					minDist = dist
				}
			}
			if minDist > maxDist {
				maxDist = minDist
				farthest = vec
			}
		}
		centroids = append(centroids, farthest)
	}

	return centroids
}

// Compress 将向量压缩为字节码
func (pq *PQ) Compress(vector []float32) []byte {
	if len(vector) != pq.cfg.Dimension {
		return nil
	}

	codes := make([]byte, pq.subVectors)
	subDim := pq.cfg.Dimension / pq.subVectors

	for s := 0; s < pq.subVectors; s++ {
		// 提取子向量
		subVec := make([]float32, subDim)
		for j := 0; j < subDim; j++ {
			subVec[j] = vector[s*subDim+j]
		}

		// 找到最近的码字
		bestCode := 0
		minDist := sqeuclidean(subVec, pq.codebook[s*pq.cfg.CodebookSize*subDim:])
		for c := 1; c < pq.cfg.CodebookSize; c++ {
			offset := s*pq.cfg.CodebookSize*subDim + c*subDim
			dist := sqeuclidean(subVec, pq.codebook[offset:offset+subDim])
			if dist < minDist {
				minDist = dist
				bestCode = c
			}
		}
		codes[s] = byte(bestCode)
	}

	return codes
}

// Decompress 将字节码解压为近似原始向量
func (pq *PQ) Decompress(codes []byte) []float32 {
	if len(codes) != pq.subVectors {
		return nil
	}

	vector := make([]float32, pq.cfg.Dimension)
	subDim := pq.cfg.Dimension / pq.subVectors

	for s := 0; s < pq.subVectors; s++ {
		code := int(codes[s])
		offset := s*pq.cfg.CodebookSize*subDim + code*subDim
		for j := 0; j < subDim; j++ {
			vector[s*subDim+j] = pq.codebook[offset+j]
		}
	}

	return vector
}

// DecodeFromBytes 从二进制数据解码向量
func (pq *PQ) DecodeFromBytes(data []byte) ([]float32, error) {
	if len(data) != pq.subVectors {
		return nil, fmt.Errorf("invalid compressed data length: expected %d, got %d", pq.subVectors, len(data))
	}

	return pq.Decompress(data), nil
}

// EncodeToBytes 将向量编码为二进制
func (pq *PQ) EncodeToBytes(vector []float32) ([]byte, error) {
	codes := pq.Compress(vector)
	if codes == nil {
		return nil, fmt.Errorf("compression failed")
	}
	return codes, nil
}

// CodeSize 返回压缩后的字节数
func (pq *PQ) CodeSize() int {
	return pq.subVectors // 1 byte per sub-vector
}

// CompressionRatio 返回压缩率
func (pq *PQ) CompressionRatio() float64 {
	originalSize := pq.cfg.Dimension * 4 // 4 bytes per float32
	compressedSize := pq.subVectors * 1  // 1 byte per sub-vector
	return float64(originalSize) / float64(compressedSize)
}

// GetConfig 返回 PQ 配置
func (pq *PQ) GetConfig() *PQConfig {
	return pq.cfg
}

// SerializeCodebook 将码本序列化为字节
func (pq *PQ) SerializeCodebook() []byte {
	data := make([]float32, len(pq.codebook))
	copy(data, pq.codebook)

	bytes := make([]byte, len(data)*4)
	for i, v := range data {
		binary.LittleEndian.PutUint32(bytes[i*4:], math.Float32bits(v))
	}
	return bytes
}

// LoadCodebook 从字节加载码本
func (pq *PQ) LoadCodebook(data []byte) error {
	if len(data)%4 != 0 {
		return fmt.Errorf("invalid codebook data length")
	}

	expectedSize := pq.cfg.CodebookSize * pq.cfg.CompressedDim
	if len(data)/4 != expectedSize {
		return fmt.Errorf("codebook size mismatch: expected %d, got %d", expectedSize, len(data)/4)
	}

	pq.codebook = make([]float32, expectedSize)
	for i := 0; i < expectedSize; i++ {
		pq.codebook[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[i*4:]))
	}

	return nil
}

// sqeuclidean 计算平方欧氏距离
func sqeuclidean(a, b []float32) float64 {
	var sum float64
	for i := 0; i < len(a) && i < len(b); i++ {
		diff := float64(a[i] - b[i])
		sum += diff * diff
	}
	return sum
}

// PQIndex 带 PQ 压缩的向量索引
type PQIndex struct {
	pq       *PQ
	chunkIDs []string
	codes    [][]byte // 压缩后的向量码
	idToIdx  map[string]int
	mu       sync.RWMutex
}

// NewPQIndex 创建带 PQ 压缩的向量索引
func NewPQIndex(cfg *PQConfig) *PQIndex {
	return &PQIndex{
		pq:       NewPQ(cfg),
		chunkIDs: make([]string, 0),
		codes:    make([][]byte, 0),
		idToIdx:  make(map[string]int),
	}
}

// TrainIndex 从样本向量训练索引
func (pqi *PQIndex) TrainIndex(vectors map[string][]float32) error {
	// 收集所有向量用于训练
	samples := make([][]float32, 0, len(vectors))
	for _, vec := range vectors {
		samples = append(samples, vec)
	}

	// 训练 PQ 码本
	if err := pqi.pq.Train(samples); err != nil {
		return err
	}

	// 压缩所有向量并建立索引
	for id, vec := range vectors {
		code := pqi.pq.Compress(vec)
		if code != nil {
			pqi.chunkIDs = append(pqi.chunkIDs, id)
			pqi.codes = append(pqi.codes, code)
			pqi.idToIdx[id] = len(pqi.chunkIDs) - 1
		}
	}

	return nil
}

// Search 搜索最近的 k 个向量（使用压缩向量）
func (pqi *PQIndex) Search(query []float32, k int) ([]string, []float64) {
	pqi.mu.RLock()
	defer pqi.mu.RUnlock()

	if k > len(pqi.chunkIDs) {
		k = len(pqi.chunkIDs)
	}

	if k == 0 {
		return nil, nil
	}

	// 压缩查询向量
	queryCode := pqi.pq.Compress(query)
	if queryCode == nil {
		return nil, nil
	}

	// 计算距离
	type result struct {
		id         string
		approxDist float64 // 近似距离
	}
	results := make([]result, 0, len(pqi.chunkIDs))

	for i, code := range pqi.codes {
		// 计算压缩空间中的距离
		dist := pqi.approxDistance(queryCode, code)
		results = append(results, result{pqi.chunkIDs[i], dist})
	}

	// 选择 top-k
	topK := results[:k]
	if len(results) > k {
		// 简单排序
		for i := 0; i < len(results)-1; i++ {
			for j := i + 1; j < len(results); j++ {
				if results[j].approxDist < results[i].approxDist {
					results[i], results[j] = results[j], results[i]
				}
			}
		}
		topK = results[:k]
	}

	// 返回结果
	ids := make([]string, k)
	distances := make([]float64, k)
	for i, r := range topK {
		ids[i] = r.id
		distances[i] = r.approxDist
	}

	return ids, distances
}

// approxDistance 计算两个压缩码的近似距离
func (pqi *PQIndex) approxDistance(a, b []byte) float64 {
	if len(a) != len(b) {
		return float64(pqi.pq.cfg.Dimension) // 最大距离
	}

	// 在压缩空间中计算汉明距离作为近似
	hamming := 0
	for i := 0; i < len(a); i++ {
		xor := int(a[i]) ^ int(b[i])
		hamming += _popcnt(uint(xor))
	}

	// 转换为一个大概的欧氏距离估计
	// 这是近似值，实际距离需要解压才能准确计算
	avgSubDim := pqi.pq.cfg.Dimension / pqi.pq.subVectors
	estimatedDist := float64(hamming) * float64(avgSubDim) * 0.1 // 经验系数
	return estimatedDist
}

// _popcnt 计算 popcount
func _popcnt(x uint) int {
	count := 0
	for x != 0 {
		count++
		x &= x - 1
	}
	return count
}

// Add 添加向量到索引
func (pqi *PQIndex) Add(chunkID string, embedding []float32) {
	pqi.mu.Lock()
	defer pqi.mu.Unlock()

	code := pqi.pq.Compress(embedding)
	if code == nil {
		return
	}

	// 检查是否已存在
	if idx, exists := pqi.idToIdx[chunkID]; exists {
		pqi.codes[idx] = code
		return
	}

	pqi.chunkIDs = append(pqi.chunkIDs, chunkID)
	pqi.codes = append(pqi.codes, code)
	pqi.idToIdx[chunkID] = len(pqi.chunkIDs) - 1
}

// Remove 从索引中移除向量
func (pqi *PQIndex) Remove(chunkID string) {
	pqi.mu.Lock()
	defer pqi.mu.Unlock()

	idx, exists := pqi.idToIdx[chunkID]
	if !exists {
		return
	}

	// 标记为已删除
	pqi.codes[idx] = nil
	delete(pqi.idToIdx, chunkID)
}

// Count 返回索引中的向量数量
func (pqi *PQIndex) Count() int {
	pqi.mu.RLock()
	defer pqi.mu.RUnlock()
	return len(pqi.chunkIDs)
}

// MemorySize 返回索引占用的内存（字节）
func (pqi *PQIndex) MemorySize() int {
	pqi.mu.RLock()
	defer pqi.mu.RUnlock()

	codeSize := 0
	for _, code := range pqi.codes {
		if code != nil {
			codeSize += len(code)
		}
	}
	return codeSize
}
