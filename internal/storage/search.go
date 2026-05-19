package storage

import (
	"math"
	"sort"
	"strings"
	"unicode"

	"github.com/lh123aa/cortex/internal/models"
	"github.com/lh123aa/cortex/internal/vector"
)

// isCJKRune 判断是否为中日韩文字
func isCJKRune(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		(r >= 0x3040 && r <= 0x309F) || // Hiragana
		(r >= 0x30A0 && r <= 0x30FF) // Katakana
}

// expandChineseQuery 对中文搜索词进行单字展开
// 之前使用 2-gram 导致搜索词和索引不一致（如搜"商品价格"→"商品 品价 价格"，
// 但索引中是"商品 品销 销售..."，"品价"不存在→全不命中）
// 改为单字展开后：搜索"商品价格"→"商 品 价 格"，索引"商 品 销 售 价 格"
// FTS5 AND 模式下完美匹配
func expandChineseQuery(query string) string {
	hasCJK := false
	for _, r := range query {
		if isCJKRune(r) {
			hasCJK = true
			break
		}
	}
	if !hasCJK {
		return query
	}

	// 单字展开，每个中文字单独作为FTS5词项
	runes := []rune(query)
	var result strings.Builder
	for i, r := range runes {
		if isCJKRune(r) {
			if i > 0 {
				result.WriteByte(' ')
			}
			result.WriteRune(r)
		} else {
			if result.Len() > 0 && !strings.HasSuffix(result.String(), " ") {
				result.WriteByte(' ')
			}
			result.WriteRune(r)
		}
	}
	return result.String()
}

// FTSSearch 进行基于 FTS5 的全文关键词检索 (BM25)
// userID 参数用于用户数据隔离
func (s *SQLiteStorage) FTSSearch(query string, userID string, topK int) ([]*models.SearchResult, error) {
	// 对中文 query 做 2-gram 展开，匹配索引时同样展开的内容
	ftsQuery := expandChineseQuery(query)
	q := `
		SELECT c.id, c.document_id, c.heading_path, c.content, c.content_raw, bm25(chunks_fts) as score
		FROM chunks_fts fts
		JOIN chunks c ON c.rowid = fts.rowid
		JOIN documents d ON c.document_id = d.id
		WHERE chunks_fts MATCH ?
		  AND d.user_id = ?
		ORDER BY score LIMIT ?
	`
	rows, err := s.db.Query(q, ftsQuery, userID, topK)
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

		// SQLite BM25: 分数越低表示越相关（0 = 完美匹配）
		// 转为 0-1 范围的相似度分数，便于与向量分数融合
		normalizedScore := 1.0 / (1.0 + math.Abs(rawScore))

		results = append(results, &models.SearchResult{
			Chunk:    &chunk,
			Score:    normalizedScore,
			FTSScore: normalizedScore,
		})
	}
	return results, nil
}

// VectorSearch 使用内存向量索引进行搜索
// v3.0: 使用轻量级内存索引替代 HNSW（HNSW 在 1万+ 向量规模下构建不稳定）
func (s *SQLiteStorage) VectorSearch(queryVector []float32, userID string, topK int) ([]*models.SearchResult, error) {
	// 如果内存向量索引可用，使用内存搜索（最快）
	if s.flatReady {
		results, err := s.vectorSearchInMemory(queryVector, userID, topK)
		if err == nil && len(results) > 0 {
			return results, nil
		}
	}

	// 回退到数据库暴力搜索
	return s.vectorSearchBruteForce(queryVector, userID, topK)
}

// vectorSearchInMemory 使用内存向量索引进行暴力搜索（带用户隔离）
func (s *SQLiteStorage) vectorSearchInMemory(queryVector []float32, userID string, topK int) ([]*models.SearchResult, error) {
	if !s.flatReady || len(s.flatChunkIDs) == 0 {
		return nil, nil
	}

	// 对所有向量计算余弦相似度
	type scored struct {
		chunkID     string
		similarity  float64
	}

	topResults := make([]scored, 0, topK+1)

	for i, vec := range s.flatEmbeds {
		sim := vector.CosineSimilarity(queryVector, vec)

		if len(topResults) < topK {
			topResults = append(topResults, scored{chunkID: s.flatChunkIDs[i], similarity: sim})
			// 按 similarity 降序排序
			for j := len(topResults) - 1; j > 0 && topResults[j].similarity > topResults[j-1].similarity; j-- {
				topResults[j], topResults[j-1] = topResults[j-1], topResults[j]
			}
		} else if sim > topResults[topK-1].similarity {
			topResults[topK-1] = scored{chunkID: s.flatChunkIDs[i], similarity: sim}
			for j := topK - 1; j > 0 && topResults[j].similarity > topResults[j-1].similarity; j-- {
				topResults[j], topResults[j-1] = topResults[j-1], topResults[j]
			}
		}
	}

	if len(topResults) == 0 {
		return nil, nil
	}

	// 批量查询 chunk 信息并过滤 userID
	chunkIDs := make([]string, len(topResults))
	for i, r := range topResults {
		chunkIDs[i] = r.chunkID
	}

	q := `SELECT c.id, c.document_id, c.heading_path, c.content, c.content_raw, d.user_id
		FROM chunks c
		JOIN documents d ON c.document_id = d.id
		WHERE c.id IN (?` + strings.Repeat(",?", len(chunkIDs)-1) + `)`

	args := make([]interface{}, len(chunkIDs))
	for i, id := range chunkIDs {
		args[i] = id
	}

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chunkMap := make(map[string]*models.Chunk)
	for rows.Next() {
		var chunk models.Chunk
		var docUserID string
		if err := rows.Scan(&chunk.ID, &chunk.DocumentID, &chunk.HeadingPath, &chunk.Content, &chunk.ContentRaw, &docUserID); err != nil {
			return nil, err
		}
		chunkMap[chunk.ID] = &chunk
	}

	results := make([]*models.SearchResult, 0, topK)
	for _, r := range topResults {
		chunk, ok := chunkMap[r.chunkID]
		if !ok {
			continue
		}
		doc, err := s.GetDocumentByID(chunk.DocumentID, userID)
		if err != nil || doc == nil {
			continue
		}
		results = append(results, &models.SearchResult{
			Chunk:       chunk,
			Score:       r.similarity,
			VectorScore: r.similarity,
		})
	}

	return results, nil
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
		defer rows.Close()

		// 建立 id -> chunk 映射
		chunkMap := make(map[string]*models.Chunk)
		for rows.Next() {
			var chunk models.Chunk
			var docUserID string
			if err := rows.Scan(&chunk.ID, &chunk.DocumentID, &chunk.HeadingPath, &chunk.Content, &chunk.ContentRaw, &docUserID); err != nil {
				return nil, err
			}
			chunkMap[chunk.ID] = &chunk
		}

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

			similarity := vector.CosineSimilarity(queryVector, chunkVec)

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


