package embedding

import (
	"errors"
	"time"
)

// Embedding 错误类型定义
var (
	// ErrEmbeddingTimeout - Embedding 请求超时
	ErrEmbeddingTimeout = errors.New("embedding timeout")

	// ErrEmbeddingProviderUnavailable - Embedding provider 不可用
	ErrEmbeddingProviderUnavailable = errors.New("embedding provider unavailable")

	// ErrEmbeddingModelNotFound - Embedding 模型未找到
	ErrEmbeddingModelNotFound = errors.New("embedding model not found")

	// ErrEmbeddingRateLimit - Embedding 速率限制
	ErrEmbeddingRateLimit = errors.New("embedding rate limit exceeded")

	// ErrEmbeddingInvalidResponse - Embedding 返回无效响应
	ErrEmbeddingInvalidResponse = errors.New("embedding invalid response")

	// ErrEmbeddingDimensionMismatch - 向量维度不匹配
	ErrEmbeddingDimensionMismatch = errors.New("embedding dimension mismatch")
)

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries    int
	BaseDelay     time.Duration
	MaxDelay      time.Duration
	RetryableErrs []error
}

// DefaultRetryConfig 默认重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries: 3,
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   5 * time.Second,
		RetryableErrs: []error{
			ErrEmbeddingTimeout,
			ErrEmbeddingRateLimit,
			ErrEmbeddingProviderUnavailable,
		},
	}
}

// IsRetryable 判断错误是否可重试
func IsRetryable(err error, cfg *RetryConfig) bool {
	if err == nil {
		return false
	}
	for _, retryable := range cfg.RetryableErrs {
		if errors.Is(err, retryable) {
			return true
		}
	}
	return false
}

// EmbedError Embedding 操作错误详情
type EmbedError struct {
	Provider string
	Op       string // "embed" | "embed_batch"
	Err      error
	Retryable bool
	Timestamp time.Time
}

func (e *EmbedError) Error() string {
	return e.Provider + "." + e.Op + ": " + e.Err.Error()
}

func (e *EmbedError) Unwrap() error {
	return e.Err
}

// NewEmbedError 创建 EmbedError
func NewEmbedError(provider, op string, err error, retryable bool) *EmbedError {
	return &EmbedError{
		Provider:  provider,
		Op:       op,
		Err:      err,
		Retryable: retryable,
		Timestamp: time.Now(),
	}
}

// RetryableError 判断是否为可重试错误
type RetryableError interface {
	error
	IsRetryable() bool
}

// WrapRetryableError 包装错误为可重试
func WrapRetryableError(err error, provider, op string) *EmbedError {
	retryable := false
	for _, e := range []error{ErrEmbeddingTimeout, ErrEmbeddingRateLimit} {
		if errors.Is(err, e) {
			retryable = true
			break
		}
	}
	return &EmbedError{
		Provider:  provider,
		Op:       op,
		Err:      err,
		Retryable: retryable,
		Timestamp: time.Now(),
	}
}