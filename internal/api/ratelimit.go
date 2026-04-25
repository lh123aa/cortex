package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter 基于令牌桶算法的限流器
type RateLimiter struct {
	rate       int           // 每秒产生的令牌数
	burst      int           // 桶的容量
	mu         sync.Mutex
	tokens     float64       // 当前令牌数
	lastUpdate time.Time     // 上次更新时间
}

// NewRateLimiter 创建一个新的限流器
// rate: 每秒产生的令牌数
// burst: 桶的容量（最大突发流量）
func NewRateLimiter(rate, burst int) *RateLimiter {
	rl := &RateLimiter{
		rate:       rate,
		burst:      burst,
		tokens:     float64(burst), // 初始时桶是满的
		lastUpdate: time.Now(),
	}
	return rl
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastUpdate).Seconds()
	rl.lastUpdate = now

	// 添加令牌
	rl.tokens += elapsed * float64(rl.rate)
	if rl.tokens > float64(rl.burst) {
		rl.tokens = float64(rl.burst)
	}

	// 如果有令牌，则允许请求
	if rl.tokens >= 1 {
		rl.tokens--
		return true
	}

	return false
}

// RateLimitMiddleware 限流中间件
// 使用全局的默认限流器
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate limit exceeded",
				"message": "too many requests, please try again later",
			})
			return
		}
		c.Next()
	}
}

// MultiRateLimiter 支持按用户/端点限流的多维度限流器
type MultiRateLimiter struct {
	defaultLimiter *RateLimiter
	limiters       map[string]*RateLimiter
	mu             sync.RWMutex
	rate           int
	burst          int
}

// NewMultiRateLimiter 创建一个多维度限流器
func NewMultiRateLimiter(rate, burst int) *MultiRateLimiter {
	return &MultiRateLimiter{
		defaultLimiter: NewRateLimiter(rate, burst),
		limiters:       make(map[string]*RateLimiter),
		rate:           rate,
		burst:          burst,
	}
}

// GetLimiter 获取指定 key 的限流器
func (mrl *MultiRateLimiter) GetLimiter(key string) *RateLimiter {
	mrl.mu.RLock()
	limiter, exists := mrl.limiters[key]
	mrl.mu.RUnlock()

	if exists {
		return limiter
	}

	mrl.mu.Lock()
	defer mrl.mu.Unlock()

	// 双重检查
	if limiter, exists = mrl.limiters[key]; exists {
		return limiter
	}

	// 创建新的限流器
	limiter = NewRateLimiter(mrl.rate, mrl.burst)
	mrl.limiters[key] = limiter
	return limiter
}

// IPRateLimitMiddleware IP 限流中间件
func IPRateLimitMiddleware(mrl *MultiRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := mrl.GetLimiter(ip)

		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate limit exceeded",
				"message": "too many requests from your IP, please try again later",
			})
			return
		}
		c.Next()
	}
}

// UserRateLimitMiddleware 用户限流中间件（需要认证）
func UserRateLimitMiddleware(mrl *MultiRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 优先使用用户 ID，其次使用 IP
		var key string
		if userID, exists := c.Get("userID"); exists {
			key = userID.(string)
		} else {
			key = c.ClientIP()
		}

		limiter := mrl.GetLimiter(key)

		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate limit exceeded",
				"message": "rate limit exceeded for this user, please try again later",
			})
			return
		}
		c.Next()
	}
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	// Enabled 是否启用限流
	Enabled bool
	// Rate 每秒产生的令牌数
	Rate int
	// Burst 桶的容量
	Burst int
	// PerIP 是否按 IP 限流
	PerIP bool
	// PerUser 是否按用户限流（需要认证）
	PerUser bool
}

// NewRateLimitConfig 创建默认的限流配置
func NewRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Enabled: true,
		Rate:   100,  // 每秒 100 个请求
		Burst:  200,  // 最多突发 200 个请求
		PerIP:  true,  // 默认按 IP 限流
		PerUser: false,
	}
}

// DefaultRateLimiter 默认的全局限流器
var defaultMultiLimiter = NewMultiRateLimiter(100, 200)

// DefaultRateLimitMiddleware 默认的限流中间件
func DefaultRateLimitMiddleware() gin.HandlerFunc {
	return IPRateLimitMiddleware(defaultMultiLimiter)
}
