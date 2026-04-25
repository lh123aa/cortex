package api

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lh123aa/cortex/internal/chunker"
	"github.com/lh123aa/cortex/internal/embedding"
	"github.com/lh123aa/cortex/internal/models"
	"github.com/lh123aa/cortex/internal/rag"
	"github.com/lh123aa/cortex/internal/search"
	"github.com/lh123aa/cortex/internal/storage"
	"go.uber.org/zap"
)

// MemoryHandler 记忆系统 API 处理
type MemoryHandler struct {
	storage  storage.Storage
	engine   *search.HybridSearchEngine
	rag      *rag.RAGBuilder
	em       embedding.EmbeddingProvider
	chunker  chunker.Chunker
	logger   *zap.Logger
}

// NewMemoryHandler 创建记忆处理器
func NewMemoryHandler(s storage.Storage, se *search.HybridSearchEngine, em embedding.EmbeddingProvider, logger *zap.Logger) *MemoryHandler {
	// 使用文本 chunker 用于记忆内容分块
	tk, _ := chunker.NewTextChunker(chunker.ChunkConfig{
		MinChars:   50,
		MaxTokens: 512,
	})

	return &MemoryHandler{
		storage:  s,
		engine:   se,
		rag:      rag.NewRAGBuilder(se),
		em:       em,
		chunker:  tk,
		logger:   logger,
	}
}

// WriteMemory 写入单条记忆
func (h *MemoryHandler) WriteMemory(c *gin.Context) {
	var req models.MemoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// 获取用户ID
	userID := ""
	if uc := GetUserContext(c); uc != nil {
		userID = uc.UserID
	}

	// 生成记忆ID（基于内容hash）
	contentHash := sha256.Sum256([]byte(req.Content))
	memoryID := hex.EncodeToString(contentHash[:16])

	// 如果没有提供摘要，自动生成（取前100字符）
	summary := req.Summary
	if summary == "" && len(req.Content) > 100 {
		summary = req.Content[:100] + "..."
	} else if summary == "" {
		summary = req.Content
	}

	// 保存记忆到数据库
	memory := &models.Memory{
		ID:        memoryID,
		UserID:    userID,
		Content:   req.Content,
		Summary:   summary,
		Tags:      req.Tags,
		Source:    req.Source,
		SourceID:  memoryID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 分块并生成向量
	chunks, err := h.chunker.Chunk(req.Content, fmt.Sprintf("memory://%s", memoryID))
	if err != nil {
		h.logger.Error("failed to chunk memory", zap.Error(err))
		c.JSON(500, gin.H{"error": "failed to process memory"})
		return
	}

	// 生成 embedding
	if h.em != nil && len(chunks) > 0 {
		texts := make([]string, len(chunks))
		for i, chunk := range chunks {
			texts[i] = chunk.ContentRaw
		}

		embeddings, err := h.em.EmbedBatch(texts)
		if err != nil {
			h.logger.Warn("failed to generate embeddings for memory", zap.Error(err))
		} else {
			for i, chunk := range chunks {
				chunk.Embedding = embeddings[i]
				chunk.EmbeddingModel = h.em.Name()
			}
		}
	}

	// 保存记忆元数据
	// 注意：这里简化处理，实际应存储到专门的 memory 表
	// 目前复用 chunks 表，通过 source 字段区分

	// 创建虚拟文档用于存储记忆
	docID := memoryID
	doc := &models.Document{
		ID:          docID,
		UserID:      userID,
		Path:        fmt.Sprintf("memory://%s", memoryID),
		FileType:    "memory",
		ContentHash: hex.EncodeToString(contentHash[:]),
		ChunkCount:  len(chunks),
		Status:      "indexed",
	}

	if err := h.storage.SaveDocument(doc); err != nil {
		h.logger.Error("failed to save memory document", zap.Error(err))
		c.JSON(500, gin.H{"error": "failed to save memory"})
		return
	}

	// 保存 chunks
	for _, chunk := range chunks {
		chunk.UserID = userID
		chunk.DocumentID = docID
	}
	if err := h.storage.SaveChunks(chunks); err != nil {
		h.logger.Error("failed to save memory chunks", zap.Error(err))
		c.JSON(500, gin.H{"error": "failed to save memory chunks"})
		return
	}

	c.JSON(201, models.MemoryResponse{
		ID:        memoryID,
		Content:   req.Content,
		Summary:   summary,
		Tags:      req.Tags,
		Source:    req.Source,
		CreatedAt: time.Now().Format(time.RFC3339),
	})
}

// WriteMemoryBatch 批量写入记忆
func (h *MemoryHandler) WriteMemoryBatch(c *gin.Context) {
	var req struct {
		Memories []models.MemoryRequest `json:"memories" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// 获取用户ID
	userID := ""
	if uc := GetUserContext(c); uc != nil {
		userID = uc.UserID
	}

	results := make([]models.MemoryResponse, 0, len(req.Memories))
	errs := make([]string, 0)

	for i, mem := range req.Memories {
		// 生成记忆ID（基于内容hash）
		contentHash := sha256.Sum256([]byte(mem.Content))
		memoryID := hex.EncodeToString(contentHash[:16])

		// 如果没有提供摘要，自动生成
		summary := mem.Summary
		if summary == "" && len(mem.Content) > 100 {
			summary = mem.Content[:100] + "..."
		} else if summary == "" {
			summary = mem.Content
		}

		// 分块并生成向量
		chunks, err := h.chunker.Chunk(mem.Content, fmt.Sprintf("memory://%s", memoryID))
		if err != nil {
			errs = append(errs, fmt.Sprintf("[%d] failed to chunk: %v", i, err))
			continue
		}

		// 生成 embedding
		if h.em != nil && len(chunks) > 0 {
			texts := make([]string, len(chunks))
			for j, chunk := range chunks {
				texts[j] = chunk.ContentRaw
			}

			embeddings, err := h.em.EmbedBatch(texts)
			if err != nil {
				h.logger.Warn("failed to generate embeddings for memory", zap.Error(err))
			} else {
				for j, chunk := range chunks {
					chunk.Embedding = embeddings[j]
					chunk.EmbeddingModel = h.em.Name()
				}
			}
		}

		// 创建虚拟文档
		docID := memoryID
		doc := &models.Document{
			ID:          docID,
			UserID:      userID,
			Path:        fmt.Sprintf("memory://%s", memoryID),
			FileType:    "memory",
			ContentHash: hex.EncodeToString(contentHash[:]),
			ChunkCount:  len(chunks),
			Status:      "indexed",
		}

		if err := h.storage.SaveDocument(doc); err != nil {
			errs = append(errs, fmt.Sprintf("[%d] failed to save doc: %v", i, err))
			continue
		}

		// 保存 chunks
		for _, chunk := range chunks {
			chunk.UserID = userID
			chunk.DocumentID = docID
		}
		if err := h.storage.SaveChunks(chunks); err != nil {
			errs = append(errs, fmt.Sprintf("[%d] failed to save chunks: %v", i, err))
			continue
		}

		results = append(results, models.MemoryResponse{
			ID:        memoryID,
			Content:   mem.Content,
			Summary:   summary,
			Tags:      mem.Tags,
			Source:    mem.Source,
			CreatedAt: time.Now().Format(time.RFC3339),
		})
	}

	c.JSON(201, gin.H{
		"total":   len(req.Memories),
		"success": len(results),
		"failed":  len(errs),
		"results": results,
		"errors":  errs,
	})
}

// SearchMemory 搜索记忆
func (h *MemoryHandler) SearchMemory(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		c.JSON(400, gin.H{"error": "q (query) is required"})
		return
	}

	topK := 10
	if tk := c.Query("top_k"); tk != "" {
		if n, err := parsePositiveInt(tk); err == nil {
			topK = n
		}
	}

	userID := ""
	if uc := GetUserContext(c); uc != nil {
		userID = uc.UserID
	}

	opts := models.SearchOptions{
		TopK:   topK,
		Mode:   "hybrid",
		UserID: userID,
	}

	results, err := h.engine.Search(c.Request.Context(), q, opts)
	if err != nil {
		h.logger.Error("memory search failed", zap.Error(err))
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// 过滤出 memory 类型的文档
	memoryResults := make([]models.MemorySearchResult, 0)
	for _, r := range results {
		// 检查是否是记忆（通过 path 判断）
		if doc, _ := h.storage.GetDocumentByID(r.Chunk.DocumentID, userID); doc != nil && doc.FileType == "memory" {
			memoryResults = append(memoryResults, models.MemorySearchResult{
				ID:      r.Chunk.ID,
				Content: r.Chunk.ContentRaw,
				Summary: r.Chunk.HeadingPath, // 简化：使用 heading 作为 summary
				Score:   r.Score,
				Source:  doc.Path,
			})
		}
	}

	c.JSON(200, gin.H{
		"query":   q,
		"total":   len(memoryResults),
		"results": memoryResults,
	})
}

// GetMemoryContext 获取记忆上下文（RAG）
func (h *MemoryHandler) GetMemoryContext(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		c.JSON(400, gin.H{"error": "q (query) is required"})
		return
	}

	budget := 1500
	if b := c.Query("budget"); b != "" {
		if n, err := parsePositiveInt(b); err == nil {
			budget = n
		}
	}

	userID := ""
	if uc := GetUserContext(c); uc != nil {
		userID = uc.UserID
	}

	opts := models.SearchOptions{
		TopK: 50,
		Mode: "hybrid",
		UserID: userID,
	}

	rc, err := h.rag.BuildContext(c.Request.Context(), q, budget, opts)
	if err != nil {
		h.logger.Error("memory context build failed", zap.Error(err))
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"query":        q,
		"context":       rc.Context,
		"token_count":   rc.TokenCount,
		"token_budget":  rc.TokenBudget,
		"truncated":     rc.Truncated,
	})
}

// DeleteMemory 删除记忆
func (h *MemoryHandler) DeleteMemory(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, gin.H{"error": "id is required"})
		return
	}

	userID := ""
	if uc := GetUserContext(c); uc != nil {
		userID = uc.UserID
	}

	// 通过 documentID 删除记忆
	if err := h.storage.DeleteDocumentByPath(fmt.Sprintf("memory://%s", id), userID); err != nil {
		h.logger.Error("failed to delete memory", zap.Error(err))
		c.JSON(500, gin.H{"error": "failed to delete memory"})
		return
	}

	// 删除后失效缓存，避免搜索返回已删除记忆
	if err := h.storage.InvalidateSearchCache(); err != nil {
		h.logger.Warn("failed to invalidate search cache after memory deletion", zap.Error(err))
	}

	c.JSON(200, gin.H{"message": "memory deleted"})
}

// RegisterMemoryRoutes 注册记忆相关路由
func (s *RESTServer) RegisterMemoryRoutes(mh *MemoryHandler) {
	// 记忆写入
	s.router.POST("/v1/memory", mh.WriteMemory)
	s.router.POST("/v1/memory/batch", mh.WriteMemoryBatch)

	// 记忆搜索
	s.router.GET("/v1/memory/search", mh.SearchMemory)
	s.router.GET("/v1/memory/context", mh.GetMemoryContext)

	// 记忆管理
	s.router.DELETE("/v1/memory/:id", mh.DeleteMemory)
}