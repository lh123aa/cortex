package storage

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lh123aa/cortex/internal/models"
)

// SearchCacheEntry 缓存条目
type SearchCacheEntry struct {
	QueryHash string    `json:"query_hash"`
	Query     string    `json:"query"`
	Mode      string    `json:"mode"`
	TopK      int       `json:"top_k"`
	Results   string    `json:"results"` // JSON 序列化的结果
	HitCount  int       `json:"hit_count"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// GetCachedSearch 获取缓存的搜索结果（用户隔离）
func (s *SQLiteStorage) GetCachedSearch(query string, userID string, mode string, topK int) ([]*models.SearchResult, bool) {
	queryHash := hashQuery(query, userID, mode, topK)

	var entry SearchCacheEntry
	err := s.db.QueryRow(`
		SELECT query_hash, query, mode, top_k, results, hit_count, created_at, expires_at
		FROM search_cache
		WHERE query_hash = ? AND expires_at > datetime('now')
	`, queryHash).Scan(
		&entry.QueryHash, &entry.Query, &entry.Mode, &entry.TopK,
		&entry.Results, &entry.HitCount, &entry.CreatedAt, &entry.ExpiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		return nil, false
	}

	// 反序列化结果
	var results []*models.SearchResult
	if err := json.Unmarshal([]byte(entry.Results), &results); err != nil {
		return nil, false
	}

	// 更新命中计数
	s.db.Exec(`UPDATE search_cache SET hit_count = hit_count + 1 WHERE query_hash = ?`, queryHash)

	return results, true
}

// SetCachedSearch 设置搜索缓存（用户隔离）
func (s *SQLiteStorage) SetCachedSearch(query string, userID string, mode string, topK int, results []*models.SearchResult, ttl time.Duration) error {
	queryHash := hashQuery(query, userID, mode, topK)

	// 序列化结果
	resultsJSON, err := json.Marshal(results)
	if err != nil {
		return err
	}

	expiresAt := time.Now().Add(ttl)

	_, err = s.db.Exec(`
		INSERT OR REPLACE INTO search_cache
		(query_hash, query, mode, top_k, results, hit_count, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, 0, datetime('now'), ?)
	`, queryHash, query, mode, topK, string(resultsJSON), expiresAt)

	return err
}

// InvalidateSearchCache 使缓存失效（文档更新时调用）
func (s *SQLiteStorage) InvalidateSearchCache() error {
	_, err := s.db.Exec(`DELETE FROM search_cache`)
	return err
}

// InvalidateUserSearchCache 使某个用户的缓存失效
func (s *SQLiteStorage) InvalidateUserSearchCache(userID string) error {
	// 缓存 key 包含 user_id，所以只需要清空所有缓存（简单实现）
	// 未来可以添加 user_id 列来精确清理
	_, err := s.db.Exec(`DELETE FROM search_cache`)
	return err
}

// CleanupExpiredCache 清理过期缓存
func (s *SQLiteStorage) CleanupExpiredCache() (int, error) {
	result, err := s.db.Exec(`DELETE FROM search_cache WHERE expires_at <= datetime('now')`)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}

// GetCacheStats 获取缓存统计
func (s *SQLiteStorage) GetCacheStats() (total int, expired int, avgHits float64, err error) {
	var totalCount, expiredCount, totalHits sql.NullInt64

	err = s.db.QueryRow(`SELECT COUNT(*), SUM(hit_count) FROM search_cache`).Scan(&totalCount, &totalHits)
	if err != nil {
		return 0, 0, 0, err
	}

	err = s.db.QueryRow(`SELECT COUNT(*) FROM search_cache WHERE expires_at <= datetime('now')`).Scan(&expiredCount)
	if err != nil {
		return 0, 0, 0, err
	}

	total = int(totalCount.Int64)
	expired = int(expiredCount.Int64)

	if total > 0 && totalHits.Valid {
		avgHits = float64(totalHits.Int64) / float64(total)
	}

	return total, expired, avgHits, nil
}

// hashQuery 生成查询的 hash（包含 user_id 以实现用户隔离缓存）
// 使用 SHA256 哈希，确保分布均匀
func hashQuery(query, userID, mode string, topK int) string {
	data := fmt.Sprintf("%s|%s|%s|%d", userID, query, mode, topK)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])[:32]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
