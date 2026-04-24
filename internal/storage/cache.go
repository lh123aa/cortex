package storage

import (
	"database/sql"
	"encoding/json"
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

// GetCachedSearch 获取缓存的搜索结果
func (s *SQLiteStorage) GetCachedSearch(query string, mode string, topK int) ([]*models.SearchResult, bool) {
	queryHash := hashQuery(query, mode, topK)

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

// SetCachedSearch 设置搜索缓存
func (s *SQLiteStorage) SetCachedSearch(query string, mode string, topK int, results []*models.SearchResult, ttl time.Duration) error {
	queryHash := hashQuery(query, mode, topK)

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

// hashQuery 生成查询的 hash
func hashQuery(query, mode string, topK int) string {
	// 简单的 hash 实现
	data := query + mode + string(rune(topK))
	hash := 0
	for i, c := range data {
		hash = hash*31 + int(c) + i
	}
	if hash < 0 {
		hash = -hash
	}
	return string(rune(hash%1000000)) + "_" + query[:min(50, len(query))]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
