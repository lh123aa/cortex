package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lh123aa/cortex/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// 错误定义
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
	ErrAPIKeyNotFound     = errors.New("api key not found")
	ErrAPIKeyInvalid      = errors.New("invalid api key")
	ErrInsufficientPerms  = errors.New("insufficient permissions")
)

// AuthService 认证服务
type AuthService struct {
	users      map[string]*models.User    // username -> User
	apiKeys    map[string]*models.APIKey // keyHash -> APIKey
	tokens     map[string]*models.AuthToken // token -> AuthToken
	tokenExp   time.Duration
	mu         sync.RWMutex
}

// NewAuthService 创建认证服务
func NewAuthService(tokenExpiry time.Duration) *AuthService {
	return &AuthService{
		users:    make(map[string]*models.User),
		apiKeys:  make(map[string]*models.APIKey),
		tokens:   make(map[string]*models.AuthToken),
		tokenExp: tokenExpiry,
	}
}

// NewAuthServiceWithDefaults 创建认证服务（默认配置）
func NewAuthServiceWithDefaults() *AuthService {
	return NewAuthService(24 * time.Hour)
}

// Register 创建新用户
func (s *AuthService) Register(req *models.RegisterRequest) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查用户是否存在
	if _, exists := s.users[req.Username]; exists {
		return nil, ErrUserExists
	}

	// 密码哈希
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		ID:           generateID(),
		Username:     req.Username,
		PasswordHash: string(hash),
		Role:         models.RoleUser,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		IsActive:    true,
	}

	s.users[req.Username] = user
	return user, nil
}

// Login 用户登录
func (s *AuthService) Login(req *models.LoginRequest) (*models.AuthToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[req.Username]
	if !exists {
		return nil, ErrUserNotFound
	}

	if !user.IsActive {
		return nil, ErrUserNotFound
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidPassword
	}

	// 生成 token
	token := generateToken()
	authToken := &models.AuthToken{
		Token:     token,
		ExpiresIn: int64(s.tokenExp.Seconds()),
		UserID:    user.ID,
		Username:  user.Username,
	}

	s.tokens[token] = authToken

	// 更新最后登录时间
	user.LastLoginAt = time.Now()

	return authToken, nil
}

// Logout 用户登出
func (s *AuthService) Logout(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.tokens, token)
	return nil
}

// ValidateToken 验证 token
func (s *AuthService) ValidateToken(token string) (*models.AuthToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	authToken, exists := s.tokens[token]
	if !exists {
		return nil, ErrInvalidToken
	}

	return authToken, nil
}

// GetUserByToken 根据 token 获取用户
func (s *AuthService) GetUserByToken(token string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	authToken, exists := s.tokens[token]
	if !exists {
		return nil, ErrInvalidToken
	}

	user, exists := s.users[authToken.Username]
	if !exists {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// GetUserByUsername 根据用户名获取用户
func (s *AuthService) GetUserByUsername(username string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[username]
	if !exists {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// CreateAPIKey 创建 API Key（用户级别）
func (s *AuthService) CreateAPIKey(userID string, req *models.CreateAPIKeyRequest) (*models.APIKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 生成 API Key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	key := hex.EncodeToString(keyBytes)
	keyHash := hashAPIKey(key)

	apiKey := &models.APIKey{
		ID:        generateID(),
		UserID:    userID,
		KeyHash:   keyHash,
		Name:      req.Name,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}

	if req.ExpiresIn > 0 {
		apiKey.ExpiresAt = time.Now().Add(time.Duration(req.ExpiresIn) * time.Second)
	}

	s.apiKeys[keyHash] = apiKey

	// 返回完整 key（只返回一次）
	apiKey.KeyHash = key // 临时存储完整 key 用于返回
	return apiKey, nil
}

// ValidateAPIKey 验证 API Key
func (s *AuthService) ValidateAPIKey(key string) (*models.APIKey, *models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	keyHash := hashAPIKey(key)
	apiKey, exists := s.apiKeys[keyHash]
	if !exists {
		return nil, nil, ErrAPIKeyNotFound
	}

	// 检查过期
	if !apiKey.ExpiresAt.IsZero() && time.Now().After(apiKey.ExpiresAt) {
		return nil, nil, ErrAPIKeyExpired
	}

	// 更新最后使用时间
	apiKey.LastUsed = time.Now()

	// 获取用户
	var user *models.User
	for _, u := range s.users {
		if u.ID == apiKey.UserID {
			user = u
			break
		}
	}

	if user == nil {
		return nil, nil, ErrUserNotFound
	}

	return apiKey, user, nil
}

// DeleteAPIKey 删除 API Key
func (s *AuthService) DeleteAPIKey(keyHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.apiKeys, keyHash)
	return nil
}

// ListAPIKeys 列出用户的 API Keys
func (s *AuthService) ListAPIKeys(userID string) []*models.APIKey {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var keys []*models.APIKey
	for _, k := range s.apiKeys {
		if k.UserID == userID {
			keys = append(keys, &models.APIKey{
				ID:        k.ID,
				UserID:    k.UserID,
				Name:      k.Name,
				LastUsed:  k.LastUsed,
				CreatedAt: k.CreatedAt,
				ExpiresAt: k.ExpiresAt,
			})
		}
	}
	return keys
}

// ChangePassword 修改密码
func (s *AuthService) ChangePassword(userID string, oldPwd, newPwd string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var user *models.User
	for _, u := range s.users {
		if u.ID == userID {
			user = u
			break
		}
	}

	if user == nil {
		return ErrUserNotFound
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPwd)); err != nil {
		return ErrInvalidPassword
	}

	// 设置新密码
	hash, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.PasswordHash = string(hash)
	user.UpdatedAt = time.Now()

	return nil
}

// ListUsers 列出所有用户（管理员）
func (s *AuthService) ListUsers() []*models.User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var users []*models.User
	for _, u := range s.users {
		users = append(users, &models.User{
			ID:        u.ID,
			Username:  u.Username,
			Role:      u.Role,
			CreatedAt: u.CreatedAt,
			IsActive:  u.IsActive,
		})
	}
	return users
}

// DeactivateUser 禁用用户
func (s *AuthService) DeactivateUser(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, u := range s.users {
		if u.ID == userID {
			u.IsActive = false
			u.UpdatedAt = time.Now()
			return nil
		}
	}
	return ErrUserNotFound
}

// CleanupExpiredTokens 清理过期 tokens
func (s *AuthService) CleanupExpiredTokens() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for token, authToken := range s.tokens {
		// token 过期时间由外部控制，这里只做清理
		_ = authToken
		delete(s.tokens, token)
	}
}

// ==================== 辅助函数 ====================

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func hashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}