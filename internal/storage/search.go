package storage

import (
	"math"
	"sort"

	"github.com/lh123aa/cortex/internal/models"
)

// FTSSearch 进行基于 FTS5 的 全文关键词检索 (BM25)
func (s *SQLiteStorage) FTSSearch(query string, topK int) ([]*models.SearchResult, error) {
	// FTS5 按 bm25 分数倒序
	q := `
		SELECT c.id, c.document_id, c.heading_path, c.content, c.content_raw, bm25(chunks_fts) as score
		FROM chunks_fts fts
		JOIN chunks c ON c.rowid = fts.rowid
		WHERE chunks_fts MATCH ?
		ORDER BY score LIMIT ?
	`
	rows, err := s.db.Query(q, query, topK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.SearchResult
	for rows.Next() {
		var chunk models.Chunk
		var rawScore float64
		if err := rows.Scan(&chunk.ID, &chunk.DocumentID, &chunk.HeadingPath, &chunk.Content, &chunk.ContentRaw, &rawScore); err != nil {
			return nil, err
		}

		// sqlite FTS bm25 约小越相关，但通常可简单转换为正相关的分数，此处直接反转使其为正
		// (如果 sqlite 默认 bm25() 是负数代表高相关，可乘以 -1)
		positiveScore := math.Abs(rawScore) 

		results = append(results, &models.SearchResult{
			Chunk:    &chunk,
			Score:    positiveScore,
			FTSScore: positiveScore,
		})
	}
	return results, nil
}

// VectorSearch 载入内存计算 Cosine 相似度 (v1.1: 采用边读边推的 Bounded Slice 容量限制, 防御全表扫入内存的 OOM 问题)
func (s *SQLiteStorage) VectorSearch(queryVector []float32, topK int) ([]*models.SearchResult, error) {
	q := `
		SELECT v.chunk_id, v.embedding, c.document_id, c.heading_path, c.content, c.content_raw
		FROM vectors v
		JOIN chunks c ON v.chunk_id = c.id
	`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 维护 TopK 缓冲池
	var topResults []*models.SearchResult

	for rows.Next() {
		var chunkID string
		var embeddingData []byte
		chunk := models.Chunk{}

		if err := rows.Scan(&chunkID, &embeddingData, &chunk.DocumentID, &chunk.HeadingPath, &chunk.Content, &chunk.ContentRaw); err != nil {
			return nil, err
		}
		chunk.ID = chunkID

		// v1.1 极速二进制解码
		chunkVec := BytesToFloat32Array(embeddingData)
		if len(chunkVec) == 0 {
			continue
		}

		similarity := cosineSimilarity(queryVector, chunkVec)

		// 边界优化插入逻辑：如果是符合入榜标准的结果，立刻存入缓冲并踢掉最后一名
		if len(topResults) < topK {
			topResults = insertSorted(topResults, &chunk, similarity)
		} else if similarity > topResults[topK-1].Score {
			topResults = insertSorted(topResults, &chunk, similarity)[:topK]
		}
	}

	return topResults, nil
}

// insertSorted 按分数倒序插入切片
func insertSorted(res []*models.SearchResult, chunk *models.Chunk, sim float64) []*models.SearchResult {
	item := &models.SearchResult{
		Chunk:       chunk,
		Score:       sim,
		VectorScore: sim,
	}
	// 找到应当插入的位置
	index := sort.Search(len(res), func(i int) bool {
		return res[i].Score <= sim
	})
	
	// 扩容并插入
	res = append(res, nil)
	copy(res[index+1:], res[index:])
	res[index] = item
	return res
}


// cosineSimilarity 计算余弦相似度
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dotProduct, normA, normB float64
	for i := range a {
		valA := float64(a[i])
		valB := float64(b[i])
		dotProduct += valA * valB
		normA += valA * valA
		normB += valB * valB
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
