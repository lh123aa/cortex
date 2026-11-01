package auth

import (
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/lh123aa/cortex/internal/models"
	"github.com/lh123aa/cortex/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

// 错误定义
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserDeactivated   = errors.New("account deactivated")
	ErrInvalidPassword   = errors.New("invalid password")
	ErrUserExists        = errors.New("user already exists")
	ErrInvalidToken      = errors.New("invalid token")
	ErrTokenExpired      = errors.New("token expired")
	ErrAPIKeyNotFound    = errors.New("api key not found")
	ErrAPIKeyInvalid     = errors.New("invalid api key")
	ErrAPIKeyExpired     = errors.New("api key expired")
	ErrInsufficientPerms = errors.New("insufficient permissions")
	// ErrInvalidRole 角色取值非法
	ErrInvalidRole = errors.New("invalid role")
	// ErrLastAdmin 不允许降级最后一个管理员（会把自己锁在门外）
	ErrLastAdmin = errors.New("cannot demote the last admin")
)

// AuthService 认证服务
type AuthService struct {
	storage  storage.Storage // 兼容层：保持向后兼容
	tokenExp time.Duration
	tokenMu  sync.RWMutex
	apiKeyMu sync.RWMutex
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

// validatePassword 校验密码强度
func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	if len(password) > 128 {
		return fmt.Errorf("password must be at most 128 characters")
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		case unicode.IsPunct(ch) || unicode.IsSymbol(ch):
			hasSpecial = true
		}
	}

	var missing []string
	if !hasUpper {
		missing = append(missing, "uppercase letter")
	}
	if !hasLower {
		missing = append(missing, "lowercase letter")
	}
	if !hasDigit {
		missing = append(missing, "digit")
	}
	if !hasSpecial {
		missing = append(missing, "special character")
	}
	if len(missing) > 0 {
		return fmt.Errorf("password must contain at least one %s", strings.Join(missing, ", "))
	}

	return nil
}

// Register 创建新用户
func (s *AuthService) Register(req *models.RegisterRequest) (*models.User, error) {
	// 校验密码强度
	if err := validatePassword(req.Password); err != nil {
		return nil, err
	}

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
		ID:                generateID(),
		Username:          req.Username,
		PasswordHash:      string(hash),
		Role:              models.RoleEditor,
		Tier:              string(models.TierFree),
		StorageLimitBytes: models.TierFree.StorageLimit(),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		IsActive:          true,
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
		return nil, ErrUserDeactivated
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

	// 返回完整 key（只返回一次），使用独立的 RawKey 字段
	apiKey.RawKey = key
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

	// 校验新密码强度
	if err := validatePassword(newPwd); err != nil {
		return err
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

// ActivateUser 重新启用用户
func (s *AuthService) ActivateUser(userID string) error {
	user, err := s.storage.GetUserByID(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}
	user.IsActive = true
	user.UpdatedAt = time.Now()
	return s.storage.SaveUser(user)
}

// SetUserRole 修改用户角色（admin / editor / viewer）。
// role 会先经过 NormalizeRole，因此传历史值 "user" 等价于 editor。
func (s *AuthService) SetUserRole(username, role string) (*models.User, error) {
	if !models.IsValidRole(role) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidRole, role)
	}
	user, err := s.storage.GetUserByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	normalized := models.NormalizeRole(role)
	// 安全护栏：不允许把最后一个管理员降级，否则会把自己锁在门外。
	if models.NormalizeRole(user.Role) == models.RoleAdmin && normalized != models.RoleAdmin {
		if count, err := s.adminCount(); err == nil && count <= 1 {
			return nil, ErrLastAdmin
		}
	}

	user.Role = normalized
	user.UpdatedAt = time.Now()
	if err := s.storage.SaveUser(user); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}
	return user, nil
}

// CreateUserWithRole 创建指定角色的用户（管理员用）。
func (s *AuthService) CreateUserWithRole(username, password, role string) (*models.User, error) {
	if !models.IsValidRole(role) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidRole, role)
	}
	user, err := s.Register(&models.RegisterRequest{Username: username, Password: password})
	if err != nil {
		return nil, err
	}
	if models.NormalizeRole(role) != models.RoleEditor {
		return s.SetUserRole(username, role)
	}
	return user, nil
}

// EnsureBootstrapAdmin 首次启用认证时，把第一个用户提升为 admin。
//
// 背景：v3.x 的注册接口只会创建普通用户，启用 auth 后没有任何人能管理用户/密钥。
// 这里在服务启动时做一次幂等修复：如果系统里没有任何 admin 且已有用户，
// 就把最早的用户提升为 admin。
func (s *AuthService) EnsureBootstrapAdmin() (*models.User, error) {
	users, err := s.storage.ListUsers(10, 0)
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		if models.NormalizeRole(u.Role) == models.RoleAdmin {
			return nil, nil // 已有管理员
		}
	}
	if len(users) == 0 {
		return nil, nil // 还没有用户，等第一个注册的人（其角色为 editor，可再手动提升）
	}
	u := users[0]
	u.Role = models.RoleAdmin
	u.UpdatedAt = time.Now()
	if err := s.storage.SaveUser(u); err != nil {
		return nil, err
	}
	return u, nil
}

// adminCount 统计当前活跃的 admin 数量。
func (s *AuthService) adminCount() (int, error) {
	users, err := s.storage.ListUsers(1000, 0)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, u := range users {
		if models.NormalizeRole(u.Role) == models.RoleAdmin && u.IsActive {
			n++
		}
	}
	return n, nil
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
