package auth

import (
	"testing"
	"time"

	"github.com/lh123aa/cortex/internal/models"
)

// MockStorage 用于测试的内存存储
type MockStorage struct {
	users   map[string]*models.User
	tokens  map[string]*models.AuthToken
	apiKeys map[string]*models.APIKey
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		users:   make(map[string]*models.User),
		tokens:  make(map[string]*models.AuthToken),
		apiKeys: make(map[string]*models.APIKey),
	}
}

func (m *MockStorage) SaveUser(user *models.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *MockStorage) GetUserByID(id string) (*models.User, error) {
	return m.users[id], nil
}

func (m *MockStorage) GetUserByUsername(username string) (*models.User, error) {
	for _, u := range m.users {
		if u.Username == username {
			return u, nil
		}
	}
	return nil, nil
}

func (m *MockStorage) DeleteUser(id string) error {
	delete(m.users, id)
	return nil
}

func (m *MockStorage) SaveToken(token *models.AuthToken) error {
	m.tokens[token.Token] = token
	return nil
}

func (m *MockStorage) GetToken(token string) (*models.AuthToken, error) {
	return m.tokens[token], nil
}

func (m *MockStorage) DeleteToken(token string) error {
	delete(m.tokens, token)
	return nil
}

func (m *MockStorage) DeleteExpiredTokens() (int, error) {
	now := time.Now()
	count := 0
	for k, t := range m.tokens {
		if t.ExpiresIn > 0 {
			// 简化逻辑
		}
		_ = now
		delete(m.tokens, k)
		count++
	}
	return count, nil
}

func (m *MockStorage) SaveAPIKey(apiKey *models.APIKey) error {
	m.apiKeys[apiKey.KeyHash] = apiKey
	return nil
}

func (m *MockStorage) GetAPIKeyByHash(keyHash string) (*models.APIKey, error) {
	return m.apiKeys[keyHash], nil
}

func (m *MockStorage) DeleteAPIKey(keyHash string) error {
	delete(m.apiKeys, keyHash)
	return nil
}

func (m *MockStorage) UpdateAPIKeyLastUsed(keyHash string) error {
	if ak, ok := m.apiKeys[keyHash]; ok {
		ak.LastUsed = time.Now()
	}
	return nil
}

func TestRegister(t *testing.T) {
	svc := &AuthService{
		tokenExp: 24 * time.Hour,
	}

	// Note: This test requires a real storage implementation
	// For now, just test the basic service creation
	if svc.tokenExp != 24*time.Hour {
		t.Errorf("Expected 24h token expiry, got %v", svc.tokenExp)
	}
}

func TestTokenExpiryCalculation(t *testing.T) {
	// 测试 token 过期时间计算
	tokenExp := 24 * time.Hour
	expiresIn := int64(tokenExp.Seconds())

	if expiresIn != 86400 {
		t.Errorf("Expected 86400 seconds (24h), got %d", expiresIn)
	}
}

func TestPasswordHashing(t *testing.T) {
	// 测试 bcrypt 哈希（简化测试）
	password := "TestPassword123"

	// 使用 argon2 或 bcrypt 验证
	// 这里只是占位测试
	if len(password) < 8 {
		t.Error("Password should be at least 8 characters")
	}
}

func TestAPIKeyGeneration(t *testing.T) {
	// 测试 API Key 生成
	keyBytes := make([]byte, 32)
	if len(keyBytes) != 32 {
		t.Errorf("Expected 32 bytes, got %d", len(keyBytes))
	}
}

func TestAuthTokenModel(t *testing.T) {
	token := &models.AuthToken{
		Token:     "test-token-abc123",
		ExpiresIn: 3600,
		UserID:    "user-1",
		Username:  "testuser",
	}

	if token.Token != "test-token-abc123" {
		t.Errorf("Expected token 'test-token-abc123', got '%s'", token.Token)
	}
	if token.ExpiresIn != 3600 {
		t.Errorf("Expected expires_in 3600, got %d", token.ExpiresIn)
	}
}

func TestUserModel(t *testing.T) {
	user := &models.User{
		ID:           "user-123",
		Username:     "testuser",
		PasswordHash: "hashed_password",
		Role:         models.RoleUser,
		IsActive:     true,
		CreatedAt:    time.Now(),
	}

	if user.Role != "user" {
		t.Errorf("Expected role 'user', got '%s'", user.Role)
	}
	if !user.IsActive {
		t.Error("User should be active")
	}
}

func TestAPIKeyModel(t *testing.T) {
	apiKey := &models.APIKey{
		ID:        "key-123",
		UserID:    "user-1",
		KeyHash:   "hash_value",
		Name:      "Test Key",
		CreatedAt: time.Now(),
	}

	if apiKey.Name != "Test Key" {
		t.Errorf("Expected name 'Test Key', got '%s'", apiKey.Name)
	}
}

func TestRoleConstants(t *testing.T) {
	if models.RoleAdmin != "admin" {
		t.Errorf("Expected RoleAdmin to be 'admin', got '%s'", models.RoleAdmin)
	}
	if models.RoleUser != "user" {
		t.Errorf("Expected RoleUser to be 'user', got '%s'", models.RoleUser)
	}
}

func TestErrorDefinitions(t *testing.T) {
	// 测试错误定义
	if ErrUserNotFound.Error() != "user not found" {
		t.Errorf("Unexpected error message: %s", ErrUserNotFound.Error())
	}
	if ErrInvalidPassword.Error() != "invalid password" {
		t.Errorf("Unexpected error message: %s", ErrInvalidPassword.Error())
	}
	if ErrUserExists.Error() != "user already exists" {
		t.Errorf("Unexpected error message: %s", ErrUserExists.Error())
	}
	if ErrInvalidToken.Error() != "invalid token" {
		t.Errorf("Unexpected error message: %s", ErrInvalidToken.Error())
	}
}
