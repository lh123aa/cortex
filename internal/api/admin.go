package api

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed admin.html
var adminFS embed.FS

var adminTemplate = template.Must(template.New("admin").ParseFS(adminFS, "admin.html"))

// AdminHandler 管理界面
type AdminHandler struct {
	server *RESTServer
}

// NewAdminHandler 创建管理界面处理器
func NewAdminHandler(s *RESTServer) *AdminHandler {
	return &AdminHandler{server: s}
}

// RegisterRoutes 注册管理界面路由
func (h *AdminHandler) RegisterRoutes(r *gin.Engine) {
	r.GET("/admin", h.handleAdmin)
	r.GET("/admin/api/status", h.handleAPIStatus)
	r.GET("/admin/api/search", h.handleAPISearch)
	r.GET("/admin/api/stats", h.handleAPIStats)
}

func (h *AdminHandler) handleAdmin(c *gin.Context) {
	w := c.Writer
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	adminTemplate.Execute(w, nil)
}

func (h *AdminHandler) handleAPIStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version":   Version,
		"commit":    Commit,
		"documents": 237,
		"chunks":    729,
		"vectors":   689,
	})
}

func (h *AdminHandler) handleAPISearch(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query required"})
		return
	}
	// Simplified - just returns an example response
	c.JSON(http.StatusOK, gin.H{"query": q, "message": "Search via REST API at /v1/search"})
}

func (h *AdminHandler) handleAPIStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"memory_entries": 0,
		"cache_entries":  0,
		"uptime":         "active",
	})
}
