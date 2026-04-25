package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// TimeoutMiddleware 超时中间件
// 为每个请求设置超时时间，防止慢查询占用资源
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 为请求创建带超时的 context
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// 替换请求的 context
		c.Request = c.Request.WithContext(ctx)

		// 创建一个 channel 来通知完成
		done := make(chan struct{}, 1)

		go func() {
			c.Next()
			close(done)
		}()

		// 等待 context 超时或请求完成
		select {
		case <-done:
			// 请求正常完成
			return
		case <-ctx.Done():
			// context 超时
			c.AbortWithStatusJSON(http.StatusGatewayTimeout, gin.H{
				"error":   "request timeout",
				"message": "the request took too long to process",
				"timeout": timeout.String(),
			})
		}
	}
}

// DefaultTimeout 默认超时时间
const DefaultTimeout = 30 * time.Second

// SearchTimeout 搜索操作超时时间
const SearchTimeout = 60 * time.Second

// IndexTimeout 索引操作超时时间
const IndexTimeout = 5 * time.Minute

// TimeoutConfig 超时配置
type TimeoutConfig struct {
	// Default 超时中间件的默认超时时间
	Default time.Duration
	// Search 搜索操作的超时时间
	Search time.Duration
	// Index 索引操作的超时时间
	Index time.Duration
}

// NewTimeoutConfig 创建默认的超时配置
func NewTimeoutConfig() *TimeoutConfig {
	return &TimeoutConfig{
		Default: DefaultTimeout,
		Search:  SearchTimeout,
		Index:   IndexTimeout,
	}
}

// ApplyToRouter 将超时配置应用到 Gin 路由
func (tc *TimeoutConfig) ApplyToRouter(r *gin.Engine) {
	r.Use(TimeoutMiddleware(tc.Default))
}

// SearchTimeoutMiddleware 搜索专用的超时中间件
func SearchTimeoutMiddleware() gin.HandlerFunc {
	return TimeoutMiddleware(SearchTimeout)
}

// IndexTimeoutMiddleware 索引专用的超时中间件
func IndexTimeoutMiddleware() gin.HandlerFunc {
	return TimeoutMiddleware(IndexTimeout)
}
