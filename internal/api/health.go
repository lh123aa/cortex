package api

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lh123aa/cortex/internal/embedding"
	"github.com/lh123aa/cortex/internal/storage"
)

// HealthStatus 健康状态
type HealthStatus struct {
	Status     string            `json:"status"`
	Timestamp  time.Time         `json:"timestamp"`
	Checks     map[string]Check  `json:"checks,omitempty"`
}

// Check 单个检查项
type Check struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"`
}

// HealthChecker 健康检查器
type HealthChecker struct {
	storage   storage.Storage
	embedding embedding.EmbeddingProvider
	mu        sync.RWMutex
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(st storage.Storage, em embedding.EmbeddingProvider) *HealthChecker {
	return &HealthChecker{
		storage:   st,
		embedding: em,
	}
}

// CheckStorage 检查存储是否可用
func (hc *HealthChecker) CheckStorage() Check {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 尝试一个简单的查询
	_, err := hc.storage.GetDocumentsCount()
	latency := time.Since(start)

	if err != nil {
		return Check{
			Status:  "unhealthy",
			Message: "storage unavailable: " + err.Error(),
			Latency: latency.String(),
		}
	}

	return Check{
		Status:  "healthy",
		Latency: latency.String(),
	}
}

// CheckEmbedding 检查 Embedding 服务是否可用
func (hc *HealthChecker) CheckEmbedding() Check {
	if hc.embedding == nil {
		return Check{
			Status:  "disabled",
			Message: "embedding not configured",
		}
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Health check 通常不应该真正调用 Embed，只能检查连通性
	err := hc.embedding.Health()
	latency := time.Since(start)

	if err != nil {
		return Check{
			Status:  "unhealthy",
			Message: "embedding unavailable: " + err.Error(),
			Latency: latency.String(),
		}
	}

	return Check{
		Status:  "healthy",
		Latency: latency.String(),
	}
}

// FullCheck 执行完整健康检查
func (hc *HealthChecker) FullCheck() HealthStatus {
	status := HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Checks:    make(map[string]Check),
	}

	storageCheck := hc.CheckStorage()
	status.Checks["storage"] = storageCheck
	if storageCheck.Status != "healthy" {
		status.Status = "degraded"
	}

	embeddingCheck := hc.CheckEmbedding()
	status.Checks["embedding"] = embeddingCheck
	// embedding 检查失败不一定是 unhealthy，因为可能没有配置

	return status
}

// ReadyCheck 就绪检查 - 检查所有依赖是否就绪
func (hc *HealthChecker) ReadyCheck() bool {
	// 存储必须可用
	storageCheck := hc.CheckStorage()
	if storageCheck.Status != "healthy" {
		return false
	}

	// 如果配置了 embedding，也必须可用
	if hc.embedding != nil {
		embeddingCheck := hc.CheckEmbedding()
		if embeddingCheck.Status == "unhealthy" {
			return false
		}
	}

	return true
}

// LiveCheck 存活检查 - 简单返回 OK
func (hc *HealthChecker) LiveCheck() bool {
	return true
}