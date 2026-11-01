package api

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lh123aa/cortex/internal/auth"
	"github.com/lh123aa/cortex/internal/embedding"
	"github.com/lh123aa/cortex/internal/models"
	"github.com/lh123aa/cortex/internal/rag"
	"github.com/lh123aa/cortex/internal/search"
	"github.com/lh123aa/cortex/internal/snippet"
	"github.com/lh123aa/cortex/internal/storage"
	"github.com/lh123aa/cortex/internal/version"
	"github.com/lh123aa/cortex/pkg/ext"
	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
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
			Addr:              addr,
			Handler:           s.router,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      30 * time.Second,
			ReadHeaderTimeout: 10 * time.Second,
			MaxHeaderBytes:    1 << 20, // 1MB
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

	embeddingProvider string // embedding provider name, "none" for FTS-only
	mcpServer         *MCPServer

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

	providerName := "none"
	if em != nil {
		providerName = em.Name()
	}

	mcpSrv := NewMCPServer(se, st, em, log)

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

		embeddingProvider: providerName,
		mcpServer:         mcpSrv,
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

	providerName := "none"
	if em != nil {
		providerName = em.Name()
	}

	mcpSrv := NewMCPServer(se, st, em, log)

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

		embeddingProvider: providerName,
		mcpServer:         mcpSrv,
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

// ipRateLimiter 简单的 IP 级速率限制（令牌桶）
// 每个 IP 最多 rate 次/秒，burst 为最大突发
func ipRateLimiter(rate int, burst int) gin.HandlerFunc {
	type client struct {
		tokens    int
		lastCheck time.Time
	}
	var mu sync.Mutex
	clients := make(map[string]*client)

	// 定期清理过期客户端
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			mu.Lock()
			for ip, c := range clients {
				if time.Since(c.lastCheck) > 10*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		mu.Lock()
		cl, exists := clients[ip]
		now := time.Now()
		if !exists {
			clients[ip] = &client{tokens: burst - 1, lastCheck: now}
			mu.Unlock()
			c.Next()
			return
		}
		// 补充令牌
		elapsed := now.Sub(cl.lastCheck).Seconds()
		cl.tokens += int(elapsed * float64(rate))
		if cl.tokens > burst {
			cl.tokens = burst
		}
		cl.lastCheck = now

		if cl.tokens <= 0 {
			mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests, try again later",
			})
			return
		}
		cl.tokens--
		mu.Unlock()
		c.Next()
	}
}

func (s *RESTServer) registerRoutes() {
	// 企业版扩展必须最先注册：
	// 扩展常用 gin 的 Use() 安装全局中间件（例如审计日志），
	// 而 gin 的中间件只对「注册之后」的路由生效 —— 放在末尾会静默失效。
	if names := ext.RegisterHTTP(s.router); len(names) > 0 {
		s.logger.Info("enterprise extensions enabled (HTTP)",
			zap.Strings("extensions", names))
	}

	// Health - 始终公开（增强版）
	s.router.GET("/health", s.handleHealth)
	s.router.GET("/health/ready", s.handleReady)
	s.router.GET("/health/live", s.handleLive)

	// Auth routes (注册/登录) — 添加暴力破解防护
	authRoutes := s.router.Group("/auth")
	authRoutes.Use(ipRateLimiter(5, 10)) // 每个 IP 5次/秒，最大突发10次
	{
		authRoutes.POST("/register", s.handleRegister)
		authRoutes.POST("/login", s.handleLogin)
		authRoutes.POST("/logout", s.handleLogout)
	}

	// 认证中间件选择：
	//   - 认证开启：用 AuthMiddleware.RequireAuth()，它同时支持
	//     Authorization: Bearer <login-token> 与 X-API-Key。
	//     早期实现这里挂的是旧版 API Key 中间件，导致「登录拿到的 token 不能用」——
	//     客户端只有 X-API-Key 一条路，登录接口形同虚设。
	//   - 认证关闭：空 handler（单机本地场景）。
	authHandler := func(c *gin.Context) { c.Next() }
	if s.authEnabled && s.authMiddleware != nil {
		authHandler = s.authMiddleware.RequireAuth()
	} else if s.authEnabled {
		// 兜底：只传了 authService 没传 middleware 时，退回旧 API Key 校验
		authHandler = s.auth.Middleware()
	}

	// 写权限中间件：仅 admin/editor 可写；viewer 只读。
	// 认证关闭时不限制（单机本地场景）。
	writeHandler := func(c *gin.Context) { c.Next() }
	adminOnlyHandler := func(c *gin.Context) { c.Next() }
	if s.authEnabled && s.authMiddleware != nil {
		writeHandler = s.authMiddleware.RequireWrite()
		adminOnlyHandler = s.authMiddleware.RequireAdmin()
	}

	// Protected routes
	protected := s.router.Group("/v1")
	protected.Use(authHandler)
	{
		// Search
		protected.GET("/search", s.handleSearch)
		protected.GET("/suggest", s.handleSuggest)

		// Context (RAG)
		protected.GET("/context", s.handleContext)

		// Documents
		protected.GET("/docs", s.handleListDocs)
		protected.GET("/docs/:id", s.handleGetDoc)

		// Stats
		protected.GET("/stats", s.handleStats)

		// Memory (记忆系统) — 读接口对所有角色开放，写接口要求 editor 及以上
		protected.GET("/memory/search", s.memory.SearchMemory)
		protected.GET("/memory/context", s.memory.GetMemoryContext)
		protected.DELETE("/memory/:id", writeHandler, s.memory.DeleteMemory)

		writes := protected.Group("", writeHandler)
		{
			writes.POST("/memory", s.memory.WriteMemory)
			writes.POST("/memory/batch", s.memory.WriteMemoryBatch)
		}
	}

	// Admin routes：密钥管理仅管理员可访问。
	// 早期实现只挂了 authHandler，等于任何登录用户都能列出全局密钥 —— 已修正。
	admin := s.router.Group("/admin")
	admin.Use(authHandler, adminOnlyHandler)
	{
		admin.GET("/keys", s.handleListKeys)
	}

	// Web管理界面（无需认证）
	adminHandler := NewAdminHandler(s)
	adminHandler.RegisterRoutes(s.router)

	// Internal: Index progress (受保护)
	internal := s.router.Group("/internal")
	internal.Use(authHandler)
	{
		internal.GET("/progress/:root_path", s.handleIndexProgress)
	}

	// MCP over HTTP (Streamable HTTP Transport)
	mcpHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return s.mcpServer.Server()
	}, nil)
	s.router.Any("/mcp", gin.WrapH(mcpHandler))
}

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

func (s *RESTServer) ftsHint() string {
	if s.embeddingProvider == "none" {
		return "FTS-only mode. Install Ollama + nomic-embed-text for semantic search."
	}
	return ""
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

	offset := 0
	if off := c.Query("offset"); off != "" {
		if n, err := parsePositiveInt(off); err == nil {
			offset = n
		}
	}

	mode := c.DefaultQuery("mode", "hybrid")

	// 获取用户ID进行隔离
	userID := ""
	if uc := GetUserContext(c); uc != nil {
		userID = uc.UserID
	}

	opts := models.SearchOptions{TopK: topK, Offset: offset, Mode: mode, UserID: userID}
	results, err := s.engine.Search(c.Request.Context(), q, opts)
	if err != nil {
		s.logger.Error("search failed", zap.Error(err))
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// 组装响应。
	//
	// 性能要点（实测 limit=10 曾返回 1.9 MB）：
	//   - 默认只返回截断后的单个 content 字段，不再同时返回 content_raw 全文；
	//   - 需要全文时显式传 ?full=1；
	//   - 文档路径直接复用检索层带出的 FilePath，不再逐条查 documents（N+1）。
	full := c.Query("full") == "1" || c.Query("full") == "true"
	enriched := make([]gin.H, len(results))
	for i, r := range results {
		if r == nil || r.Chunk == nil {
			continue
		}
		path := r.FilePath
		if path == "" {
			path = r.Chunk.DocumentID
			if doc, _ := s.storage.GetDocumentByID(r.Chunk.DocumentID, userID); doc != nil {
				path = doc.Path
			}
		}
		display := r.Chunk.DisplayText()
		item := gin.H{
			"rank":    i + 1,
			"score":   r.Score,
			"path":    path,
			"section": r.Chunk.HeadingPath,
			"content": truncateRunes(display, searchSnippetRunes),
		}
		if full {
			item["content"] = display
			item["content_raw"] = r.Chunk.ContentRaw
			item["token_count"] = r.Chunk.TokenCount
		}
		enriched[i] = item
	}

	c.JSON(200, gin.H{"query": q, "total": len(results), "results": enriched})
}

// searchSnippetRunes 是默认返回的内容长度上限（按字符，不是字节）。
// 800 字符足以让调用方判断相关性；需要全文请传 ?full=1。
const searchSnippetRunes = 800

func (s *RESTServer) handleSuggest(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		c.JSON(400, gin.H{"error": "q (query) is required"})
		return
	}

	topK := 5
	if tk := c.Query("top_k"); tk != "" {
		if n, err := parsePositiveInt(tk); err == nil {
			topK = n
		}
	}
	if topK > 20 {
		topK = 20
	}

	currentFile := c.Query("file")

	pe := s.engine.GetPrefetchEngine()
	if pe != nil {
		results := pe.Suggest(c.Request.Context(), q, currentFile, topK)
		if len(results) > 0 {
			enriched := make([]gin.H, len(results))
			for i, r := range results {
				content := snippet.Build(r.Chunk.DisplayText(), q, snippet.Options{Width: 300})
				enriched[i] = gin.H{
					"rank":        i + 1,
					"score":       r.Score,
					"section":     r.Chunk.HeadingPath,
					"content":     content,
					"content_raw": r.Chunk.ContentRaw,
				}
			}
			c.JSON(200, gin.H{"query": q, "source": "cache", "total": len(results), "results": enriched, "hint": s.ftsHint()})
			return
		}
	}

	userID := ""
	if uc := GetUserContext(c); uc != nil {
		userID = uc.UserID
	}
	opts := models.SearchOptions{TopK: topK, Mode: "hybrid", UserID: userID}
	results, err := s.engine.Search(c.Request.Context(), q, opts)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	enriched := make([]gin.H, len(results))
	for i, r := range results {
		content := snippet.Build(r.Chunk.DisplayText(), q, snippet.Options{Width: 300})
		enriched[i] = gin.H{
			"rank":        i + 1,
			"score":       r.Score,
			"section":     r.Chunk.HeadingPath,
			"content":     content,
			"content_raw": r.Chunk.ContentRaw,
		}
	}
	c.JSON(200, gin.H{"query": q, "source": "search", "total": len(results), "results": enriched, "hint": s.ftsHint()})
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
	rc, err := s.rag.BuildContext(c.Request.Context(), q, budget, opts)
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
		"version":         version.Version,
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
