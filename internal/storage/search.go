package storage

import (
	"math"
	"sort"

	"github.com/lh123aa/cortex/internal/models"
	"github.com/lh123aa/cortex/internal/vector"
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

// VectorSearch 使用 HNSW 索引进行向量搜索
// v2.0: 从 O(n) 全表扫描优化为 O(log n) HNSW 近似最近邻搜索
func (s *SQLiteStorage) VectorSearch(queryVector []float32, topK int) ([]*models.SearchResult, error) {
	// 如果 HNSW 索引可用，使用 HNSW 搜索
	if s.useHNSW && s.hnsw != nil {
		return s.vectorSearchHNSW(queryVector, topK)
	}

	// 否则回退到旧的暴力搜索
	return s.vectorSearchBruteForce(queryVector, topK)
}

// vectorSearchHNSW 使用 HNSW 索引搜索
func (s *SQLiteStorage) vectorSearchHNSW(queryVector []float32, topK int) ([]*models.SearchResult, error) {
	// HNSW 搜索
	ids, distances := s.hnsw.Search(queryVector, topK)

	if len(ids) == 0 {
		return nil, nil
	}

	// 批量获取 chunk 信息
	results := make([]*models.SearchResult, 0, len(ids))
	for i, id := range ids {
		chunk, err := s.GetChunk(id)
		if err != nil || chunk == nil {
			continue
		}
		// 距离转换为相似度 (1 - distance)
		similarity := 1.0 - distances[i]
		results = append(results, &models.SearchResult{
			Chunk:       chunk,
			Score:       similarity,
			VectorScore: similarity,
		})
	}

	return results, nil
}

// vectorSearchBruteForce 回退方案：暴力搜索
func (s *SQLiteStorage) vectorSearchBruteForce(queryVector []float32, topK int) ([]*models.SearchResult, error) {
	const batchSize = 1000
	offset := 0

	// 维护 TopK 缓冲池
	var topResults []*models.SearchResult

	for {
		q := `
			SELECT v.chunk_id, v.embedding, c.document_id, c.heading_path, c.content, c.content_raw
			FROM vectors v
			JOIN chunks c ON v.chunk_id = c.id
			LIMIT ? OFFSET ?
		`
		rows, err := s.db.Query(q, batchSize, offset)
		if err != nil {
			return nil, err
		}

		hasRows := false
		for rows.Next() {
			hasRows = true
			var chunkID string
			var embeddingData []byte
			chunk := models.Chunk{}

			if err := rows.Scan(&chunkID, &embeddingData, &chunk.DocumentID, &chunk.HeadingPath, &chunk.Content, &chunk.ContentRaw); err != nil {
				rows.Close()
				return nil, err
			}
			chunk.ID = chunkID

			// 极速二进制解码
			chunkVec := BytesToFloat32Array(embeddingData)
			if len(chunkVec) == 0 {
				continue
			}

			similarity := cosineSimilarity(queryVector, chunkVec)

			// 边界优化插入逻辑
			if len(topResults) < topK {
				topResults = insertSorted(topResults, &chunk, similarity)
			} else if similarity > topResults[topK-1].Score {
				topResults = insertSorted(topResults, &chunk, similarity)[:topK]
			}
		}
		rows.Close()

		if !hasRows {
			break
		}

		offset += batchSize
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
