package storage

import (
	"database/sql"

	"github.com/lh123aa/cortex/internal/models"
)

// SaveDocument 保存或更新文档记录
func (s *SQLiteStorage) SaveDocument(doc *models.Document) error {
	query := `
		INSERT OR REPLACE INTO documents 
		(id, path, title, file_type, content_hash, file_size, chunk_count, status, indexed_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.Exec(query,
		doc.ID, doc.Path, doc.Title, doc.FileType, doc.ContentHash,
		doc.FileSize, doc.ChunkCount, doc.Status, doc.IndexedAt, doc.UpdatedAt,
	)
	return err
}

// GetDocumentByPath 根据路径获取文档
func (s *SQLiteStorage) GetDocumentByPath(path string) (*models.Document, error) {
	row := s.db.QueryRow(`SELECT id, path, title, file_type, content_hash, file_size, chunk_count, status FROM documents WHERE path = ?`, path)
	var doc models.Document
	err := row.Scan(&doc.ID, &doc.Path, &doc.Title, &doc.FileType, &doc.ContentHash, &doc.FileSize, &doc.ChunkCount, &doc.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &doc, err
}

// GetDocumentByID 根据 ID 获取文档
func (s *SQLiteStorage) GetDocumentByID(id string) (*models.Document, error) {
	row := s.db.QueryRow(`SELECT id, path, title, file_type, content_hash, file_size, chunk_count, status FROM documents WHERE id = ?`, id)
	var doc models.Document
	err := row.Scan(&doc.ID, &doc.Path, &doc.Title, &doc.FileType, &doc.ContentHash, &doc.FileSize, &doc.ChunkCount, &doc.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &doc, err
}

// DeleteDocumentByID 删除文档
func (s *SQLiteStorage) DeleteDocument(id string) error {
	_, err := s.db.Exec(`DELETE FROM documents WHERE id = ?`, id)
	return err
}

// DeleteDocumentByPath 按照路径删除文档
func (s *SQLiteStorage) DeleteDocumentByPath(path string) error {
	_, err := s.db.Exec(`DELETE FROM documents WHERE path = ?`, path)
	return err
}

// ListDocuments 遍历文档
func (s *SQLiteStorage) ListDocuments(limit, offset int) ([]*models.Document, error) {
	rows, err := s.db.Query(`SELECT id, path, title, file_type, content_hash, file_size, chunk_count, status FROM documents LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*models.Document
	for rows.Next() {
		var doc models.Document
		if err := rows.Scan(&doc.ID, &doc.Path, &doc.Title, &doc.FileType, &doc.ContentHash, &doc.FileSize, &doc.ChunkCount, &doc.Status); err != nil {
			return nil, err
		}
		docs = append(docs, &doc)
	}
	return docs, nil
}

// SaveChunks 批量保存块
func (s *SQLiteStorage) SaveChunks(chunks []*models.Chunk) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmtChunk, err := tx.Prepare(`
		INSERT OR REPLACE INTO chunks
		(id, document_id, heading_path, heading_level, content, content_raw, line_start, line_end, char_start, char_end, token_count, embedding_model)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmtChunk.Close()

	stmtVec, err := tx.Prepare(`
		INSERT OR REPLACE INTO vectors (chunk_id, embedding, dimension, model) VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmtVec.Close()

	for _, c := range chunks {
		// 1. chunks 表插入 (触发器会自动同步 FTS)
		_, err := stmtChunk.Exec(c.ID, c.DocumentID, c.HeadingPath, c.HeadingLevel, c.Content, c.ContentRaw, c.LineStart, c.LineEnd, c.CharStart, c.CharEnd, c.TokenCount, c.EmbeddingModel)
		if err != nil {
			return err
		}

		// 2. 向量 表插入 (v1.1: 升级为纯二进制写入，抛弃JSON反序列化开销)
		if len(c.Embedding) > 0 {
			embData := Float32ArrayToBytes(c.Embedding)
			if _, err := stmtVec.Exec(c.ID, embData, len(c.Embedding), c.EmbeddingModel); err != nil {
				return err
			}

			// 3. 同时更新 HNSW 索引 (如果索引已构建)
			if s.useHNSW && s.hnsw != nil {
				s.hnsw.Add(c.ID, c.Embedding)
			}
		}
	}

	return tx.Commit()
}

// GetChunk 获取单个块
func (s *SQLiteStorage) GetChunk(id string) (*models.Chunk, error) {
	row := s.db.QueryRow(`SELECT id, document_id, heading_path, heading_level, content, content_raw, token_count FROM chunks WHERE id = ?`, id)
	var c models.Chunk
	err := row.Scan(&c.ID, &c.DocumentID, &c.HeadingPath, &c.HeadingLevel, &c.Content, &c.ContentRaw, &c.TokenCount)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

// DeleteChunksByDocument 清理旧区块
func (s *SQLiteStorage) DeleteChunksByDocument(docID string) error {
	// 先获取要删除的 chunk IDs（用于更新 HNSW 索引）
	var chunkIDs []string
	rows, err := s.db.Query(`SELECT id FROM chunks WHERE document_id = ?`, docID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id string
			if rows.Scan(&id) == nil {
				chunkIDs = append(chunkIDs, id)
			}
		}
	}

	// 执行删除
	_, err = s.db.Exec(`DELETE FROM chunks WHERE document_id = ?`, docID)
	if err != nil {
		return err
	}

	// 从 HNSW 索引中移除
	if s.useHNSW && s.hnsw != nil {
		for _, id := range chunkIDs {
			s.hnsw.Remove(id)
		}
	}

	return nil
}

// GetMetadata
func (s *SQLiteStorage) GetMetadata(key string) (string, error) {
	var val string
	err := s.db.QueryRow(`SELECT value FROM metadata WHERE key = ?`, key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, err
}

// SetMetadata
func (s *SQLiteStorage) SetMetadata(key, value string) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)`, key, value)
	return err
}

// GetDocumentsCount returns total number of documents
func (s *SQLiteStorage) GetDocumentsCount() (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM documents`).Scan(&count)
	return count, err
}

// GetChunksCount returns total number of chunks
func (s *SQLiteStorage) GetChunksCount() (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM chunks`).Scan(&count)
	return count, err
}

// GetVectorsCount returns total number of vectors
func (s *SQLiteStorage) GetVectorsCount() (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM vectors`).Scan(&count)
	return count, err
}
