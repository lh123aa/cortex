package vector

import (
	"encoding/binary"
	"hash/fnv"
	"math"
	"sort"
)

// MinHash 实现用于检测近似重复内容
// 使用 k 个最小哈希值（bottom-k sketch）作为文档签名
const (
	MinHashSigSize = 128 // 签名大小：128 个 uint64
	ShingleSize    = 4   // 字符 4-gram
)

// MinHash 结构
type MinHash struct {
	signature []uint64 // k 个最小哈希值
}

// NewMinHash 从文本创建 MinHash 签名
func NewMinHash(text string) *MinHash {
	return &MinHash{
		signature: computeSignature(text),
	}
}

// Signature 返回签名数据（用于存储）
func (m *MinHash) Signature() []uint64 {
	return m.signature
}

// Bytes 将签名序列化为字节（用于数据库存储）
func (m *MinHash) Bytes() []byte {
	buf := make([]byte, MinHashSigSize*8)
	for i, v := range m.signature {
		binary.LittleEndian.PutUint64(buf[i*8:], v)
	}
	return buf
}

// MinHashFromBytes 从字节反序列化签名
func MinHashFromBytes(data []byte) *MinHash {
	if len(data) < MinHashSigSize*8 {
		return nil
	}
	sig := make([]uint64, MinHashSigSize)
	for i := range sig {
		sig[i] = binary.LittleEndian.Uint64(data[i*8:])
	}
	return &MinHash{signature: sig}
}

// Jaccard 计算两个 MinHash 签名的 Jaccard 相似度 (0.0 ~ 1.0)
func (m *MinHash) Jaccard(other *MinHash) float64 {
	if other == nil || len(m.signature) != len(other.signature) {
		return 0
	}
	intersection := 0
	for i := range m.signature {
		if m.signature[i] == other.signature[i] {
			intersection++
		}
	}
	return float64(intersection) / float64(len(m.signature))
}

// computeSignature 计算文本的 MinHash 签名
func computeSignature(text string) []uint64 {
	if len(text) < ShingleSize {
		// 短文本，直接哈希
		sig := make([]uint64, MinHashSigSize)
		h := fnv.New64a()
		h.Write([]byte(text))
		val := h.Sum64()
		for i := range sig {
			sig[i] = val ^ uint64(i)*0x9e3779b97f4a7c15 // 黄金分割偏移
		}
		return sig
	}

	// 提取所有 shingle
	shingles := make([]uint64, 0, len(text)-ShingleSize+1)
	for i := 0; i <= len(text)-ShingleSize; i++ {
		h := fnv.New64a()
		h.Write([]byte(text[i : i+ShingleSize]))
		shingles = append(shingles, h.Sum64())
	}

	if len(shingles) == 0 {
		sig := make([]uint64, MinHashSigSize)
		return sig
	}

	// bottom-k 策略：取 k 个最小哈希值
	sort.Slice(shingles, func(i, j int) bool { return shingles[i] < shingles[j] })

	sig := make([]uint64, MinHashSigSize)
	if len(shingles) < MinHashSigSize {
		// 少于 k 个 shingle，重复填充
		for i := range sig {
			sig[i] = shingles[i%len(shingles)]
		}
	} else {
		copy(sig, shingles[:MinHashSigSize])
	}

	// 确保签名单调递增（便于比对）
	sort.Slice(sig, func(i, j int) bool { return sig[i] < sig[j] })

	return sig
}

// CosineSimilarityFromSig 基于签名的近似余弦相似度
// 用于 MinHash 签名之间的快速比对
func CosineSimilarityFromSig(a, b []uint64) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dotProd, normA, normB float64
	for i := range a {
		// 将 uint64 哈希值映射到 [-1, 1] 范围作为伪向量
		va := math.Copysign(1.0, float64(a[i]&1)-0.5)
		vb := math.Copysign(1.0, float64(b[i]&1)-0.5)
		dotProd += va * vb
		normA += va * va
		normB += vb * vb
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dotProd / (math.Sqrt(normA) * math.Sqrt(normB))
}
