package storage

import (
	"database/sql"
	"fmt"

	"github.com/lh123aa/cortex/internal/models"
)

// SaveDocument 保存或更新文档记录（用户隔离）
func (s *SQLiteStorage) SaveDocument(doc *models.Document) error {
	query := `
		INSERT OR REPLACE INTO documents
		(id, user_id, path, title, file_type, content_hash, file_size, chunk_count, status, indexed_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.Exec(query,
		doc.ID, doc.UserID, doc.Path, doc.Title, doc.FileType, doc.ContentHash,
		doc.FileSize, doc.ChunkCount, doc.Status, doc.IndexedAt, doc.UpdatedAt,
	)
	return err
}

// GetDocumentByPath 根据路径获取文档（用户隔离）
func (s *SQLiteStorage) GetDocumentByPath(path string, userID string) (*models.Document, error) {
	row := s.db.QueryRow(`
		SELECT id, user_id, path, title, file_type, content_hash, file_size, chunk_count, status
		FROM documents WHERE path = ? AND user_id = ?`,
		path, userID)
	var doc models.Document
	err := row.Scan(&doc.ID, &doc.UserID, &doc.Path, &doc.Title, &doc.FileType,
		&doc.ContentHash, &doc.FileSize, &doc.ChunkCount, &doc.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &doc, err
}

// GetDocumentByID 根据 ID 获取文档（用户隔离）
func (s *SQLiteStorage) GetDocumentByID(id string, userID string) (*models.Document, error) {
	row := s.db.QueryRow(`
		SELECT id, user_id, path, title, file_type, content_hash, file_size, chunk_count, status
		FROM documents WHERE id = ? AND user_id = ?`,
		id, userID)
	var doc models.Document
	err := row.Scan(&doc.ID, &doc.UserID, &doc.Path, &doc.Title, &doc.FileType,
		&doc.ContentHash, &doc.FileSize, &doc.ChunkCount, &doc.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &doc, err
}

// GetDocumentByIDAnyUser 根据 ID 获取文档（管理员用，不过滤用户）
func (s *SQLiteStorage) GetDocumentByIDAnyUser(id string) (*models.Document, error) {
	row := s.db.QueryRow(`
		SELECT id, user_id, path, title, file_type, content_hash, file_size, chunk_count, status
		FROM documents WHERE id = ?`,
		id)
	var doc models.Document
	err := row.Scan(&doc.ID, &doc.UserID, &doc.Path, &doc.Title, &doc.FileType,
		&doc.ContentHash, &doc.FileSize, &doc.ChunkCount, &doc.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &doc, err
}

// DeleteDocument 删除文档（用户隔离）
func (s *SQLiteStorage) DeleteDocument(id string, userID string) error {
	result, err := s.db.Exec(`DELETE FROM documents WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("document not found or access denied")
	}
	return nil
}

// DeleteDocumentByPath 按照路径删除文档（用户隔离）
func (s *SQLiteStorage) DeleteDocumentByPath(path string, userID string) error {
	result, err := s.db.Exec(`DELETE FROM documents WHERE path = ? AND user_id = ?`, path, userID)
	if err != nil {
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("document not found or access denied")
	}
	return nil
}

// ListDocuments 遍历文档（用户隔离）
func (s *SQLiteStorage) ListDocuments(userID string, limit, offset int) ([]*models.Document, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, path, title, file_type, content_hash, file_size, chunk_count, status
		FROM documents WHERE user_id = ? ORDER BY indexed_at DESC LIMIT ? OFFSET ?`,
		userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*models.Document
	for rows.Next() {
		var doc models.Document
		if err := rows.Scan(&doc.ID, &doc.UserID, &doc.Path, &doc.Title, &doc.FileType,
			&doc.ContentHash, &doc.FileSize, &doc.ChunkCount, &doc.Status); err != nil {
			return nil, err
		}
		docs = append(docs, &doc)
	}
	return docs, nil
}

// ListAllDocuments 遍历所有文档（管理员用，不隔离）
func (s *SQLiteStorage) ListAllDocuments(limit, offset int) ([]*models.Document, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, path, title, file_type, content_hash, file_size, chunk_count, status
		FROM documents ORDER BY indexed_at DESC LIMIT ? OFFSET ?`,
		limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*models.Document
	for rows.Next() {
		var doc models.Document
		if err := rows.Scan(&doc.ID, &doc.UserID, &doc.Path, &doc.Title, &doc.FileType,
			&doc.ContentHash, &doc.FileSize, &doc.ChunkCount, &doc.Status); err != nil {
			return nil, err
		}
		docs = append(docs, &doc)
	}
	return docs, nil
}

// GetDocumentsCount 获取文档数量（用户隔离）
func (s *SQLiteStorage) GetDocumentsCount(userID string) (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM documents WHERE user_id = ?`, userID).Scan(&count)
	return count, err
}

// SaveChunks 批量保存块（用户隔离）
func (s *SQLiteStorage) SaveChunks(chunks []*models.Chunk) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmtChunk, err := tx.Prepare(`
		INSERT OR REPLACE INTO chunks
		(id, user_id, document_id, heading_path, heading_level, content, content_raw, line_start, line_end, char_start, char_end, token_count, embedding_model)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmtChunk.Close()

	stmtVec, err := tx.Prepare(`
		INSERT OR REPLACE INTO vectors (chunk_id, user_id, embedding, dimension, model) VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmtVec.Close()

	for _, c := range chunks {
		// 1. chunks 表插入 (触发器会自动同步 FTS)
		_, err := stmtChunk.Exec(c.ID, c.UserID, c.DocumentID, c.HeadingPath, c.HeadingLevel,
			c.Content, c.ContentRaw, c.LineStart, c.LineEnd, c.CharStart, c.CharEnd, c.TokenCount, c.EmbeddingModel)
		if err != nil {
			return err
		}

		// 2. 向量 表插入 (v1.1: 升级为纯二进制写入，抛弃JSON反序列化开销)
		if len(c.Embedding) > 0 {
			embData := Float32ArrayToBytes(c.Embedding)
			if _, err := stmtVec.Exec(c.ID, c.UserID, embData, len(c.Embedding), c.EmbeddingModel); err != nil {
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

// GetChunk 获取单个块（用户隔离）
func (s *SQLiteStorage) GetChunk(id string, userID string) (*models.Chunk, error) {
	row := s.db.QueryRow(`
		SELECT c.id, c.user_id, c.document_id, c.heading_path, c.heading_level, c.content, c.content_raw, c.token_count
		FROM chunks c
		JOIN documents d ON c.document_id = d.id
		WHERE c.id = ? AND d.user_id = ?`,
		id, userID)
	var c models.Chunk
	err := row.Scan(&c.ID, &c.UserID, &c.DocumentID, &c.HeadingPath, &c.HeadingLevel,
		&c.Content, &c.ContentRaw, &c.TokenCount)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

// GetChunkAnyUser 获取单个块（管理员用，不过滤用户）
func (s *SQLiteStorage) GetChunkAnyUser(id string) (*models.Chunk, error) {
	row := s.db.QueryRow(`
		SELECT id, user_id, document_id, heading_path, heading_level, content, content_raw, token_count
		FROM chunks WHERE id = ?`,
		id)
	var c models.Chunk
	err := row.Scan(&c.ID, &c.UserID, &c.DocumentID, &c.HeadingPath, &c.HeadingLevel,
		&c.Content, &c.ContentRaw, &c.TokenCount)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

// DeleteChunksByDocument 清理旧区块（用户隔离）
func (s *SQLiteStorage) DeleteChunksByDocument(docID string, userID string) error {
	// 先获取要删除的 chunk IDs（用于更新 HNSW 索引）
	var chunkIDs []string
	rows, err := s.db.Query(`
		SELECT c.id FROM chunks c
		JOIN documents d ON c.document_id = d.id
		WHERE c.document_id = ? AND d.user_id = ?`,
		docID, userID)
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
	_, err = s.db.Exec(`
		DELETE FROM chunks WHERE document_id = ?
		AND document_id IN (SELECT id FROM documents WHERE user_id = ?)`,
		docID, userID)
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

// GetMetadata 获取元数据
func (s *SQLiteStorage) GetMetadata(key string) (string, error) {
	var val string
	err := s.db.QueryRow(`SELECT value FROM metadata WHERE key = ?`, key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, err
}

// SetMetadata 设置元数据
func (s *SQLiteStorage) SetMetadata(key, value string) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)`, key, value)
	return err
}

// GetDocumentsCountAnyUser returns total number of documents (admin)
func (s *SQLiteStorage) GetDocumentsCountAnyUser() (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM documents`).Scan(&count)
	return count, err
}

// GetChunksCountAnyUser returns total number of chunks (admin)
func (s *SQLiteStorage) GetChunksCountAnyUser() (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM chunks`).Scan(&count)
	return count, err
}

// GetVectorsCountAnyUser returns total number of vectors (admin)
func (s *SQLiteStorage) GetVectorsCountAnyUser() (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM vectors`).Scan(&count)
	return count, err
}