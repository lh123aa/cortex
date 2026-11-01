package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lh123aa/cortex/internal/search"
	"github.com/lh123aa/cortex/internal/storage"
	"go.uber.org/zap"
)

// setupAdminTest 构造一个最小可用的 RESTServer（内存 SQLite + 无 embedding）。
func setupAdminTest(t *testing.T) (*RESTServer, func()) {
	t.Helper()

	st, err := storage.NewSQLiteStorage("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	se := search.NewHybridSearchEngine(st, nil)
	srv := NewRESTServer(se, st, nil, zap.NewNop())
	return srv, func() { _ = st.Close() }
}

// loopbackRequest 设置本机 RemoteAddr，绕过 admin 的本机访问限制。
func loopbackRequest(method, target string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	req.RemoteAddr = "127.0.0.1:54321"
	return req
}

func TestAdminPageRenders(t *testing.T) {
	srv, cleanup := setupAdminTest(t)
	defer cleanup()

	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, loopbackRequest(http.MethodGet, "/admin"))

	if w.Code != http.StatusOK {
		t.Fatalf("/admin status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if len(body) < 2000 {
		t.Fatalf("/admin body length = %d, want > 2000 (empty template regression)", len(body))
	}
	if !strings.Contains(body, "Cortex Admin") {
		t.Fatalf("/admin body does not contain dashboard markup")
	}
	if !strings.Contains(body, "/admin/api/status") {
		t.Fatalf("/admin body does not reference status endpoint")
	}
}

func TestAdminStatusAPI(t *testing.T) {
	srv, cleanup := setupAdminTest(t)
	defer cleanup()

	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, loopbackRequest(http.MethodGet, "/admin/api/status"))

	if w.Code != http.StatusOK {
		t.Fatalf("/admin/api/status = %d, want 200", w.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	for _, key := range []string{
		"version", "commit", "plan", "documents", "chunks", "vectors",
		"storage_used_bytes", "storage_limit_bytes", "health", "uptime_seconds",
	} {
		if _, ok := payload[key]; !ok {
			t.Errorf("status payload missing key %q", key)
		}
	}
	if plan, _ := payload["plan"].(string); plan == "" {
		t.Errorf("plan should never be empty, got %v", payload["plan"])
	}
	if limit, _ := payload["storage_limit_bytes"].(float64); limit <= 0 {
		t.Errorf("free plan storage limit should be positive, got %v", payload["storage_limit_bytes"])
	}
}

func TestAdminDocsAPI(t *testing.T) {
	srv, cleanup := setupAdminTest(t)
	defer cleanup()

	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, loopbackRequest(http.MethodGet, "/admin/api/docs?limit=5"))
	if w.Code != http.StatusOK {
		t.Fatalf("/admin/api/docs = %d, want 200", w.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := payload["documents"]; !ok {
		t.Errorf("docs payload missing documents key: %v", payload)
	}
}

func TestAdminSearchRequiresQuery(t *testing.T) {
	srv, cleanup := setupAdminTest(t)
	defer cleanup()

	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, loopbackRequest(http.MethodGet, "/admin/api/search"))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("/admin/api/search without q = %d, want 400", w.Code)
	}
}

func TestAdminRejectsNonLoopback(t *testing.T) {
	srv, cleanup := setupAdminTest(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.RemoteAddr = "203.0.113.9:40000"
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("remote /admin = %d, want 403", w.Code)
	}
}
