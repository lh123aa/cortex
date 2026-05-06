package api

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

// APIKeyAuth API Key 认证中间件
// 使用哈希存储密钥，防止内存泄漏后密钥被直接利用
type APIKeyAuth struct {
	headerName     string
	queryName      string
	validKeyHashes map[string]bool // 存储哈希而非原始密钥
	mu             sync.RWMutex
}

func NewAPIKeyAuth(headerName, queryName string) *APIKeyAuth {
	return &APIKeyAuth{
		headerName:     headerName,
		queryName:      queryName,
		validKeyHashes: make(map[string]bool),
	}
}

// hashKey 对密钥进行 SHA256 哈希
func (a *APIKeyAuth) hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// AddKey 添加一个有效的 API key（内部存储哈希）
func (a *APIKeyAuth) AddKey(key string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.validKeyHashes[a.hashKey(key)] = true
}

// RemoveKey 移除一个 API key
func (a *APIKeyAuth) RemoveKey(key string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.validKeyHashes, a.hashKey(key))
}

// ClearKeys 清除所有 API keys
func (a *APIKeyAuth) ClearKeys() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.validKeyHashes = make(map[string]bool)
}

// Middleware returns a Gin middleware that validates API keys
func (a *APIKeyAuth) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := a.getKeyFromRequest(c)
		if key == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing API key",
			})
			return
		}

		if !a.isValidKey(key) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid API key",
			})
			return
		}

		c.Next()
	}
}

// OptionalMiddleware returns a middleware that allows requests without API key
// but validates if present
func (a *APIKeyAuth) OptionalMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := a.getKeyFromRequest(c)
		if key == "" {
			// No key provided, allow but mark as unauthenticated
			c.Set("auth_required", false)
			c.Next()
			return
		}

		if !a.isValidKey(key) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid API key",
			})
			return
		}

		c.Set("auth_required", true)
		c.Next()
	}
}

func (a *APIKeyAuth) getKeyFromRequest(c *gin.Context) string {
	// 仅从 Header 获取 API Key，不接受 URL 查询参数
	// URL 参数方式会导致 key 出现在访问日志、浏览器历史和 Referer 头中
	return c.GetHeader(a.headerName)
}

func (a *APIKeyAuth) isValidKey(key string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.validKeyHashes[a.hashKey(key)]
}


