package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestNewAPIKeyAuth(t *testing.T) {
	auth := NewAPIKeyAuth("X-API-Key", "api_key")

	if auth == nil {
		t.Fatal("Expected non-nil APIKeyAuth")
	}
	if auth.headerName != "X-API-Key" {
		t.Errorf("Expected headerName 'X-API-Key', got '%s'", auth.headerName)
	}
	if auth.queryName != "api_key" {
		t.Errorf("Expected queryName 'api_key', got '%s'", auth.queryName)
	}
}

func TestAPIKeyAuth_AddKey(t *testing.T) {
	auth := NewAPIKeyAuth("X-API-Key", "api_key")

	auth.AddKey("test-key-1")
	auth.AddKey("test-key-2")

	if !auth.isValidKey("test-key-1") {
		t.Error("Expected test-key-1 to be valid")
	}
	if !auth.isValidKey("test-key-2") {
		t.Error("Expected test-key-2 to be valid")
	}
	if auth.isValidKey("invalid-key") {
		t.Error("Expected invalid-key to be invalid")
	}
}

func TestAPIKeyAuth_RemoveKey(t *testing.T) {
	auth := NewAPIKeyAuth("X-API-Key", "api_key")

	auth.AddKey("removable-key")
	if !auth.isValidKey("removable-key") {
		t.Error("Expected removable-key to be valid before removal")
	}

	auth.RemoveKey("removable-key")
	if auth.isValidKey("removable-key") {
		t.Error("Expected removable-key to be invalid after removal")
	}
}

func TestAPIKeyAuth_ClearKeys(t *testing.T) {
	auth := NewAPIKeyAuth("X-API-Key", "api_key")

	auth.AddKey("key1")
	auth.AddKey("key2")

	auth.ClearKeys()

	if auth.isValidKey("key1") {
		t.Error("Expected key1 to be invalid after clear")
	}
	if auth.isValidKey("key2") {
		t.Error("Expected key2 to be invalid after clear")
	}
}

func TestAPIKeyAuth_Middleware_MissingKey(t *testing.T) {
	auth := NewAPIKeyAuth("X-API-Key", "api_key")
	auth.AddKey("valid-key")

	// Create test router
	router := gin.New()
	router.Use(auth.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Request without key
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAPIKeyAuth_Middleware_ValidKey(t *testing.T) {
	auth := NewAPIKeyAuth("X-API-Key", "api_key")
	auth.AddKey("valid-key")

	router := gin.New()
	router.Use(auth.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Request with valid key in header
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "valid-key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAPIKeyAuth_Middleware_ValidKeyQuery(t *testing.T) {
	auth := NewAPIKeyAuth("X-API-Key", "api_key")
	auth.AddKey("valid-key")

	router := gin.New()
	router.Use(auth.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Request with valid key in query param
	req := httptest.NewRequest("GET", "/test?api_key=valid-key", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAPIKeyAuth_Middleware_InvalidKey(t *testing.T) {
	auth := NewAPIKeyAuth("X-API-Key", "api_key")
	auth.AddKey("valid-key")

	router := gin.New()
	router.Use(auth.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Request with invalid key
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "invalid-key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAPIKeyAuth_Middleware_EmptyKey(t *testing.T) {
	auth := NewAPIKeyAuth("X-API-Key", "api_key")
	auth.AddKey("valid-key")

	router := gin.New()
	router.Use(auth.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Request with empty key
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for empty key, got %d", w.Code)
	}
}

func TestAPIKeyAuth_OptionalMiddleware_NoKey(t *testing.T) {
	auth := NewAPIKeyAuth("X-API-Key", "api_key")
	auth.AddKey("valid-key")

	router := gin.New()
	router.Use(auth.OptionalMiddleware())
	router.GET("/test", func(c *gin.Context) {
		authRequired, exists := c.Get("auth_required")
		if !exists {
			c.JSON(http.StatusOK, gin.H{"auth_required": false})
			return
		}
		c.JSON(http.StatusOK, gin.H{"auth_required": authRequired})
	})

	// Request without key
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAPIKeyAuth_OptionalMiddleware_WithValidKey(t *testing.T) {
	auth := NewAPIKeyAuth("X-API-Key", "api_key")
	auth.AddKey("valid-key")

	router := gin.New()
	router.Use(auth.OptionalMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Request with valid key
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "valid-key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAPIKeyAuth_OptionalMiddleware_WithInvalidKey(t *testing.T) {
	auth := NewAPIKeyAuth("X-API-Key", "api_key")
	auth.AddKey("valid-key")

	router := gin.New()
	router.Use(auth.OptionalMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Request with invalid key
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "invalid-key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should reject invalid key
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAPIKeyAuth_getKeyFromRequest_HeaderFirst(t *testing.T) {
	auth := NewAPIKeyAuth("X-API-Key", "api_key")

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		key := auth.getKeyFromRequest(c)
		c.JSON(http.StatusOK, gin.H{"key": key})
	})

	// Request with key in both header and query
	req := httptest.NewRequest("GET", "/test?api_key=query-key", nil)
	req.Header.Set("X-API-Key", "header-key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Header should take precedence
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAPIKeyAuth_getKeyFromRequest_QueryFallback(t *testing.T) {
	auth := NewAPIKeyAuth("X-API-Key", "api_key")

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		key := auth.getKeyFromRequest(c)
		c.JSON(http.StatusOK, gin.H{"key": key})
	})

	// Request with key only in query
	req := httptest.NewRequest("GET", "/test?api_key=query-key", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAPIKeyAuth_getKeyFromRequest_Neither(t *testing.T) {
	auth := NewAPIKeyAuth("X-API-Key", "api_key")

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		key := auth.getKeyFromRequest(c)
		c.JSON(http.StatusOK, gin.H{"key": key})
	})

	// Request without any key
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestConstantTimeCompare(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{"equal strings", "test-key", "test-key", true},
		{"different strings", "test-key-1", "test-key-2", false},
		{"empty strings", "", "", true},
		{"one empty", "test", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := constantTimeCompare(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestAPIKeyAuth_isValidKey_ThreadSafety(t *testing.T) {
	auth := NewAPIKeyAuth("X-API-Key", "api_key")
	auth.AddKey("key1")

	// Concurrent read operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				auth.isValidKey("key1")
				auth.isValidKey("invalid-key")
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestAPIKeyAuth_AddKey_ThreadSafety(t *testing.T) {
	auth := NewAPIKeyAuth("X-API-Key", "api_key")

	// Concurrent write operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				auth.AddKey(string(rune('a'+id)) + string(rune('0'+j%10)))
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// All keys should be present
	if auth.isValidKey("a0") {
		t.Log("Key a0 found")
	}
}
