package api

import (
	"context"
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/lh123aa/cortex/internal/auth"
	"github.com/lh123aa/cortex/internal/embedding"
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
	s.httpServer = &HTTPServer{
		Server: &http.Server{
			Addr:    addr,
			Handler: s.router,
		},
	}
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the REST server
func (s *RESTServer) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
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
	engine     *search.HybridSearchEngine
	rag        *rag.RAGBuilder
	storage    storage.Storage
	logger     *zap.Logger
	router     *gin.Engine
	health     *HealthChecker
	memory     *MemoryHandler // 记忆系统处理器
	httpServer *HTTPServer    // 用于 graceful shutdown

	// Auth
	authService    *auth.AuthService
	authMiddleware *AuthMiddleware
	auth           *APIKeyAuth // 旧的 API Key 认证 (兼容)
	authEnabled    bool
	authKeys       map[string]string // key -> name mapping for audit
	authMu         sync.RWMutex
}

func NewRESTServer(se *search.HybridSearchEngine, st storage.Storage, em embedding.EmbeddingProvider, log *zap.Logger) *RESTServer {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(TimeoutMiddleware(DefaultTimeout)) // 全局超时控制
	r.Use(DefaultRateLimitMiddleware())      // 全局限流

	mh := NewMemoryHandler(st, se, em, log)

	s := &RESTServer{
		engine:      se,
		rag:         rag.NewRAGBuilder(se),
		storage:     st,
		logger:      log,
		router:      r,
		health:      NewHealthChecker(st, em),
		memory:      mh,
		auth:        NewAPIKeyAuth("X-API-Key", "api_key"),
		authEnabled: false,
		authKeys:    make(map[string]string),
	}
	s.registerRoutes()
	return s
}

// NewRESTServerWithAuth 创建带认证的 RESTServer
func NewRESTServerWithAuth(se *search.HybridSearchEngine, st storage.Storage, em embedding.EmbeddingProvider, log *zap.Logger, authService *auth.AuthService) *RESTServer {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(TimeoutMiddleware(DefaultTimeout)) // 全局超时控制
	r.Use(DefaultRateLimitMiddleware())      // 全局限流

	mh := NewMemoryHandler(st, se, em, log)

	s := &RESTServer{
		engine:         se,
		rag:            rag.NewRAGBuilder(se),
		storage:        st,
		logger:         log,
		router:         r,
		health:         NewHealthChecker(st, em),
		memory:         mh,
		authService:    authService,
		authMiddleware: NewAuthMiddleware(authService),
		auth:           NewAPIKeyAuth("X-API-Key", "api_key"),
		authEnabled:    true,
		authKeys:       make(map[string]string),
	}
	s.registerRoutes()
	return s
}

// EnableAuth 启用 API Key 认证
func (s *RESTServer) EnableAuth() {
	s.authEnabled = true
}

// DisableAuth 禁用 API Key 认证（用于开发模式）
func (s *RESTServer) DisableAuth() {
	s.authEnabled = false
}

// AddAPIKey 添加一个 API key，可选关联名称用于审计
func (s *RESTServer) AddAPIKey(key string, name string) {
	s.auth.AddKey(key)
	s.authMu.Lock()
	s.authKeys[key] = name
	s.authMu.Unlock()
}

// RemoveAPIKey 移除一个 API key
func (s *RESTServer) RemoveAPIKey(key string) {
	s.auth.RemoveKey(key)
	s.authMu.Lock()
	delete(s.authKeys, key)
	s.authMu.Unlock()
}

// ListAPIKeys 返回所有 API key 的名称列表（不返回 key 本身）
func (s *RESTServer) ListAPIKeys() []string {
	s.authMu.RLock()
	defer s.authMu.RUnlock()
	names := make([]string, 0, len(s.authKeys))
	for _, name := range s.authKeys {
		names = append(names, name)
	}
	return names
}

func (s *RESTServer) registerRoutes() {
	// Health - 始终公开（增强版）
	s.router.GET("/health", s.handleHealth)
	s.router.GET("/health/ready", s.handleReady)
	s.router.GET("/health/live", s.handleLive)

	// Auth routes (注册/登录)
	authRoutes := s.router.Group("/auth")
	{
		authRoutes.POST("/register", s.handleRegister)
		authRoutes.POST("/login", s.handleLogin)
		authRoutes.POST("/logout", s.handleLogout)
	}

	// API Key auth middleware wrapper
	authHandler := s.auth.Middleware()
	if !s.authEnabled {
		// 如果认证被禁用，使用一个空的 handler
		authHandler = func(c *gin.Context) { c.Next() }
	}

	// Protected routes
	protected := s.router.Group("/v1")
	protected.Use(authHandler)
	{
		// Search
		protected.GET("/search", s.handleSearch)

		// Context (RAG)
		protected.GET("/context", s.handleContext)

		// Documents
		protected.GET("/docs", s.handleListDocs)
		protected.GET("/docs/:id", s.handleGetDoc)

		// Stats
		protected.GET("/stats", s.handleStats)

		// Memory (记忆系统)
		protected.POST("/memory", s.memory.WriteMemory)
		protected.POST("/memory/batch", s.memory.WriteMemoryBatch)
		protected.GET("/memory/search", s.memory.SearchMemory)
		protected.GET("/memory/context", s.memory.GetMemoryContext)
		protected.DELETE("/memory/:id", s.memory.DeleteMemory)
	}

	// Admin routes (也受保护，用于密钥管理)
	admin := s.router.Group("/admin")
	admin.Use(authHandler)
	{
		admin.GET("/keys", s.handleListKeys)
	}

	// Internal: Index progress (受保护)
	internal := s.router.Group("/internal")
	internal.Use(authHandler)
	{
		internal.GET("/progress/:root_path", s.handleIndexProgress)
	}
}

func (s *RESTServer) Run(addr string) error {
	s.logger.Info("starting REST API server", zap.String("addr", addr), zap.Bool("auth_enabled", s.authEnabled))
	return s.router.Run(addr)
}

// --- Auth Handlers ---

func (s *RESTServer) handleRegister(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required,min=3,max=50"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if s.authService == nil {
		c.JSON(500, gin.H{"error": "auth service not initialized"})
		return
	}

	user, err := s.authService.Register(&models.RegisterRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(201, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"role":     user.Role,
	})
}

func (s *RESTServer) handleLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if s.authService == nil {
		c.JSON(500, gin.H{"error": "auth service not initialized"})
		return
	}

	token, err := s.authService.Login(&models.LoginRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}

	// Login returns token, we need to get user from token
	user, err := s.authService.GetUserByToken(token.Token)
	if err != nil {
		c.JSON(401, gin.H{"error": "login failed"})
		return
	}

	c.JSON(200, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
		},
	})
}

func (s *RESTServer) handleLogout(c *gin.Context) {
	// 从 context 获取 token 并使其失效
	token := extractBearerToken(c)
	if token != "" && s.authService != nil {
		s.authService.Logout(token)
	}
	c.JSON(200, gin.H{"message": "logged out"})
}

// --- Handlers ---

func (s *RESTServer) handleHealth(c *gin.Context) {
	// 返回完整健康状态（包含所有检查）
	status := s.health.FullCheck()
	if status.Status == "healthy" {
		c.JSON(200, status)
	} else {
		c.JSON(503, status) // Service Unavailable
	}
}

func (s *RESTServer) handleReady(c *gin.Context) {
	// 就绪检查
	if s.health.ReadyCheck() {
		c.JSON(200, gin.H{"status": "ready"})
	} else {
		c.JSON(503, gin.H{"status": "not_ready"})
	}
}

func (s *RESTServer) handleLive(c *gin.Context) {
	// 存活检查
	if s.health.LiveCheck() {
		c.JSON(200, gin.H{"status": "alive"})
	} else {
		c.JSON(503, gin.H{"status": "dead"})
	}
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

	// 获取用户ID进行隔离
	userID := ""
	if uc := GetUserContext(c); uc != nil {
		userID = uc.UserID
	}

	opts := models.SearchOptions{TopK: topK, Mode: mode, UserID: userID}
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
		if doc, _ := s.storage.GetDocumentByID(r.Chunk.DocumentID, userID); doc != nil {
			path = doc.Path
		}
		enriched[i] = gin.H{
			"rank":        i + 1,
			"score":       r.Score,
			"path":        path,
			"section":     r.Chunk.HeadingPath,
			"content_raw": r.Chunk.ContentRaw,
			"token_count": r.Chunk.TokenCount,
		}
	}

	c.JSON(200, gin.H{
		"query":   q,
		"total":   len(results),
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

	// 获取用户ID进行隔离
	userID := ""
	if uc := GetUserContext(c); uc != nil {
		userID = uc.UserID
	}

	opts := models.SearchOptions{TopK: 50, Mode: "hybrid", UserID: userID}
	rc, err := s.rag.BuildContext(context.Background(), q, budget, opts)
	if err != nil {
		s.logger.Error("RAG context build failed", zap.Error(err))
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"query":        q,
		"context":      rc.Context,
		"token_count":  rc.TokenCount,
		"token_budget": rc.TokenBudget,
		"truncated":    rc.Truncated,
	})
}

func (s *RESTServer) handleListDocs(c *gin.Context) {
	// 获取用户ID进行隔离
	userID := ""
	if uc := GetUserContext(c); uc != nil {
		userID = uc.UserID
	}

	docs, err := s.storage.ListDocuments(userID, 100, 0)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"total": len(docs), "documents": docs})
}

func (s *RESTServer) handleGetDoc(c *gin.Context) {
	id := c.Param("id")

	// 获取用户ID进行隔离
	userID := ""
	if uc := GetUserContext(c); uc != nil {
		userID = uc.UserID
	}

	doc, err := s.storage.GetDocumentByID(id, userID)
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
	// 获取用户ID进行隔离
	userID := ""
	if uc := GetUserContext(c); uc != nil {
		userID = uc.UserID
	}

	docsCount, _ := s.storage.GetDocumentsCount(userID)
	chunksCount, _ := s.storage.GetChunksCount(userID)
	vectorsCount, _ := s.storage.GetVectorsCount(userID)
	c.JSON(200, gin.H{
		"documents_count": docsCount,
		"chunks_count":    chunksCount,
		"vectors_count":   vectorsCount,
		"version":         "dev",
	})
}

// handleListKeys 返回所有 API key 的名称（不返回 key 本身）
func (s *RESTServer) handleListKeys(c *gin.Context) {
	keys := s.ListAPIKeys()
	c.JSON(200, gin.H{"keys": keys, "count": len(keys)})
}

// handleIndexProgress 获取索引进度（内部使用）
func (s *RESTServer) handleIndexProgress(c *gin.Context) {
	rootPath := c.Param("root_path")
	if rootPath == "" {
		c.JSON(400, gin.H{"error": "root_path is required"})
		return
	}

	progress, err := s.storage.GetIndexProgress(rootPath)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if progress == nil {
		c.JSON(404, gin.H{"error": "no index progress found"})
		return
	}

	c.JSON(200, progress)
}
