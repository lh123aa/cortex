package storage

import (
	"github.com/lh123aa/cortex/internal/models"
)

// Storage 定义了Cortex核心存储引擎必须实现的所有方法
// 用于解耦具体数据库实现(目前为SQLite)
type Storage interface {
	// 文档操作
	SaveDocument(doc *models.Document) error
	GetDocumentByID(id string) (*models.Document, error)
	GetDocumentByPath(path string) (*models.Document, error)
	DeleteDocument(id string) error
	DeleteDocumentByPath(path string) error
	ListDocuments(limit, offset int) ([]*models.Document, error)
	GetDocumentsCount() (int, error)

	// 分块操作
	SaveChunks(chunks []*models.Chunk) error
	GetChunk(id string) (*models.Chunk, error)
	DeleteChunksByDocument(docID string) error
	GetChunksCount() (int, error)

	// 向量操作
	GetVectorsCount() (int, error)

	// 搜索操作
	// 向量检索(余弦相似度)，需预加载至内存或利用特定算法
	VectorSearch(vector []float32, topK int) ([]*models.SearchResult, error)
	// FTS 全文检索
	FTSSearch(query string, topK int) ([]*models.SearchResult, error)

	// 元数据读写
	GetMetadata(key string) (string, error)
	SetMetadata(key, value string) error

	// 关闭数据库连接
	Close() error
}
