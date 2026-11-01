package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/lh123aa/cortex/internal/auth"
	"github.com/lh123aa/cortex/internal/models"
	"github.com/lh123aa/cortex/internal/search"
	"github.com/lh123aa/cortex/internal/storage"
	"go.uber.org/zap"
)

// rbacTestEnv 构建一个启用了认证的 RESTServer，并准备好三种角色的 token。
type rbacTestEnv struct {
	srv    *RESTServer
	tokens map[string]string // role -> bearer token
}

func setupRBACTest(t *testing.T) (*rbacTestEnv, func()) {
	t.Helper()

	// 用独立临时文件库：file::memory:?cache=shared 会在同一进程内被所有测试共享，
	// 造成用户/ token 串扰，测试之间不隔离。
	f, err := os.CreateTemp("", "cortex-rbac-*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	_ = f.Close()

	st, err := storage.NewSQLiteStorage(f.Name())
	if err != nil {
		_ = os.Remove(f.Name())
		t.Fatalf("create storage: %v", err)
	}
	se := search.NewHybridSearchEngine(st, nil)
	authSvc := auth.NewAuthServiceWithStorage(st)
	srv := NewRESTServerWithAuth(se, st, nil, zap.NewNop(), authSvc)

	env := &rbacTestEnv{srv: srv, tokens: map[string]string{}}
	roles := []string{models.RoleViewer, models.RoleEditor, models.RoleAdmin}
	for _, role := range roles {
		u, err := authSvc.CreateUserWithRole("user-"+role, "S3cret!pass", role)
		if err != nil {
			t.Fatalf("create %s user: %v", role, err)
		}
		tok, err := authSvc.Login(&models.LoginRequest{Username: u.Username, Password: "S3cret!pass"})
		if err != nil {
			t.Fatalf("login %s: %v", role, err)
		}
		// 自检：登录拿到的 token 必须能立刻换回用户。
		// 这一步用于区分「token 存储坏了」和「中间件解析坏了」。
		if got, err := authSvc.GetUserByToken(tok.Token); err != nil || got == nil {
			t.Fatalf("token for role %s cannot be resolved (err=%v)", role, err)
		}
		env.tokens[role] = tok.Token
	}

	return env, func() {
		_ = st.Close()
		_ = os.Remove(f.Name())
	}
}

// do 发起一个带指定角色 token 的请求。
func (e *rbacTestEnv) do(t *testing.T, role, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var buf *bytes.Buffer
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		buf = bytes.NewBuffer(b)
	} else {
		buf = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, buf)
	req.Header.Set("Content-Type", "application/json")
	tok, ok := e.tokens[role]
	if !ok {
		t.Fatalf("no token for role %q", role)
	}
	req.Header.Set("Authorization", "Bearer "+tok)

	w := httptest.NewRecorder()
	e.srv.Router().ServeHTTP(w, req)
	return w
}

// TestViewerCannotWriteMemory viewer 只能读，写记忆必须 403。
func TestViewerCannotWriteMemory(t *testing.T) {
	env, cleanup := setupRBACTest(t)
	defer cleanup()

	w := env.do(t, models.RoleViewer, http.MethodPost, "/v1/memory", map[string]string{"content": "viewer 不应能写入"})
	if w.Code != http.StatusForbidden {
		t.Fatalf("viewer POST /v1/memory = %d, want 403 (body: %s)", w.Code, w.Body.String())
	}
}

// TestEditorCanWriteMemory editor 可以写。
func TestEditorCanWriteMemory(t *testing.T) {
	env, cleanup := setupRBACTest(t)
	defer cleanup()

	w := env.do(t, models.RoleEditor, http.MethodPost, "/v1/memory", map[string]string{
		"content": "editor 可以写入这条记忆内容，长度需要足够触发分块处理。",
	})
	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("editor POST /v1/memory = %d, want 2xx (body: %s)", w.Code, w.Body.String())
	}
}

// TestAdminCanWriteMemory admin 可以写。
func TestAdminCanWriteMemory(t *testing.T) {
	env, cleanup := setupRBACTest(t)
	defer cleanup()

	w := env.do(t, models.RoleAdmin, http.MethodPost, "/v1/memory", map[string]string{
		"content": "admin 可以写入这条记忆内容，长度需要足够触发分块处理。",
	})
	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("admin POST /v1/memory = %d, want 2xx (body: %s)", w.Code, w.Body.String())
	}
}

// TestViewerCanRead viewer 读接口应当放行。
func TestViewerCanRead(t *testing.T) {
	env, cleanup := setupRBACTest(t)
	defer cleanup()

	w := env.do(t, models.RoleViewer, http.MethodGet, "/v1/search?q=test", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("viewer GET /v1/search = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
}

// TestViewerCannotDeleteMemory viewer 不能删。
func TestViewerCannotDeleteMemory(t *testing.T) {
	env, cleanup := setupRBACTest(t)
	defer cleanup()

	w := env.do(t, models.RoleViewer, http.MethodDelete, "/v1/memory/abc123", nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("viewer DELETE /v1/memory/:id = %d, want 403 (body: %s)", w.Code, w.Body.String())
	}
}

// TestAdminKeysRequiresAdmin 密钥列表只有 admin 能看。
func TestAdminKeysRequiresAdmin(t *testing.T) {
	env, cleanup := setupRBACTest(t)
	defer cleanup()

	for _, role := range []string{models.RoleViewer, models.RoleEditor} {
		w := env.do(t, role, http.MethodGet, "/admin/keys", nil)
		if w.Code != http.StatusForbidden {
			t.Errorf("%s GET /admin/keys = %d, want 403 (body: %s)", role, w.Code, w.Body.String())
		}
	}

	w := env.do(t, models.RoleAdmin, http.MethodGet, "/admin/keys", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("admin GET /admin/keys = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
}

// TestNoTokenRejected 无 token 必须 401。
func TestNoTokenRejected(t *testing.T) {
	env, cleanup := setupRBACTest(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=test", nil)
	w := httptest.NewRecorder()
	env.srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no-token request = %d, want 401", w.Code)
	}
}

// TestAuthDisabledAllowsWrites 关闭认证时（单机本地场景）不限制写入。
func TestAuthDisabledAllowsWrites(t *testing.T) {
	st, err := storage.NewSQLiteStorage("file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	se := search.NewHybridSearchEngine(st, nil)
	srv := NewRESTServer(se, st, nil, zap.NewNop())

	body := bytes.NewBufferString(`{"content":"单机模式下无需认证即可写入记忆内容，长度足够分块。"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/memory", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("auth-disabled POST /v1/memory = %d, want 2xx (body: %s)", w.Code, w.Body.String())
	}
}
