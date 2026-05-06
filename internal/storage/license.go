package storage

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/lh123aa/cortex/internal/models"
)

// SaveLicense 保存 License
func (s *SQLiteStorage) SaveLicense(lic *models.License) error {
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO licenses
		(id, key, tier, user_id, max_users, expires_at, created_at, active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		lic.ID, lic.Key, lic.Tier, lic.UserID, lic.MaxUsers, lic.ExpiresAt, lic.CreatedAt, lic.Active)
	return err
}

// GetLicenseByKey 根据 Key 获取 License
func (s *SQLiteStorage) GetLicenseByKey(key string) (*models.License, error) {
	row := s.db.QueryRow(`
		SELECT id, key, tier, COALESCE(user_id,''), max_users, expires_at, created_at, active
		FROM licenses WHERE key = ?`, key)
	var lic models.License
	err := row.Scan(&lic.ID, &lic.Key, &lic.Tier, &lic.UserID, &lic.MaxUsers, &lic.ExpiresAt, &lic.CreatedAt, &lic.Active)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &lic, err
}

// ListLicensesByUser 列出用户的 License
func (s *SQLiteStorage) ListLicensesByUser(userID string) ([]*models.License, error) {
	rows, err := s.db.Query(`
		SELECT id, key, tier, COALESCE(user_id,''), max_users, expires_at, created_at, active
		FROM licenses WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.License
	for rows.Next() {
		var lic models.License
		if err := rows.Scan(&lic.ID, &lic.Key, &lic.Tier, &lic.UserID, &lic.MaxUsers, &lic.ExpiresAt, &lic.CreatedAt, &lic.Active); err != nil {
			return nil, err
		}
		list = append(list, &lic)
	}
	return list, nil
}

// DeactivateLicense 吊销 License
func (s *SQLiteStorage) DeactivateLicense(id string) error {
	_, err := s.db.Exec(`UPDATE licenses SET active = 0 WHERE id = ?`, id)
	return err
}

// GenerateLicenseKey 生成唯一的 License Key
func GenerateLicenseKey() string {
	b := make([]byte, 16)
	rand.Read(b)
	hash := sha256.Sum256(b)
	key := hex.EncodeToString(hash[:16])
	// 按 4-4-4-4 格式展示
	return fmt.Sprintf("%s-%s-%s-%s", key[0:4], key[4:8], key[8:12], key[12:16])
}

// CreateLicense 创建许可证（内部调用）
func (s *SQLiteStorage) CreateLicense(tier string, maxUsers int, duration time.Duration) (*models.License, error) {
	lic := &models.License{
		ID:        hex.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano()))),
		Key:       GenerateLicenseKey(),
		Tier:      tier,
		MaxUsers:  maxUsers,
		ExpiresAt: time.Now().Add(duration),
		CreatedAt: time.Now(),
		Active:    true,
	}
	if err := s.SaveLicense(lic); err != nil {
		return nil, err
	}
	return lic, nil
}
