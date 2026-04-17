package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lh123aa/cortex/internal/models"
	"github.com/lh123aa/cortex/internal/rag"
	"github.com/lh123aa/cortex/internal/search"
	"github.com/lh123aa/cortex/internal/storage"
	"go.uber.org/zap"
)

// HTTPServer wraps http.Server with graceful shutdown support
type HTTPServer struct {
	*http.Server
}

// Router returns the Gin router for external use
func (s *RESTServer) Router() *gin.Engine {
	return s.router
}

// ListenAndServe starts the HTTP server
func (s *RESTServer) ListenAndServe(addr string) error {
	return s.router.Run(addr)
}

// Shutdown gracefully shuts down the server
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	return s.Server.Shutdown(ctx)
}

// parsePositiveInt parses a string to positive int, returns 0 if invalid
func parsePositiveInt(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, err
	}
	return n, nil
}

// RESTServer Gin-based HTTP API server
type RESTServer struct {
	engine *search.HybridSearchEngine
	rag    *rag.RAGBuilder
	storage storage.Storage
	logger  *zap.Logger
	router  *gin.Engine
}

func NewRESTServer(se *search.HybridSearchEngine, st storage.Storage, log *zap.Logger) *RESTServer {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	s := &RESTServer{
		engine: se,
		rag:    rag.NewRAGBuilder(se),
		storage: st,
		logger:  log,
		router:  r,
	}
	s.registerRoutes()
	return s
}

func (s *RESTServer) registerRoutes() {
	// Health
	s.router.GET("/health", s.handleHealth)

	// Search
	s.router.GET("/v1/search", s.handleSearch)

	// Context (RAG)
	s.router.GET("/v1/context", s.handleContext)

	// Documents
	s.router.GET("/v1/docs", s.handleListDocs)
	s.router.GET("/v1/docs/:id", s.handleGetDoc)

	// Stats
	s.router.GET("/v1/stats", s.handleStats)
}

func (s *RESTServer) Run(addr string) error {
	s.logger.Info("starting REST API server", zap.String("addr", addr))
	return s.router.Run(addr)
}

// --- Handlers ---

func (s *RESTServer) handleHealth(c *gin.Context) {
	// storage has version info?
	if _, err := s.storage.GetMetadata("version"); err == nil {
		c.JSON(200, gin.H{"status": "ok"})
		return
	}
	c.JSON(200, gin.H{"status": "ok"})
}

func (s *RESTServer) handleSearch(c *gin.Context) {
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

	mode := c.DefaultQuery("mode", "hybrid")

	opts := models.SearchOptions{TopK: topK, Mode: mode}
	results, err := s.engine.Search(context.Background(), q, opts)
	if err != nil {
		s.logger.Error("search failed", zap.Error(err))
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Enrich with document path
	enriched := make([]gin.H, len(results))
	for i, r := range results {
		path := r.Chunk.DocumentID
		if doc, _ := s.storage.GetDocumentByID(r.Chunk.DocumentID); doc != nil {
			path = doc.Path
		}
		enriched[i] = gin.H{
			"rank":          i + 1,
			"score":        r.Score,
			"path":         path,
			"section":      r.Chunk.HeadingPath,
			"content_raw":  r.Chunk.ContentRaw,
			"token_count":  r.Chunk.TokenCount,
		}
	}

	c.JSON(200, gin.H{
		"query":  q,
		"total":  len(results),
		"results": enriched,
	})
}

func (s *RESTServer) handleContext(c *gin.Context) {
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

	opts := models.SearchOptions{TopK: 50, Mode: "hybrid"}
	rc, err := s.rag.BuildContext(context.Background(), q, budget, opts)
	if err != nil {
		s.logger.Error("RAG context build failed", zap.Error(err))
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"query":         q,
		"context":        rc.Context,
		"token_count":    rc.TokenCount,
		"token_budget":   rc.TokenBudget,
		"truncated":      rc.Truncated,
	})
}

func (s *RESTServer) handleListDocs(c *gin.Context) {
	docs, err := s.storage.ListDocuments(100, 0)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"total": len(docs), "documents": docs})
}

func (s *RESTServer) handleGetDoc(c *gin.Context) {
	id := c.Param("id")
	doc, err := s.storage.GetDocumentByID(id)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if doc == nil {
		c.JSON(404, gin.H{"error": "document not found"})
		return
	}
	c.JSON(200, doc)
}

func (s *RESTServer) handleStats(c *gin.Context) {
	docsCount, _ := s.storage.GetDocumentsCount()
	chunksCount, _ := s.storage.GetChunksCount()
	vectorsCount, _ := s.storage.GetVectorsCount()
	c.JSON(200, gin.H{
		"documents_count": docsCount,
		"chunks_count":     chunksCount,
		"vectors_count":    vectorsCount,
		"version":          "dev",
	})
}
