package models

import "time"

// User 用户模型
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // 不返回给客户端
	Role         string    `json:"role"` // admin/user
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	LastLoginAt  time.Time `json:"last_login_at,omitempty"`
	IsActive     bool      `json:"is_active"`
}

// UserRole 用户角色常量
const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

// APIKey API Key 模型
type APIKey struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	KeyHash   string    `json:"-"` // 不返回给客户端
	Name      string    `json:"name"` // key 名称，如 "我的电脑"
	LastUsed  time.Time `json:"last_used_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// AuthToken 认证 token
type AuthToken struct {
	Token     string `json:"token"`
	ExpiresIn int64  `json:"expires_in"` // seconds
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" binding:"required,min=6"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// CreateAPIKeyRequest 创建 API Key 请求
type CreateAPIKeyRequest struct {
	Name      string `json:"name" binding:"required"`
	ExpiresIn int64  `json:"expires_in"` // 0 = never expires
}