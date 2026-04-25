package storage

import (
	"time"

	"github.com/lh123aa/cortex/internal/models"
)

// Storage 定义了Cortex核心存储引擎必须实现的所有方法
// 用于解耦具体数据库实现(目前为SQLite)
type Storage interface {
	// ========== 文档操作 ==========

	// SaveDocument 保存文档（自动关联当前用户）
	SaveDocument(doc *models.Document) error

	// GetDocumentByID 获取文档（用户隔离）
	GetDocumentByID(id string, userID string) (*models.Document, error)

	// GetDocumentByPath 根据路径获取文档（用户隔离）
	GetDocumentByPath(path string, userID string) (*models.Document, error)

	// DeleteDocument 删除文档（用户隔离）
	DeleteDocument(id string, userID string) error

	// DeleteDocumentByPath 按路径删除文档（用户隔离）
	DeleteDocumentByPath(path string, userID string) error

	// ListDocuments 列出文档（用户隔离）
	ListDocuments(userID string, limit, offset int) ([]*models.Document, error)

	// ListAllDocuments 列出所有文档（管理员用，不隔离）
	ListAllDocuments(limit, offset int) ([]*models.Document, error)

	// GetDocumentsCount 获取文档数量（用户隔离）
	GetDocumentsCount(userID string) (int, error)

	// ========== 分块操作 ==========

	// SaveChunks 保存分块（自动关联当前用户）
	SaveChunks(chunks []*models.Chunk) error

	// GetChunk 获取分块（用户隔离）
	GetChunk(id string, userID string) (*models.Chunk, error)

	// DeleteChunksByDocument 删除文档的所有分块（用户隔离）
	DeleteChunksByDocument(docID string, userID string) error

	// GetChunksCount 获取分块数量（用户隔离）
	GetChunksCount(userID string) (int, error)

	// ========== 向量操作 ==========

	// GetVectorsCount 获取向量数量（用户隔离）
	GetVectorsCount(userID string) (int, error)

	// ========== 搜索操作 ==========

	// VectorSearch 向量搜索（用户隔离）
	VectorSearch(vector []float32, userID string, topK int) ([]*models.SearchResult, error)

	// FTSSearch 全文搜索（用户隔离）
	FTSSearch(query string, userID string, topK int) ([]*models.SearchResult, error)

	// ========== 缓存操作 ==========

	// GetCachedSearch 获取缓存（用户隔离）
	GetCachedSearch(query string, userID string, mode string, topK int) ([]*models.SearchResult, bool)

	// SetCachedSearch 设置缓存（用户隔离）
	SetCachedSearch(query string, userID string, mode string, topK int, results []*models.SearchResult, ttl time.Duration) error

	// InvalidateSearchCache 使缓存失效
	InvalidateSearchCache() error

	// InvalidateUserSearchCache 使某个用户的缓存失效
	InvalidateUserSearchCache(userID string) error

	// ========== 元数据读写 ==========

	// GetMetadata 获取元数据
	GetMetadata(key string) (string, error)

	// SetMetadata 设置元数据
	SetMetadata(key, value string) error

	// ========== 索引进度操作（断点恢复）============

	// SaveIndexProgress 保存索引进度
	SaveIndexProgress(p *models.IndexProgress) error

	// GetIndexProgress 获取索引进度
	GetIndexProgress(rootPath string) (*models.IndexProgress, error)

	// ListIndexProgress 列出索引进度
	ListIndexProgress(limit, offset int) ([]*models.IndexProgress, error)

	// DeleteIndexProgress 删除索引进度
	DeleteIndexProgress(id int) error

	// CompleteIndexProgress 标记索引完成
	CompleteIndexProgress(rootPath string) error

	// FailIndexProgress 标记索引失败
	FailIndexProgress(rootPath string, errMsg string) error

	// ========== 用户操作（管理员）============

	// DeleteUserData 删除用户的所有数据（用于删除用户时）
	DeleteUserData(userID string) error

	// SaveUser 保存用户
	SaveUser(user *models.User) error

	// GetUserByID 根据 ID 获取用户
	GetUserByID(id string) (*models.User, error)

	// GetUserByUsername 根据用户名获取用户
	GetUserByUsername(username string) (*models.User, error)

	// DeleteUser 删除用户
	DeleteUser(id string) error

	// ========== Token 操作 ==========

	// SaveToken 保存认证 token
	SaveToken(token *models.AuthToken) error

	// GetToken 获取 token
	GetToken(token string) (*models.AuthToken, error)

	// DeleteToken 删除 token
	DeleteToken(token string) error

	// DeleteExpiredTokens 删除过期 tokens
	DeleteExpiredTokens() (int, error)

	// ========== API Key 操作 ==========

	// SaveAPIKey 保存 API Key
	SaveAPIKey(apiKey *models.APIKey) error

	// GetAPIKeyByHash 根据 hash 获取 API Key
	GetAPIKeyByHash(keyHash string) (*models.APIKey, error)

	// DeleteAPIKey 删除 API Key
	DeleteAPIKey(keyHash string) error

	// UpdateAPIKeyLastUsed 更新 API Key 最后使用时间
	UpdateAPIKeyLastUsed(keyHash string) error

	// ========== 系统操作 ==========

	// Close 关闭数据库连接
	Close() error

	// BuildHNSWIndex 构建 HNSW 索引
	BuildHNSWIndex() error

	// SaveVectorIndex 保存向量索引到磁盘
	SaveVectorIndex() error
}