package auth

import (
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lh123aa/cortex/internal/models"
	"github.com/lh123aa/cortex/internal/storage"
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
	ErrAPIKeyExpired      = errors.New("api key expired")
	ErrInsufficientPerms  = errors.New("insufficient permissions")
)

// AuthService 认证服务
type AuthService struct {
	storage   storage.Storage
	tokenExp  time.Duration
	tokenMu   sync.RWMutex
	apiKeyMu  sync.RWMutex
}

// NewAuthService 创建认证服务（使用指定的存储）
func NewAuthService(s storage.Storage, tokenExpiry time.Duration) *AuthService {
	return &AuthService{
		storage:  s,
		tokenExp: tokenExpiry,
	}
}

// NewAuthServiceWithDefaults 创建认证服务（默认配置，使用内存存储）
func NewAuthServiceWithDefaults() *AuthService {
	// 注意：这个方法创建的是旧版本，保留向后兼容
	// 新代码应该使用 NewAuthService 并传入实际的 storage
	return &AuthService{
		tokenExp: 24 * time.Hour,
	}
}

// NewAuthServiceWithStorage 创建认证服务（使用指定存储）
func NewAuthServiceWithStorage(s storage.Storage) *AuthService {
	return NewAuthService(s, 24*time.Hour)
}

// Register 创建新用户
func (s *AuthService) Register(req *models.RegisterRequest) (*models.User, error) {
	// 检查用户是否已存在
	existing, err := s.storage.GetUserByUsername(req.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to check user: %w", err)
	}
	if existing != nil {
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

	if err := s.storage.SaveUser(user); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}
	return user, nil
}

// Login 用户登录
func (s *AuthService) Login(req *models.LoginRequest) (*models.AuthToken, error) {
	user, err := s.storage.GetUserByUsername(req.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
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

	if err := s.storage.SaveToken(authToken); err != nil {
		return nil, fmt.Errorf("failed to save token: %w", err)
	}

	// 更新最后登录时间
	user.LastLoginAt = time.Now()
	s.storage.SaveUser(user)

	return authToken, nil
}

// Logout 用户登出
func (s *AuthService) Logout(token string) error {
	return s.storage.DeleteToken(token)
}

// ValidateToken 验证 token
func (s *AuthService) ValidateToken(token string) (*models.AuthToken, error) {
	authToken, err := s.storage.GetToken(token)
	if err != nil {
		return nil, err
	}
	if authToken == nil {
		return nil, ErrInvalidToken
	}
	return authToken, nil
}

// GetUserByToken 根据 token 获取用户
func (s *AuthService) GetUserByToken(token string) (*models.User, error) {
	authToken, err := s.storage.GetToken(token)
	if err != nil {
		return nil, err
	}
	if authToken == nil {
		return nil, ErrInvalidToken
	}

	user, err := s.storage.GetUserByID(authToken.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// GetUserByUsername 根据用户名获取用户
func (s *AuthService) GetUserByUsername(username string) (*models.User, error) {
	return s.storage.GetUserByUsername(username)
}

// CreateAPIKey 创建 API Key（用户级别）
func (s *AuthService) CreateAPIKey(userID string, req *models.CreateAPIKeyRequest) (*models.APIKey, error) {
	// 生成 API Key
	keyBytes := make([]byte, 32)
	if _, err := crand.Read(keyBytes); err != nil {
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

	if err := s.storage.SaveAPIKey(apiKey); err != nil {
		return nil, fmt.Errorf("failed to save api key: %w", err)
	}

	// 返回完整 key（只返回一次）
	apiKey.KeyHash = key // 临时存储完整 key 用于返回
	return apiKey, nil
}

// ValidateAPIKey 验证 API Key
func (s *AuthService) ValidateAPIKey(key string) (*models.APIKey, *models.User, error) {
	keyHash := hashAPIKey(key)
	apiKey, err := s.storage.GetAPIKeyByHash(keyHash)
	if err != nil {
		return nil, nil, err
	}
	if apiKey == nil {
		return nil, nil, ErrAPIKeyNotFound
	}

	// 检查过期
	if !apiKey.ExpiresAt.IsZero() && time.Now().After(apiKey.ExpiresAt) {
		return nil, nil, ErrAPIKeyExpired
	}

	// 更新最后使用时间
	s.storage.UpdateAPIKeyLastUsed(keyHash)

	// 获取用户
	user, err := s.storage.GetUserByID(apiKey.UserID)
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return nil, nil, ErrUserNotFound
	}

	return apiKey, user, nil
}

// DeleteAPIKey 删除 API Key
func (s *AuthService) DeleteAPIKey(keyHash string) error {
	return s.storage.DeleteAPIKey(keyHash)
}

// ListAPIKeys 列出用户的 API Keys
func (s *AuthService) ListAPIKeys(userID string) ([]*models.APIKey, error) {
	// 获取该用户的所有 API keys
	return s.storage.ListAPIKeysByUser(userID)
}

// ChangePassword 修改密码
func (s *AuthService) ChangePassword(userID string, oldPwd, newPwd string) error {
	user, err := s.storage.GetUserByID(userID)
	if err != nil {
		return err
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

	return s.storage.SaveUser(user)
}

// ListUsers 列出所有用户（管理员）
func (s *AuthService) ListUsers() ([]*models.User, error) {
	return s.storage.ListUsers(1000, 0) // 限制最多返回1000个用户
}

// DeactivateUser 禁用用户
func (s *AuthService) DeactivateUser(userID string) error {
	user, err := s.storage.GetUserByID(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}
	user.IsActive = false
	user.UpdatedAt = time.Now()
	return s.storage.SaveUser(user)
}

// CleanupExpiredTokens 清理过期 tokens
func (s *AuthService) CleanupExpiredTokens() error {
	_, err := s.storage.DeleteExpiredTokens()
	return err
}

// ==================== 辅助函数 ====================

func generateID() string {
	b := make([]byte, 16)
	crand.Read(b)
	return hex.EncodeToString(b)
}

func generateToken() string {
	b := make([]byte, 32)
	crand.Read(b)
	return hex.EncodeToString(b)
}

func hashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}