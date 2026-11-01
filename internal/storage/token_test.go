package storage

import (
	"os"
	"testing"
	"time"

	"github.com/lh123aa/cortex/internal/models"
)

func newTokenTestDB(t *testing.T) (*SQLiteStorage, func()) {
	t.Helper()
	f, err := os.CreateTemp("", "cortex-token-*.db")
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	st, err := NewSQLiteStorage(f.Name())
	if err != nil {
		_ = os.Remove(f.Name())
		t.Fatal(err)
	}
	return st, func() {
		_ = st.Close()
		_ = os.Remove(f.Name())
	}
}

// seedTokenUser 插入一个用户：auth_tokens.user_id 有外键约束，必须先生成用户。
func seedTokenUser(t *testing.T, st *SQLiteStorage) {
	t.Helper()
	u := &models.User{
		ID: "u1", Username: "alice", PasswordHash: "x",
		Role: models.RoleEditor, Tier: "free", IsActive: true,
	}
	if err := st.SaveUser(u); err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

func TestSaveAndGetToken(t *testing.T) {
	st, cleanup := newTokenTestDB(t)
	defer cleanup()
	seedTokenUser(t, st)

	tok := &models.AuthToken{
		Token:     "test-token-abc",
		UserID:    "u1",
		Username:  "alice",
		ExpiresIn: 3600,
	}
	if err := st.SaveToken(tok); err != nil {
		t.Fatalf("SaveToken failed: %v", err)
	}

	got, err := st.GetToken("test-token-abc")
	if err != nil {
		t.Fatalf("GetToken error: %v", err)
	}
	if got == nil {
		t.Fatal("token not found immediately after save (type mismatch between integer and text timestamps?)")
	}
	if got.UserID != "u1" {
		t.Fatalf("UserID = %q, want u1", got.UserID)
	}
	if got.ExpiresIn <= 0 || got.ExpiresIn > 3600 {
		t.Fatalf("ExpiresIn = %d, want (0,3600]", got.ExpiresIn)
	}
}

func TestGetExpiredTokenReturnsNil(t *testing.T) {
	st, cleanup := newTokenTestDB(t)
	defer cleanup()
	seedTokenUser(t, st)

	// ExpiresIn 为负数 => 已过期
	tok := &models.AuthToken{Token: "expired", UserID: "u1", Username: "a", ExpiresIn: -60}
	if err := st.SaveToken(tok); err != nil {
		t.Fatal(err)
	}

	got, err := st.GetToken("expired")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("expired token should not be returned")
	}
}

func TestGetUnknownToken(t *testing.T) {
	st, cleanup := newTokenTestDB(t)
	defer cleanup()

	got, err := st.GetToken("nope")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("unknown token should return nil")
	}
}

func TestDeleteExpiredTokens(t *testing.T) {
	st, cleanup := newTokenTestDB(t)
	defer cleanup()
	seedTokenUser(t, st)

	_ = st.SaveToken(&models.AuthToken{Token: "live", UserID: "u1", ExpiresIn: 3600})
	_ = st.SaveToken(&models.AuthToken{Token: "dead", UserID: "u1", ExpiresIn: -60})

	n, err := st.DeleteExpiredTokens()
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("deleted %d tokens, want 1", n)
	}
	if got, _ := st.GetToken("live"); got == nil {
		t.Fatal("live token should survive cleanup")
	}
}

var _ = time.Second
