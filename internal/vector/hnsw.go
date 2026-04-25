package vector

import (
	"container/heap"
	"math"
	"math/rand"
	"sort"
	"sync"
)

// HNSW 配置参数
type Config struct {
	MaxLayers    int     // 最大层数，默认为 16
	EfConstruction int   // 构建时的动态列表大小，默认为 200
	M            int     // 底层连接数，默认为 32
	ML           float64 // 层间因子，默认为 1/log(32) ≈ 0.216
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		MaxLayers:     16,
		EfConstruction: 200,
		M:             32,
		ML:           1 / math.Log(32),
	}
}

// HNSW 索引结构
type HNSW struct {
	cfg        *Config
	vectors    [][]float32   // 存储所有向量
	ids        []string      // 向量对应的 ID
	neighbors  [][][]int     // 邻居节点: neighbors[layer][node_id] -> []neighbors
	enterPoint int           // 入口点
	mu         sync.RWMutex  // 并发控制
	dim        int           // 向量维度
	count      int           // 向量数量
	maxLevel   int           // 当前最大层
}

// NewHNSW 创建新的 HNSW 索引
func NewHNSW(cfg *Config) *HNSW {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &HNSW{
		cfg:       cfg,
		vectors:   make([][]float32, 0),
		ids:       make([]string, 0),
		neighbors: make([][][]int, cfg.MaxLayers),
	}
}

// Dimension 返回向量维度
func (h *HNSW) Dimension() int {
	return h.dim
}

// Count 返回索引中的向量数量
func (h *HNSW) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.count
}

// randomLevel 生成随机层数，遵循指数分布
// 使用标准 HNSW 算法: level 0 的概率为 1/e^ml
func (h *HNSW) randomLevel() int {
	lvl := 0
	for lvl < h.cfg.MaxLayers-1 {
		if rand.Float64() > math.Exp(-h.cfg.ML) {
			break
		}
		lvl++
	}
	return lvl
}

// normalize 向量归一化
func normalize(v []float32) []float32 {
	var norm float64
	for _, val := range v {
		norm += float64(val * val)
	}
	if norm == 0 {
		return v
	}
	norm = math.Sqrt(norm)
	result := make([]float32, len(v))
	for i, val := range v {
		result[i] = float32(float64(val) / norm)
	}
	return result
}

// cosineDistance 计算余弦距离 (1 - cosine_similarity)
func cosineDistance(a, b []float32) float64 {
	var dotProd, normA, normB float64
	for i := range a {
		dotProd += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 1.0
	}
	sim := dotProd / (math.Sqrt(normA) * math.Sqrt(normB))
	return 1.0 - sim
}

// l2Distance 计算 L2 距离
func l2Distance(a, b []float32) float64 {
	var sum float64
	for i := range a {
		diff := float64(a[i] - b[i])
		sum += diff * diff
	}
	return math.Sqrt(sum)
}

// isZeroVector 检查向量是否为零向量（已删除标记）
func isZeroVector(v []float32) bool {
	for _, val := range v {
		if val != 0 {
			return false
		}
	}
	return true
}

// Add 向索引添加向量
func (h *HNSW) Add(id string, vector []float32) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 第一次添加时确定维度
	if h.count == 0 {
		h.dim = len(vector)
	}

	// 归一化向量
	vector = normalize(vector)

	// 分配节点ID
	nodeID := h.count
	h.vectors = append(h.vectors, vector)
	h.ids = append(h.ids, id)
	h.count++

	// 如果是第一个节点，初始化第0层邻居（自连接）
	if nodeID == 0 {
		h.enterPoint = 0
		h.maxLevel = h.randomLevel()
		// 初始化邻居结构
		for i := 0; i <= h.maxLevel; i++ {
			h.neighbors = append(h.neighbors, nil)
			h.neighbors[i] = make([][]int, 0)
		}
		// 在第0层添加自连接（node 0 连接到自己）
		h.neighbors[0] = append(h.neighbors[0], []int{0})
		return
	}

	// 生成新节点的层数
	newLevel := h.randomLevel()
	if newLevel > h.maxLevel {
		newLevel = h.maxLevel + 1
		if newLevel >= h.cfg.MaxLayers {
			newLevel = h.cfg.MaxLayers - 1
		}
	}

	// 从顶层到底层逐层插入
	for level := h.maxLevel; level >= 0; level-- {
		h.insertAtLevel(nodeID, vector, level)
	}
}

// insertAtLevel 在指定层插入节点
func (h *HNSW) insertAtLevel(nodeID int, vector []float32, level int) {
	// 确保邻居切片已初始化
	for len(h.neighbors) <= level {
		h.neighbors = append(h.neighbors, nil)
	}
	if h.neighbors[level] == nil {
		h.neighbors[level] = make([][]int, 0)
	}

	// 扩展邻居数组
	for len(h.neighbors[level]) <= nodeID {
		h.neighbors[level] = append(h.neighbors[level], nil)
	}

	// 获取该层的入口点
	ep := h.enterPoint
	if level > h.maxLevel {
		return
	}

	// 搜索最近邻
	visited := make(map[int]bool)
	ef := h.cfg.EfConstruction
	candidates := &priorityQueue{}

	// 初始化：从入口点开始
	dist := cosineDistance(vector, h.vectors[ep])
	candidates.push(ep, dist)
	visited[ep] = true

	for candidates.Len() > 0 {
		// 获取当前最近但最远的候选
		current, d := candidates.pop()

		// 获取与当前候选的距离 (声明但未使用，实际使用 candidates 中的 d)
		_ = cosineDistance(vector, h.vectors[current])

		// 检查是否需要终止
		if d > candidates.getWorst() && candidates.Len() >= ef {
			break
		}

		// 遍历邻居
		for _, neighbor := range h.neighbors[level][current] {
			if neighbor >= len(h.vectors) || neighbor >= len(h.neighbors[level]) {
				continue
			}
			if visited[neighbor] {
				continue
			}
			visited[neighbor] = true

			ndist := cosineDistance(vector, h.vectors[neighbor])
			if ndist < d || candidates.Len() < ef {
				candidates.push(neighbor, ndist)
			}
		}
	}

	// 选择连接
	var connections []int
	for candidates.Len() > 0 && len(connections) < h.cfg.M {
		id, _ := candidates.pop()
		connections = append(connections, id)
	}

	// Fallback: 如果没有找到任何连接，至少连接到入口点
	// 这是HNSW正确工作的关键
	if len(connections) == 0 && ep != nodeID {
		connections = append(connections, ep)
	}

	// 添加双向连接
	for _, neighbor := range connections {
		// 添加 neighbor -> nodeID 的连接
		h.neighbors[level][neighbor] = append(h.neighbors[level][neighbor], nodeID)
		// 添加 nodeID -> neighbor 的连接
		h.neighbors[level][nodeID] = append(h.neighbors[level][nodeID], neighbor)
	}

	// 更新入口点为最近邻
	if len(connections) > 0 {
		h.enterPoint = connections[0]
	}
}

// Search 搜索最近的 k 个向量
func (h *HNSW) Search(query []float32, k int) ([]string, []float64) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.count == 0 {
		return nil, nil
	}

	// 归一化查询向量
	query = normalize(query)

	// 从顶层开始，逐层下降到第 0 层
	ep := h.enterPoint
	for level := h.maxLevel; level > 0; level-- {
		layerResults := h.searchLayer(query, ep, 1, level)
		if len(layerResults) > 0 && layerResults[0].id < len(h.vectors) {
			ep = layerResults[0].id
		} else {
			// 如果该层搜索失败，保持当前 ep
		}
	}

	// 底层精细搜索
	results := h.searchLayer(query, ep, h.cfg.EfConstruction, 0)

	// 按距离排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].dist < results[j].dist
	})

	// 取前 k 个
	if len(results) > k {
		results = results[:k]
	}

	// 过滤已删除的向量
	var validIDs []string
	var validDistances []float64
	for i, r := range results {
		if r.id < len(h.vectors) && !isZeroVector(h.vectors[r.id]) {
			validIDs = append(validIDs, h.ids[r.id])
			validDistances = append(validDistances, results[i].dist)
		}
	}

	// 取前 k 个
	if len(validIDs) > k {
		validIDs = validIDs[:k]
		validDistances = validDistances[:k]
	}

	return validIDs, validDistances
}

// searchResult 搜索结果
type searchResult struct {
	id  int
	dist float64
}

// priorityQueue 最小堆优先队列
type priorityQueue struct {
	items []searchResult
}

func (pq *priorityQueue) Len() int {
	return len(pq.items)
}

func (pq *priorityQueue) Less(i, j int) bool {
	return pq.items[i].dist < pq.items[j].dist
}

func (pq *priorityQueue) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
}

func (pq *priorityQueue) Push(x any) {
	pq.items = append(pq.items, x.(searchResult))
}

func (pq *priorityQueue) Pop() any {
	// heap.Pop 已经做了 swap 和 sift down
	// 这里只需要返回 items[n]（原来的堆顶）
	n := len(pq.items) - 1
	item := pq.items[n]
	pq.items = pq.items[:n]
	return item
}

func (pq *priorityQueue) push(id int, dist float64) {
	heap.Push(pq, searchResult{id: id, dist: dist})
}

func (pq *priorityQueue) pop() (int, float64) {
	if len(pq.items) == 0 {
		return 0, 0
	}
	item := heap.Pop(pq).(searchResult)
	return item.id, item.dist
}

func (pq *priorityQueue) getWorst() float64 {
	if len(pq.items) == 0 {
		return math.MaxFloat64
	}
	return pq.items[len(pq.items)-1].dist
}

// searchLayer 在指定层搜索最近邻
func (h *HNSW) searchLayer(query []float32, entryPoint int, ef int, level int) []searchResult {
	// 空索引检查
	if len(h.vectors) == 0 || entryPoint >= len(h.vectors) {
		return []searchResult{}
	}

	// 如果入口点已被删除，使用下一个有效节点
	if isZeroVector(h.vectors[entryPoint]) {
		for i := 0; i < len(h.vectors); i++ {
			if !isZeroVector(h.vectors[i]) {
				entryPoint = i
				break
			}
		}
		// 如果所有节点都被删除，返回空
		if isZeroVector(h.vectors[entryPoint]) {
			return []searchResult{}
		}
	}

	visited := make(map[int]bool)
	candidates := &priorityQueue{}
	result := &priorityQueue{}

	candidates.push(entryPoint, cosineDistance(query, h.vectors[entryPoint]))
	visited[entryPoint] = true

	for candidates.Len() > 0 {
		current, currentDist := candidates.pop()

		if result.Len() >= ef && currentDist > result.getWorst() {
			break
		}

		result.push(current, currentDist)

		// 遍历邻居
		if level >= len(h.neighbors) || current >= len(h.neighbors[level]) {
			continue
		}
		for _, neighbor := range h.neighbors[level][current] {
			if neighbor >= len(h.vectors) || visited[neighbor] {
				continue
			}
			// 跳过已删除的向量
			if isZeroVector(h.vectors[neighbor]) {
				continue
			}
			visited[neighbor] = true

			dist := cosineDistance(query, h.vectors[neighbor])

			if result.Len() < ef || dist < result.getWorst() {
				candidates.push(neighbor, dist)
			}
		}
	}

	return result.items
}

// Remove 从索引中删除向量（通过标记）
func (h *HNSW) Remove(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 找到向量索引
	idx := -1
	for i, v := range h.ids {
		if v == id {
			idx = i
			break
		}
	}

	if idx == -1 {
		return
	}

	// 将向量置零（标记为已删除）
	h.vectors[idx] = make([]float32, h.dim)
}

// Clear 清空索引
func (h *HNSW) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.vectors = make([][]float32, 0)
	h.ids = make([]string, 0)
	h.neighbors = make([][][]int, h.cfg.MaxLayers)
	h.count = 0
	h.enterPoint = 0
	h.maxLevel = 0
}
