package storage

import (
	"time"

	"github.com/lh123aa/cortex/internal/models"
)

// ============================================================
// 拆分后的 Small Interfaces
// ============================================================

// DocumentStore 文档 CRUD
type DocumentStore interface {
	SaveDocument(doc *models.Document) error
	GetDocumentByID(id string, userID string) (*models.Document, error)
	GetDocumentByPath(path string, userID string) (*models.Document, error)
	DeleteDocument(id string, userID string) error
	DeleteDocumentByPath(path string, userID string) error
	ListDocuments(userID string, limit, offset int) ([]*models.Document, error)
	ListAllDocuments(limit, offset int) ([]*models.Document, error)
	GetDocumentsCount(userID string) (int, error)
}

// ChunkStore 分块 CRUD
type ChunkStore interface {
	SaveChunks(chunks []*models.Chunk) error
	GetChunk(id string, userID string) (*models.Chunk, error)
	GetChunkByHash(hash string, userID string) (*models.Chunk, error)
	DeleteChunksByDocument(docID string, userID string) error
	GetChunksCount(userID string) (int, error)
}

// VectorStore 向量存储与搜索
type VectorStore interface {
	GetVectorsCount(userID string) (int, error)
	BuildHNSWIndex() error
	SaveVectorIndex() error
}

// Searcher 搜索接口
type Searcher interface {
	VectorSearch(vector []float32, userID string, topK int) ([]*models.SearchResult, error)
	FTSSearch(query string, userID string, topK int) ([]*models.SearchResult, error)
}

// CacheStore 搜索缓存
type CacheStore interface {
	GetCachedSearch(query string, userID string, mode string, topK int) ([]*models.SearchResult, bool)
	SetCachedSearch(query string, userID string, mode string, topK int, results []*models.SearchResult, ttl time.Duration) error
	InvalidateSearchCache() error
	InvalidateUserSearchCache(userID string) error
	StartCacheCleanup(interval time.Duration)
}

// MetaStore 元数据读写
type MetaStore interface {
	GetMetadata(key string) (string, error)
	SetMetadata(key, value string) error
}

// IndexProgressStore 索引进度（断点恢复）
type IndexProgressStore interface {
	SaveIndexProgress(p *models.IndexProgress) error
	GetIndexProgress(rootPath string) (*models.IndexProgress, error)
	ListIndexProgress(limit, offset int) ([]*models.IndexProgress, error)
	DeleteIndexProgress(id int) error
	CompleteIndexProgress(rootPath string) error
	FailIndexProgress(rootPath string, errMsg string) error
	InitIndexProgressTable() error
}

// UserStore 用户管理
type UserStore interface {
	SaveUser(user *models.User) error
	GetUserByID(id string) (*models.User, error)
	GetUserByUsername(username string) (*models.User, error)
	ListUsers(limit, offset int) ([]*models.User, error)
	DeleteUser(id string) error
	DeleteUserData(userID string) error
}

// TokenStore 认证 Token
type TokenStore interface {
	SaveToken(token *models.AuthToken) error
	GetToken(token string) (*models.AuthToken, error)
	DeleteToken(token string) error
	DeleteExpiredTokens() (int, error)
}

// APIKeyStore API Key 管理
type APIKeyStore interface {
	SaveAPIKey(apiKey *models.APIKey) error
	GetAPIKeyByHash(keyHash string) (*models.APIKey, error)
	DeleteAPIKey(keyHash string) error
	UpdateAPIKeyLastUsed(keyHash string) error
	ListAPIKeys(limit, offset int) ([]*models.APIKey, error)
	ListAPIKeysByUser(userID string) ([]*models.APIKey, error)
}

// SystemStore 系统操作
type SystemStore interface {
	Close() error
	DedupChunks() (removed int, groups int, err error)
	DedupByVector(threshold float64) (removed int, candidates int, err error)
	DedupByMinHash(threshold float64) (removed int, candidates int, err error)
}

// ============================================================
// 兼容层：完整 Storage 接口（组合所有小接口）
// 旧代码继续使用 Storage，新代码使用特定小接口
// ============================================================

// Storage 完整存储接口（保持向后兼容）
type Storage interface {
	DocumentStore
	ChunkStore
	VectorStore
	Searcher
	CacheStore
	MetaStore
	IndexProgressStore
	UserStore
	TokenStore
	APIKeyStore
	SystemStore
}
