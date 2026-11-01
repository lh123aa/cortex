package storage

import (
	"testing"

	"github.com/lh123aa/cortex/internal/models"
)

// TestTokenWithSharedMemoryDSN 复现 API 测试使用的连接串，
// 确认 token 在 file::memory:?cache=shared 下也能存取。
func TestTokenWithSharedMemoryDSN(t *testing.T) {
	st, err := NewSQLiteStorage("file::memory:?cache=shared&_tokentest=1")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	seedTokenUser(t, st)

	tok := &models.AuthToken{Token: "mem-token", UserID: "u1", Username: "alice", ExpiresIn: 3600}
	if err := st.SaveToken(tok); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	got, err := st.GetToken("mem-token")
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}
	if got == nil {
		// 打印实际存储值，帮助定位类型/时区问题
		var raw any
		_ = st.db.QueryRow(`SELECT expires_at FROM auth_tokens WHERE token = ?`, "mem-token").Scan(&raw)
		t.Fatalf("token not found; raw expires_at = %#v", raw)
	}
}
