package storage

import (
	"math"
	"sort"
	"strings"

	"github.com/lh123aa/cortex/internal/models"
)

// FTSSearch 进行基于 FTS5 的全文关键词检索 (BM25)
// userID 参数用于用户数据隔离
func (s *SQLiteStorage) FTSSearch(query string, userID string, topK int) ([]*models.SearchResult, error) {
	// FTS5 按 bm25 分数倒序，同时通过 document.user_id 隔离用户数据
	q := `
		SELECT c.id, c.document_id, c.heading_path, c.content, c.content_raw, bm25(chunks_fts) as score
		FROM chunks_fts fts
		JOIN chunks c ON c.rowid = fts.rowid
		JOIN documents d ON c.document_id = d.id
		WHERE chunks_fts MATCH ?
		  AND d.user_id = ?
		ORDER BY score LIMIT ?
	`
	rows, err := s.db.Query(q, query, userID, topK)
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
// v2.1: HNSW 失败时自动降级到 brute-force
// v2.2: 添加用户隔离
func (s *SQLiteStorage) VectorSearch(queryVector []float32, userID string, topK int) ([]*models.SearchResult, error) {
	// 如果 HNSW 索引可用，尝试使用 HNSW 搜索
	if s.useHNSW && s.hnsw != nil {
		results, err := s.vectorSearchHNSW(queryVector, userID, topK)
		if err == nil && len(results) > 0 {
			return results, nil
		}
		// HNSW 失败，触发降级
		s.logDegraded("hnsw", err)
	}

	// 降级到旧的暴力搜索
	return s.vectorSearchBruteForce(queryVector, userID, topK)
}

// logDegraded 记录降级事件（未来可通过 metrics 暴露）
func (s *SQLiteStorage) logDegraded(reason string, err error) {
	// 目前只是日志记录，未来可添加 metrics 计数
	// metrics.SearchDegraded.Inc()
	_ = reason
	_ = err
}

// vectorSearchHNSW 使用 HNSW 索引搜索（带用户隔离）
func (s *SQLiteStorage) vectorSearchHNSW(queryVector []float32, userID string, topK int) ([]*models.SearchResult, error) {
	// HNSW 搜索 - 需要在应用层做用户隔离
	// 因为 HNSW 索引不存储 user_id，我们需要在获取结果后过滤
	ids, distances := s.hnsw.Search(queryVector, topK*3) // 多取一些，后续过滤

	if len(ids) == 0 {
		return nil, nil
	}

	// 构建批量查询获取 chunk 和对应的 user_id
	results := make([]*models.SearchResult, 0, topK)
	chunkIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		chunkIDs = append(chunkIDs, id)
	}

	// 批量获取 chunks 和 documents 来过滤 user_id
	if len(chunkIDs) > 0 {
		// 构建参数化查询（防止 SQL 注入）
		q := `
			SELECT c.id, c.document_id, c.heading_path, c.content, c.content_raw, d.user_id
			FROM chunks c
			JOIN documents d ON c.document_id = d.id
			WHERE c.id IN (?` + strings.Repeat(",?", len(chunkIDs)-1) + `)
		`
		// 将 chunkIDs 作为参数传递
		args := make([]interface{}, len(chunkIDs))
		for i, id := range chunkIDs {
			args[i] = id
		}
		rows, err := s.db.Query(q, args...)
		if err != nil {
			return nil, err
		}

		// 建立 id -> chunk 映射
		chunkMap := make(map[string]*models.Chunk)
		for rows.Next() {
			var chunk models.Chunk
			var docUserID string
			if err := rows.Scan(&chunk.ID, &chunk.DocumentID, &chunk.HeadingPath, &chunk.Content, &chunk.ContentRaw, &docUserID); err != nil {
				rows.Close()
				return nil, err
			}
			chunkMap[chunk.ID] = &chunk
		}
		rows.Close()

		// 按原始顺序处理，仅保留匹配 userID 的结果
		for i, id := range ids {
			chunk, ok := chunkMap[id]
			if !ok {
				continue
			}
			// 用户隔离：检查 document 的 user_id
			doc, err := s.GetDocumentByID(chunk.DocumentID, userID)
			if err != nil || doc == nil {
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
	}

	// 截取 topK
	if len(results) > topK {
		results = results[:topK]
	}

	return results, nil
}

// vectorSearchBruteForce 回退方案：暴力搜索（带用户隔离）
func (s *SQLiteStorage) vectorSearchBruteForce(queryVector []float32, userID string, topK int) ([]*models.SearchResult, error) {
	const batchSize = 1000
	offset := 0

	// 维护 TopK 缓冲池
	var topResults []*models.SearchResult

	for {
		// 通过 documents 表过滤 user_id，确保用户只能搜索自己的数据
		q := `
			SELECT v.chunk_id, v.embedding, c.document_id, c.heading_path, c.content, c.content_raw
			FROM vectors v
			JOIN chunks c ON v.chunk_id = c.id
			JOIN documents d ON c.document_id = d.id
			WHERE d.user_id = ?
			LIMIT ? OFFSET ?
		`
		rows, err := s.db.Query(q, userID, batchSize, offset)
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

// joinStrings 将字符串切片拼接为 SQL IN 子句的字符串列表
func joinStrings(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	result := ""
	for _, id := range ids {
		if result != "" {
			result += "','"
		}
		result += id
	}
	return result
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
