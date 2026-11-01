package auth

import (
	"errors"
	"os"
	"testing"

	"github.com/lh123aa/cortex/internal/models"
	"github.com/lh123aa/cortex/internal/storage"
)

// newRoleTestService 使用真实 SQLite（纯 Go，无 CGO）构建认证服务。
// 不用 MockStorage：它没有实现完整 storage.Storage 接口，
// 而 EnsureBootstrapAdmin / ListUsers 需要真实查询。
func newRoleTestService(t *testing.T) (*AuthService, storage.Storage) {
	t.Helper()

	f, err := os.CreateTemp("", "cortex-auth-*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	_ = f.Close()
	t.Cleanup(func() { _ = os.Remove(f.Name()) })

	st, err := storage.NewSQLiteStorage(f.Name())
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	return NewAuthServiceWithStorage(st), st
}

func TestRegisterDefaultsToEditor(t *testing.T) {
	svc, _ := newRoleTestService(t)

	u, err := svc.Register(&models.RegisterRequest{Username: "alice", Password: "S3cret!pass"})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if got := models.NormalizeRole(u.Role); got != models.RoleEditor {
		t.Fatalf("new user role = %q, want editor", got)
	}
}

func TestSetUserRole(t *testing.T) {
	svc, _ := newRoleTestService(t)
	if _, err := svc.Register(&models.RegisterRequest{Username: "alice", Password: "S3cret!pass"}); err != nil {
		t.Fatal(err)
	}

	u, err := svc.SetUserRole("alice", models.RoleAdmin)
	if err != nil {
		t.Fatalf("SetUserRole failed: %v", err)
	}
	if u.Role != models.RoleAdmin {
		t.Fatalf("role = %q, want admin", u.Role)
	}

	// 再查一次确认已持久化
	again, _ := svc.GetUserByUsername("alice")
	if again.Role != models.RoleAdmin {
		t.Fatalf("role not persisted, got %q", again.Role)
	}
}

func TestSetUserRoleRejectsInvalid(t *testing.T) {
	svc, _ := newRoleTestService(t)
	if _, err := svc.Register(&models.RegisterRequest{Username: "alice", Password: "S3cret!pass"}); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.SetUserRole("alice", "root"); !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("expected ErrInvalidRole, got %v", err)
	}
}

func TestSetUserRoleUnknownUser(t *testing.T) {
	svc, _ := newRoleTestService(t)
	if _, err := svc.SetUserRole("nobody", models.RoleViewer); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

// TestCannotDemoteLastAdmin 关键安全护栏：
// 系统只剩一个管理员时降级他，会导致再也无人能管理用户/密钥。
func TestCannotDemoteLastAdmin(t *testing.T) {
	svc, _ := newRoleTestService(t)
	if _, err := svc.Register(&models.RegisterRequest{Username: "root", Password: "S3cret!pass"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetUserRole("root", models.RoleAdmin); err != nil {
		t.Fatal(err)
	}

	_, err := svc.SetUserRole("root", models.RoleViewer)
	if !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("expected ErrLastAdmin, got %v", err)
	}

	// 角色必须保持不变
	u, _ := svc.GetUserByUsername("root")
	if models.NormalizeRole(u.Role) != models.RoleAdmin {
		t.Fatalf("admin role was changed despite last-admin guard: %q", u.Role)
	}
}

func TestCanDemoteAdminWhenAnotherExists(t *testing.T) {
	svc, _ := newRoleTestService(t)
	for _, name := range []string{"a", "b"} {
		if _, err := svc.Register(&models.RegisterRequest{Username: name, Password: "S3cret!pass"}); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.SetUserRole(name, models.RoleAdmin); err != nil {
			t.Fatal(err)
		}
	}

	// 有两个 admin 时，降级其中一个应成功
	if _, err := svc.SetUserRole("a", models.RoleViewer); err != nil {
		t.Fatalf("demoting a non-last admin should succeed: %v", err)
	}
}

func TestEnsureBootstrapAdminPromotesFirstUser(t *testing.T) {
	svc, _ := newRoleTestService(t)
	if _, err := svc.Register(&models.RegisterRequest{Username: "first", Password: "S3cret!pass"}); err != nil {
		t.Fatal(err)
	}

	promoted, err := svc.EnsureBootstrapAdmin()
	if err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}
	if promoted == nil {
		t.Fatal("expected first user to be promoted")
	}
	if promoted.Username != "first" || models.NormalizeRole(promoted.Role) != models.RoleAdmin {
		t.Fatalf("unexpected promotion: %+v", promoted)
	}
}

// TestEnsureBootstrapAdminIdempotent 已有管理员时不应再改动。
func TestEnsureBootstrapAdminIdempotent(t *testing.T) {
	svc, _ := newRoleTestService(t)
	if _, err := svc.Register(&models.RegisterRequest{Username: "already", Password: "S3cret!pass"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetUserRole("already", models.RoleAdmin); err != nil {
		t.Fatal(err)
	}

	promoted, err := svc.EnsureBootstrapAdmin()
	if err != nil {
		t.Fatal(err)
	}
	if promoted != nil {
		t.Fatalf("should be a no-op when an admin exists, got %+v", promoted)
	}
}

// TestEnsureBootstrapAdminNoUsers 空库不应报错。
func TestEnsureBootstrapAdminNoUsers(t *testing.T) {
	svc, _ := newRoleTestService(t)
	promoted, err := svc.EnsureBootstrapAdmin()
	if err != nil {
		t.Fatalf("bootstrap on empty db failed: %v", err)
	}
	if promoted != nil {
		t.Fatalf("nothing to promote, got %+v", promoted)
	}
}

func TestCreateUserWithRole(t *testing.T) {
	svc, _ := newRoleTestService(t)

	u, err := svc.CreateUserWithRole("bob", "S3cret!pass", models.RoleViewer)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if models.NormalizeRole(u.Role) != models.RoleViewer {
		t.Fatalf("role = %q, want viewer", u.Role)
	}

	if _, err := svc.CreateUserWithRole("carol", "S3cret!pass", "wizard"); !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("expected ErrInvalidRole, got %v", err)
	}
}

func TestActivateDeactivateUser(t *testing.T) {
	svc, _ := newRoleTestService(t)
	u, err := svc.Register(&models.RegisterRequest{Username: "dave", Password: "S3cret!pass"})
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.DeactivateUser(u.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := svc.GetUserByUsername(u.Username)
	if got.IsActive {
		t.Fatal("user should be deactivated")
	}

	if err := svc.ActivateUser(u.ID); err != nil {
		t.Fatal(err)
	}
	got, _ = svc.GetUserByUsername(u.Username)
	if !got.IsActive {
		t.Fatal("user should be re-activated")
	}
}
