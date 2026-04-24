package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lh123aa/cortex/internal/auth"
	"github.com/lh123aa/cortex/internal/models"
)

// AuthMiddleware 认证中间件
type AuthMiddleware struct {
	authService *auth.AuthService
}

// NewAuthMiddleware 创建认证中间件
func NewAuthMiddleware(authService *auth.AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

// RequireAuth 需要认证的请求
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c)

		if token == "" {
			// 检查是否有 API Key
			apiKey := c.GetHeader("X-API-Key")
			if apiKey != "" {
				_, user, err := m.authService.ValidateAPIKey(apiKey)
				if err != nil {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
					return
				}
				c.Set("user", user)
				c.Next()
				return
			}

			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		user, err := m.authService.GetUserByToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		c.Set("user", user)
		c.Next()
	}
}

// RequireAdmin 需要管理员权限
func (m *AuthMiddleware) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := getUserFromContext(c)
		if user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		if user.Role != models.RoleAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}

		c.Next()
	}
}

// OptionalAuth 可选的认证（有就解析，没有就跳过）
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c)

		if token != "" {
			user, err := m.authService.GetUserByToken(token)
			if err == nil && user != nil {
				c.Set("user", user)
			}
		} else {
			// 检查 API Key
			apiKey := c.GetHeader("X-API-Key")
			if apiKey != "" {
				_, user, err := m.authService.ValidateAPIKey(apiKey)
				if err == nil && user != nil {
					c.Set("user", user)
				}
			}
		}

		c.Next()
	}
}

// AdminOrOwner 管理员或资源所有者
// 用于检查用户是否有权访问某个资源
func (m *AuthMiddleware) AdminOrOwner(getOwnerID func(c *gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := getUserFromContext(c)
		if user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		// Admin 可以访问所有资源
		if user.Role == models.RoleAdmin {
			c.Next()
			return
		}

		// 普通用户只能访问自己的资源
		ownerID := getOwnerID(c)
		if ownerID == user.ID {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied"})
	}
}

// UserContext 解析用户信息并存入 context
type UserContext struct {
	UserID   string
	Username string
	Role     string
}

// GetUserContext 从 gin.Context 获取用户上下文
func GetUserContext(c *gin.Context) *UserContext {
	user := getUserFromContext(c)
	if user == nil {
		return nil
	}
	return &UserContext{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
	}
}

// HasRole 检查用户是否有指定角色
func HasRole(c *gin.Context, role string) bool {
	user := getUserFromContext(c)
	if user == nil {
		return false
	}
	return user.Role == role
}

// IsAdmin 检查是否是管理员
func IsAdmin(c *gin.Context) bool {
	return HasRole(c, models.RoleAdmin)
}

// ParseAPIKeyWithPrefix 解析带前缀的 API Key
// 例如: "cxk_live_xxxxx" 或 "cxk_test_xxxxx"
func ParseAPIKeyWithPrefix(key string) (prefix string, actualKey string, ok bool) {
	parts := strings.SplitN(key, "_", 3)
	if len(parts) >= 3 {
		return parts[0] + "_" + parts[1], parts[2], true
	}
	return "", key, false
}