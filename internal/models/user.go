package models

import (
	"strings"
	"time"
)

// UserTier 用户套餐等级
type UserTier string

const (
	TierFree       UserTier = "free"
	TierPro        UserTier = "pro"
	TierEnterprise UserTier = "enterprise"
)

// StorageLimit 根据套餐返回存储上限（字节）
func (t UserTier) StorageLimit() int64 {
	switch t {
	case TierFree:
		return 1 * 1024 * 1024 * 1024 // 1 GB
	case TierPro:
		return 100 * 1024 * 1024 * 1024 // 100 GB（软限制）
	case TierEnterprise:
		return -1 // 不限
	default:
		return 1 * 1024 * 1024 * 1024
	}
}

// IsValid 验证套餐是否有效
func (t UserTier) IsValid() bool {
	switch t {
	case TierFree, TierPro, TierEnterprise:
		return true
	default:
		return false
	}
}

// User 用户模型
type User struct {
	ID                string    `json:"id"`
	Username          string    `json:"username"`
	PasswordHash      string    `json:"-"`                   // 不返回给客户端
	Role              string    `json:"role"`                // admin/user
	Tier              string    `json:"tier"`                // free | pro | enterprise
	StorageUsedBytes  int64     `json:"storage_used_bytes"`  // 当前存储用量
	StorageLimitBytes int64     `json:"storage_limit_bytes"` // 存储上限
	LicenseKey        string    `json:"license_key,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	LastLoginAt       time.Time `json:"last_login_at,omitempty"`
	IsActive          bool      `json:"is_active"`
}

// UserRole 用户角色常量
const (
	RoleAdmin  = "admin"
	RoleEditor = "editor"
	RoleViewer = "viewer"
	// RoleUser 是 v3.x 的历史取值，等价于 editor；
	// 旧库里的 "user" 会被 NormalizeRole 归一化为 editor，不破坏既有部署。
	RoleUser = "user"
)

// NormalizeRole 把任意角色字符串归一化为标准角色。
// 空值、"user" 统一按 editor 处理（保持旧版本可写的行为）；
// 未知值按最小权限（viewer）处理，避免拼写错误导致越权。
func NormalizeRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case RoleAdmin:
		return RoleAdmin
	case RoleViewer:
		return RoleViewer
	case RoleEditor, RoleUser, "":
		return RoleEditor
	default:
		return RoleViewer
	}
}

// IsValidRole 判断角色字符串是否受支持（含历史值 "user"）。
func IsValidRole(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case RoleAdmin, RoleEditor, RoleViewer, RoleUser:
		return true
	default:
		return false
	}
}

// RoleRank 返回角色等级：viewer=1, editor=2, admin=3。
func RoleRank(role string) int {
	switch NormalizeRole(role) {
	case RoleAdmin:
		return 3
	case RoleEditor:
		return 2
	default:
		return 1
	}
}

// CanRead 是否可读。所有已登录角色都可读。
func CanRead(role string) bool { return RoleRank(role) >= 1 }

// CanWrite 是否可写（索引、写记忆、删除条目）。
func CanWrite(role string) bool { return RoleRank(role) >= 2 }

// CanAdmin 是否可做管理操作（用户/密钥管理）。
func CanAdmin(role string) bool { return RoleRank(role) >= 3 }

// APIKey API Key 模型
type APIKey struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	KeyHash   string    `json:"-"`    // 哈希值，存储用，不返回给客户端
	RawKey    string    `json:"-"`    // 原始 Key，仅在创建时一次性使用，不序列化
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
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// CreateAPIKeyRequest 创建 API Key 请求
type CreateAPIKeyRequest struct {
	Name      string `json:"name" binding:"required"`
	ExpiresIn int64  `json:"expires_in"` // 0 = never expires
}
