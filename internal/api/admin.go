package api

import (
	"embed"
	"html/template"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lh123aa/cortex/internal/models"
	"github.com/lh123aa/cortex/internal/snippet"
	"github.com/lh123aa/cortex/internal/version"
	"github.com/lh123aa/cortex/pkg/license"
)

//go:embed admin.html
var adminFS embed.FS

// adminTemplate 由 admin.html 直接解析。
// 注意：ParseFS 会把文件内容挂到「admin.html」这个名字的关联模板上，
// 因此必须用 template.ParseFS 而不是 template.New("admin").ParseFS ——
// 后者的根模板是空的，Execute 根模板只会输出 0 字节（旧版 /admin 空白的原因）。
var adminTemplate = template.Must(template.ParseFS(adminFS, "admin.html"))

// AdminHandler 管理界面
type AdminHandler struct {
	server    *RESTServer
	startedAt time.Time
}

// NewAdminHandler 创建管理界面处理器
func NewAdminHandler(s *RESTServer) *AdminHandler {
	return &AdminHandler{server: s, startedAt: time.Now()}
}

// RegisterRoutes 注册管理界面路由。
// 管理面板默认只允许本机访问；如确需远程访问，设置 CORTEX_ADMIN_ALLOW_REMOTE=1。
func (h *AdminHandler) RegisterRoutes(r *gin.Engine) {
	g := r.Group("/admin")
	g.Use(adminLocalOnly())
	g.GET("", h.handleAdmin)
	g.GET("/", h.handleAdmin)
	g.GET("/api/status", h.handleAPIStatus)
	g.GET("/api/search", h.handleAPISearch)
	g.GET("/api/docs", h.handleAPIDocs)
	g.GET("/api/health", h.handleAPIHealth)
}

func (h *AdminHandler) handleAdmin(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Cache-Control", "no-store")
	_ = adminTemplate.Execute(c.Writer, nil)
}

// handleAPIStatus 返回控制台需要的真实运行数据。
func (h *AdminHandler) handleAPIStatus(c *gin.Context) {
	docs, chunks, vectors := 0, 0, 0
	var storageUsed int64
	if h.server.storage != nil {
		docs, _ = h.server.storage.GetDocumentsCount("")
		chunks, _ = h.server.storage.GetChunksCount("")
		vectors, _ = h.server.storage.GetVectorsCount("")
		storageUsed, _ = h.server.storage.CalculateStorageUsed("")
	}

	plan := "free"
	features := []string{}
	email := ""
	expiresAt := ""
	seats := 0
	licenseError := ""
	if lic, err := license.Current(); err != nil {
		licenseError = err.Error()
	} else if lic != nil {
		plan = string(lic.Plan)
		features = lic.Features
		email = lic.Email
		seats = lic.Seats
		if !lic.ExpiresAt.IsZero() {
			expiresAt = lic.ExpiresAt.UTC().Format(time.RFC3339)
		}
	}

	healthStatus := "unknown"
	if h.server.health != nil {
		healthStatus = h.server.health.FullCheck().Status
	}

	embeddingProvider := h.server.embeddingProvider
	if embeddingProvider == "" {
		embeddingProvider = "none"
	}

	c.JSON(http.StatusOK, gin.H{
		"version":             version.Version,
		"commit":              version.Commit,
		"built_at":            version.Date,
		"plan":                plan,
		"plan_email":          email,
		"plan_seats":          seats,
		"plan_expires_at":     expiresAt,
		"plan_features":       features,
		"license_error":       licenseError,
		"documents":           docs,
		"chunks":              chunks,
		"vectors":             vectors,
		"storage_used_bytes":  storageUsed,
		"storage_limit_bytes": planStorageLimit(plan),
		"documents_limit":     planDocumentsLimit(plan),
		"embedding_provider":  embeddingProvider,
		"health":              healthStatus,
		"auth_enabled":        h.server.authEnabled,
		"uptime_seconds":      int64(time.Since(h.startedAt).Seconds()),
	})
}

// handleAPISearch 控制台内搜索，复用混合检索引擎。
func (h *AdminHandler) handleAPISearch(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query required"})
		return
	}
	topK := 10
	if v, err := strconv.Atoi(c.Query("top_k")); err == nil && v > 0 && v <= 50 {
		topK = v
	}
	mode := c.DefaultQuery("mode", "hybrid")
	if mode != "hybrid" && mode != "fts" && mode != "vector" {
		mode = "hybrid"
	}

	start := time.Now()
	results, err := h.server.engine.Search(c.Request.Context(), q, models.SearchOptions{TopK: topK, Mode: mode})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	items := make([]gin.H, 0, len(results))
	for _, r := range results {
		if r == nil || r.Chunk == nil {
			continue
		}
		path := r.FilePath
		if path == "" {
			path = r.Chunk.HeadingPath
		}
		items = append(items, gin.H{
			"chunk_id":     r.Chunk.ID,
			"document_id":  r.Chunk.DocumentID,
			"path":         path,
			"heading":      r.Chunk.HeadingPath,
			"score":        r.Score,
			"fts_score":    r.FTSScore,
			"vector_score": r.VectorScore,
			"line_start":   r.Chunk.LineStart,
			"line_end":     r.Chunk.LineEnd,
			"snippet":      snippet.Build(r.Chunk.DisplayText(), q, snippet.Options{Width: 320}),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"query":   q,
		"mode":    mode,
		"took_ms": time.Since(start).Milliseconds(),
		"count":   len(items),
		"results": items,
	})
}

// handleAPIDocs 最近索引的文档列表。
func (h *AdminHandler) handleAPIDocs(c *gin.Context) {
	limit := 20
	if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 && v <= 100 {
		limit = v
	}
	docs, err := h.server.storage.ListDocuments("", limit, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(docs))
	for _, d := range docs {
		items = append(items, gin.H{
			"id":          d.ID,
			"path":        d.Path,
			"title":       d.Title,
			"file_type":   d.FileType,
			"file_size":   d.FileSize,
			"chunk_count": d.ChunkCount,
			"status":      d.Status,
			"indexed_at":  d.IndexedAt,
		})
	}
	total, _ := h.server.storage.GetDocumentsCount("")
	c.JSON(http.StatusOK, gin.H{"total": total, "count": len(items), "documents": items})
}

// handleAPIHealth 返回健康检查结果。
func (h *AdminHandler) handleAPIHealth(c *gin.Context) {
	if h.server.health == nil {
		c.JSON(http.StatusOK, gin.H{"status": "unknown"})
		return
	}
	c.JSON(http.StatusOK, h.server.health.FullCheck())
}

// adminLocalOnly 限制管理面板仅本机访问，避免把未鉴权的写接口暴露到公网。
func adminLocalOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		if os.Getenv("CORTEX_ADMIN_ALLOW_REMOTE") == "1" {
			c.Next()
			return
		}
		host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		if err != nil {
			host = c.Request.RemoteAddr
		}
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "admin console is local-only; set CORTEX_ADMIN_ALLOW_REMOTE=1 to allow remote access",
			})
			return
		}
		c.Next()
	}
}

// planStorageLimit 返回各套餐的存储上限（0 表示不限），数据来自 license 包。
func planStorageLimit(plan string) int64 {
	l := license.Limits(license.Plan(plan))
	if l.StorageBytes < 0 {
		return 0
	}
	return l.StorageBytes
}

// planDocumentsLimit 返回各套餐的文档数上限（0 表示不限）。
func planDocumentsLimit(plan string) int64 {
	l := license.Limits(license.Plan(plan))
	if l.Documents < 0 {
		return 0
	}
	return l.Documents
}

func truncateRunes(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
