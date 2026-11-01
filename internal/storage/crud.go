package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lh123aa/cortex/internal/models"
	"github.com/lh123aa/cortex/internal/vector"
)

// SaveDocument 保存或更新文档记录（用户隔离）
func (s *SQLiteStorage) SaveDocument(doc *models.Document) error {
	// 时间戳兜底：调用方（如索引器）可能没有设置 IndexedAt/UpdatedAt，
	// 早期版本会因此把 0001-01-01 写进库，导致「最近索引」列表显示成远古时间。
	if doc.IndexedAt.IsZero() {
		doc.IndexedAt = time.Now()
	}
	if doc.UpdatedAt.IsZero() {
		doc.UpdatedAt = doc.IndexedAt
	}
	if doc.Status == "" {
		doc.Status = "indexed"
	}

	// 显式以 UTC RFC3339 字符串写入：
	// 直接传 time.Time 时驱动会写成 "2026-10-22 17:15:20.250834 +0300 +03" 这类
	// 非标准格式（含单调时钟/双时区后缀），跨平台回读时无法可靠解析。
	indexedAt := doc.IndexedAt.UTC().Format(time.RFC3339Nano)
	updatedAt := doc.UpdatedAt.UTC().Format(time.RFC3339Nano)
	doc.IndexedAt = doc.IndexedAt.UTC()
	doc.UpdatedAt = doc.UpdatedAt.UTC()

	query := `
		INSERT OR REPLACE INTO documents
		(id, user_id, path, title, file_type, content_hash, file_size, chunk_count, status, indexed_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.Exec(query,
		doc.ID, doc.UserID, doc.Path, doc.Title, doc.FileType, doc.ContentHash,
		doc.FileSize, doc.ChunkCount, doc.Status, indexedAt, updatedAt,
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
		SELECT id, user_id, path, title, file_type, content_hash, file_size, chunk_count, status,
		       COALESCE(indexed_at, ''), COALESCE(updated_at, '')
		FROM documents WHERE user_id = ? ORDER BY indexed_at DESC LIMIT ? OFFSET ?`,
		userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*models.Document
	for rows.Next() {
		var doc models.Document
		var indexedAt, updatedAt string
		if err := rows.Scan(&doc.ID, &doc.UserID, &doc.Path, &doc.Title, &doc.FileType,
			&doc.ContentHash, &doc.FileSize, &doc.ChunkCount, &doc.Status,
			&indexedAt, &updatedAt); err != nil {
			return nil, err
		}
		doc.IndexedAt = parseSQLiteTime(indexedAt)
		doc.UpdatedAt = parseSQLiteTime(updatedAt)
		docs = append(docs, &doc)
	}
	return docs, nil
}

// parseSQLiteTime 解析 SQLite 中常见的时间字符串形式。
// modernc.org/sqlite 以 TEXT 存储时间，格式可能是 RFC3339、"2006-01-02 15:04:05" 或
// 只有日期；解析失败时返回零值，由调用方决定如何展示。
func parseSQLiteTime(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05.999999999 -0700",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t
		}
	}
	return time.Time{}
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
		(id, user_id, document_id, content_hash, minhash_sig, heading_path, heading_level, content, content_raw, line_start, line_end, char_start, char_end, token_count, embedding_model)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
		var minhashData []byte
		if len(c.Embedding) > 0 {
			mh := vector.NewMinHash(c.ContentRaw)
			minhashData = mh.Bytes()
		}
		_, err := stmtChunk.Exec(c.ID, c.UserID, c.DocumentID, c.ContentHash, minhashData, c.HeadingPath, c.HeadingLevel,
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

// GetChunkByHash 根据内容哈希查找已存在的 chunk（用于去重）
func (s *SQLiteStorage) GetChunkByHash(hash string, userID string) (*models.Chunk, error) {
	row := s.db.QueryRow(`
		SELECT c.id, c.user_id, c.document_id, c.content_hash
		FROM chunks c
		WHERE c.content_hash = ? AND c.user_id = ?
		LIMIT 1`, hash, userID)
	var c models.Chunk
	err := row.Scan(&c.ID, &c.UserID, &c.DocumentID, &c.ContentHash)
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

// SaveUser 保存用户到数据库。
//
// 必须用 UPSERT（ON CONFLICT DO UPDATE）而不是 INSERT OR REPLACE：
// REPLACE 会「先删后插」，而 auth_tokens / api_keys 对 users(id) 有
// ON DELETE CASCADE，于是每次更新用户（例如登录时写 last_login_at）
// 都会把他的 token 和 API Key 一起删掉 —— 表现为「登录成功但立刻 401」。
func (s *SQLiteStorage) SaveUser(user *models.User) error {
	if user.Role == "" {
		user.Role = models.RoleEditor
	}
	if user.Tier == "" {
		user.Tier = string(models.TierFree)
	}
	if user.StorageLimitBytes == 0 {
		user.StorageLimitBytes = models.TierFree.StorageLimit()
	}
	now := time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}

	query := `
		INSERT INTO users
		(id, username, password_hash, role, tier, storage_used_bytes, storage_limit_bytes,
		 license_key, created_at, updated_at, last_login_at, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			username = excluded.username,
			password_hash = excluded.password_hash,
			role = excluded.role,
			tier = excluded.tier,
			storage_used_bytes = excluded.storage_used_bytes,
			storage_limit_bytes = excluded.storage_limit_bytes,
			license_key = excluded.license_key,
			updated_at = excluded.updated_at,
			last_login_at = excluded.last_login_at,
			is_active = excluded.is_active
	`
	_, err := s.db.Exec(query,
		user.ID, user.Username, user.PasswordHash, user.Role, user.Tier,
		user.StorageUsedBytes, user.StorageLimitBytes, user.LicenseKey,
		user.CreatedAt, now, nullableTime(user.LastLoginAt), user.IsActive,
	)
	return err
}

// nullableTime 把零值时间转成 nil，避免写入 0001-01-01。
func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// scanUser 统一扫描用户行（字段顺序与 userSelectColumns 一致）。
func scanUser(row interface{ Scan(...any) error }) (*models.User, error) {
	var u models.User
	var createdAt, lastLogin sql.NullString
	err := row.Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.Tier,
		&u.StorageUsedBytes, &u.StorageLimitBytes, &u.LicenseKey,
		&createdAt, &lastLogin, &u.IsActive,
	)
	if err != nil {
		return nil, err
	}
	u.CreatedAt = parseSQLiteTime(nullStringValue(createdAt))
	u.UpdatedAt = u.CreatedAt
	if lastLogin.Valid && lastLogin.String != "" {
		u.LastLoginAt = parseSQLiteTime(lastLogin.String)
	}
	return &u, nil
}

func nullStringValue(s sql.NullString) string {
	if !s.Valid {
		return ""
	}
	return s.String
}

const userSelectColumns = `id, username, password_hash, role,
	COALESCE(tier,'free'), COALESCE(storage_used_bytes,0), COALESCE(storage_limit_bytes,0),
	COALESCE(license_key,''), COALESCE(created_at,''), COALESCE(last_login_at,''),
	COALESCE(is_active,1)`

// GetUserByID 根据 ID 获取用户
func (s *SQLiteStorage) GetUserByID(id string) (*models.User, error) {
	row := s.db.QueryRow(`SELECT `+userSelectColumns+` FROM users WHERE id = ?`, id)
	u, err := scanUser(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

// GetUserByUsername 根据用户名获取用户
func (s *SQLiteStorage) GetUserByUsername(username string) (*models.User, error) {
	row := s.db.QueryRow(`SELECT `+userSelectColumns+` FROM users WHERE username = ?`, username)
	u, err := scanUser(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

// DeleteUser 删除用户
func (s *SQLiteStorage) DeleteUser(id string) error {
	_, err := s.db.Exec(`DELETE FROM users WHERE id = ?`, id)
	return err
}

// ListUsers 列出所有用户（管理员）
func (s *SQLiteStorage) ListUsers(limit, offset int) ([]*models.User, error) {
	rows, err := s.db.Query(`SELECT `+userSelectColumns+` FROM users ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// ListAPIKeys 列出所有 API Keys（管理员）
func (s *SQLiteStorage) ListAPIKeys(limit, offset int) ([]*models.APIKey, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, key_hash, name, last_used, created_at, expires_at
		FROM api_keys
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`,
		limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apiKeys []*models.APIKey
	for rows.Next() {
		var key models.APIKey
		if err := rows.Scan(&key.ID, &key.UserID, &key.KeyHash, &key.Name, &key.LastUsed, &key.CreatedAt, &key.ExpiresAt); err != nil {
			return nil, err
		}
		apiKeys = append(apiKeys, &key)
	}
	return apiKeys, rows.Err()
}

// ListAPIKeysByUser 列出用户的所有 API Keys
func (s *SQLiteStorage) ListAPIKeysByUser(userID string) ([]*models.APIKey, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, key_hash, name, last_used, created_at, expires_at
		FROM api_keys
		WHERE user_id = ?
		ORDER BY created_at DESC`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apiKeys []*models.APIKey
	for rows.Next() {
		var key models.APIKey
		if err := rows.Scan(&key.ID, &key.UserID, &key.KeyHash, &key.Name, &key.LastUsed, &key.CreatedAt, &key.ExpiresAt); err != nil {
			return nil, err
		}
		apiKeys = append(apiKeys, &key)
	}
	return apiKeys, rows.Err()
}

// DeleteUserData 删除用户的所有数据
func (s *SQLiteStorage) DeleteUserData(userID string) error {
	// 删除用户的文档、chunks、向量等
	// 由于外键级联删除，主要删除文档即可
	docs, err := s.ListDocuments(userID, 10000, 0)
	if err != nil {
		return err
	}
	for _, doc := range docs {
		if err := s.DeleteDocument(doc.ID, userID); err != nil {
			return err
		}
	}
	// 删除用户的缓存
	if err := s.InvalidateUserSearchCache(userID); err != nil {
		return err
	}
	return nil
}

// GetChunksCount 获取分块数量（用户隔离）
func (s *SQLiteStorage) GetChunksCount(userID string) (int, error) {
	var count int
	if userID == "" {
		err := s.db.QueryRow(`SELECT COUNT(*) FROM chunks`).Scan(&count)
		return count, err
	}
	err := s.db.QueryRow(`SELECT COUNT(*) FROM chunks WHERE user_id = ?`, userID).Scan(&count)
	return count, err
}

// CalculateStorageUsed 计算用户已使用的存储量（所有文档 file_size 总和）
func (s *SQLiteStorage) CalculateStorageUsed(userID string) (int64, error) {
	var total int64
	err := s.db.QueryRow(`SELECT COALESCE(SUM(file_size), 0) FROM documents WHERE user_id = ?`, userID).Scan(&total)
	return total, err
}

// GetVectorsCount 获取向量数量（用户隔离）
func (s *SQLiteStorage) GetVectorsCount(userID string) (int, error) {
	var count int
	if userID == "" {
		err := s.db.QueryRow(`SELECT COUNT(*) FROM vectors`).Scan(&count)
		return count, err
	}
	err := s.db.QueryRow(`SELECT COUNT(*) FROM vectors WHERE user_id = ?`, userID).Scan(&count)
	return count, err
}

// ========== Token 操作 ==========

// SaveToken 保存认证 token
//
// 存储格式约定：expires_at 统一存 **整数 Unix 秒**。
// 早期实现写入整数、却用 `expires_at > datetime('now')` 比较，
// SQLite 的类型排序是 INTEGER < TEXT，整数永远「小于」文本，
// 于是刚登录拿到的 token 永远查不到 —— 表现为「登录成功但每个请求都 401」。
func (s *SQLiteStorage) SaveToken(token *models.AuthToken) error {
	query := `
		INSERT OR REPLACE INTO auth_tokens
		(token, user_id, username, expires_at)
		VALUES (?, ?, ?, ?)
	`
	expiresAt := time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	_, err := s.db.Exec(query, token.Token, token.UserID, token.Username, expiresAt.Unix())
	return err
}

// GetToken 获取未过期 token
func (s *SQLiteStorage) GetToken(token string) (*models.AuthToken, error) {
	row := s.db.QueryRow(`
		SELECT token, user_id, username, CAST(expires_at AS INTEGER)
		FROM auth_tokens
		WHERE token = ? AND CAST(expires_at AS INTEGER) > CAST(strftime('%s','now') AS INTEGER)`,
		token)

	var authToken models.AuthToken
	var expiresUnix int64
	err := row.Scan(&authToken.Token, &authToken.UserID, &authToken.Username, &expiresUnix)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	authToken.ExpiresIn = expiresUnix - time.Now().Unix()
	return &authToken, nil
}

// DeleteToken 删除 token
func (s *SQLiteStorage) DeleteToken(token string) error {
	_, err := s.db.Exec(`DELETE FROM auth_tokens WHERE token = ?`, token)
	return err
}

// DeleteExpiredTokens 删除过期 tokens
func (s *SQLiteStorage) DeleteExpiredTokens() (int, error) {
	result, err := s.db.Exec(
		`DELETE FROM auth_tokens WHERE CAST(expires_at AS INTEGER) <= CAST(strftime('%s','now') AS INTEGER)`)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}

// ========== API Key 操作 ==========

// SaveAPIKey 保存 API Key
func (s *SQLiteStorage) SaveAPIKey(apiKey *models.APIKey) error {
	query := `
		INSERT OR REPLACE INTO api_keys
		(id, user_id, key_hash, name, last_used_at, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.Exec(query,
		apiKey.ID, apiKey.UserID, apiKey.KeyHash, apiKey.Name,
		apiKey.LastUsed, apiKey.CreatedAt, apiKey.ExpiresAt,
	)
	return err
}

// GetAPIKeyByHash 根据 hash 获取 API Key
func (s *SQLiteStorage) GetAPIKeyByHash(keyHash string) (*models.APIKey, error) {
	row := s.db.QueryRow(`
		SELECT id, user_id, key_hash, name, last_used_at, created_at, expires_at
		FROM api_keys
		WHERE key_hash = ?`,
		keyHash)
	var apiKey models.APIKey
	err := row.Scan(&apiKey.ID, &apiKey.UserID, &apiKey.KeyHash, &apiKey.Name,
		&apiKey.LastUsed, &apiKey.CreatedAt, &apiKey.ExpiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &apiKey, err
}

// DeleteAPIKey 删除 API Key
func (s *SQLiteStorage) DeleteAPIKey(keyHash string) error {
	_, err := s.db.Exec(`DELETE FROM api_keys WHERE key_hash = ?`, keyHash)
	return err
}

// UpdateAPIKeyLastUsed 更新 API Key 最后使用时间
func (s *SQLiteStorage) UpdateAPIKeyLastUsed(keyHash string) error {
	_, err := s.db.Exec(`UPDATE api_keys SET last_used_at = datetime('now') WHERE key_hash = ?`, keyHash)
	return err
}
