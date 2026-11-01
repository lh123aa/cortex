package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// withTestKey 生成一对临时密钥，并替换包级公钥；测试结束后恢复。
func withTestKey(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	orig := PublicKeyHex
	PublicKeyHex = hex.EncodeToString(pub)
	t.Cleanup(func() { PublicKeyHex = orig })
	return priv
}

// withLicensePath 把许可文件重定向到临时目录，避免污染 ~/.cortex。
func withLicensePath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "license.key")
	orig := os.Getenv(EnvLicensePath)
	if err := os.Setenv(EnvLicensePath, path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if orig == "" {
			_ = os.Unsetenv(EnvLicensePath)
		} else {
			_ = os.Setenv(EnvLicensePath, orig)
		}
	})
	return path
}

func TestSignVerifyRoundtrip(t *testing.T) {
	priv := withTestKey(t)

	info := Info{
		Plan:      PlanPro,
		Email:     "buyer@example.com",
		Order:     "ord_123",
		Seats:     1,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(365 * 24 * time.Hour),
	}
	key, err := Sign(priv, info)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	got, err := Verify(key)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got.Plan != PlanPro || got.Email != "buyer@example.com" || got.Seats != 1 {
		t.Fatalf("verified info mismatch: %+v", got)
	}
	if got.DaysLeft() < 360 {
		t.Fatalf("DaysLeft = %d, want ~365", got.DaysLeft())
	}
}

func TestVerifyRejectsTamperedPayload(t *testing.T) {
	priv := withTestKey(t)
	key, err := Sign(priv, Info{Plan: PlanPro, Email: "a@b.c"})
	if err != nil {
		t.Fatal(err)
	}

	parts := strings.Split(key, ".")
	if len(parts) != 2 {
		t.Fatalf("unexpected key format: %q", key)
	}
	// 篡改 payload：在 base64 尾部追加一个字符（仍是合法 base64 字符）
	tampered := parts[0] + "A." + parts[1]
	if _, err := Verify(tampered); err == nil {
		t.Fatal("tampered payload must fail verification")
	}
}

func TestVerifyRejectsTamperedSignature(t *testing.T) {
	priv := withTestKey(t)
	key, err := Sign(priv, Info{Plan: PlanPro})
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(key, ".")
	sig := []byte(parts[1])
	if len(sig) == 0 {
		t.Fatal("empty signature")
	}
	// 翻转签名首字符
	if sig[0] == 'A' {
		sig[0] = 'B'
	} else {
		sig[0] = 'A'
	}
	if _, err := Verify(parts[0] + "." + string(sig)); err == nil {
		t.Fatal("tampered signature must fail verification")
	}
}

func TestVerifyRejectsExpired(t *testing.T) {
	priv := withTestKey(t)
	key, err := Sign(priv, Info{
		Plan:      PlanPro,
		IssuedAt:  time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-time.Hour),
	})
	if err != nil {
		t.Fatalf("Sign expired license: %v", err)
	}
	_, err = Verify(key)
	if err == nil {
		t.Fatal("expired license must be rejected")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "expired") {
		t.Fatalf("error should mention expiry, got: %v", err)
	}
}

func TestVerifyRejectsGarbage(t *testing.T) {
	withTestKey(t)
	for _, bad := range []string{"", "not-a-license", "a.b.c", "...."} {
		if _, err := Verify(bad); err == nil {
			t.Errorf("Verify(%q) should fail", bad)
		}
	}
}

func TestVerifyRejectsWrongKey(t *testing.T) {
	priv := withTestKey(t)
	key, err := Sign(priv, Info{Plan: PlanPro})
	if err != nil {
		t.Fatal(err)
	}
	// 换一把公钥（模拟非官方签发者）
	otherPub, _, _ := ed25519.GenerateKey(rand.Reader)
	orig := PublicKeyHex
	PublicKeyHex = hex.EncodeToString(otherPub)
	defer func() { PublicKeyHex = orig }()

	if _, err := Verify(key); err == nil {
		t.Fatal("license signed by another key must be rejected")
	}
}

func TestCurrentWithoutFile(t *testing.T) {
	withTestKey(t)
	withLicensePath(t)

	info, err := Current()
	if err != nil {
		t.Fatalf("Current without file should not error: %v", err)
	}
	if info != nil {
		t.Fatalf("expected nil license, got %+v", info)
	}
}

func TestActivateAndDeactivate(t *testing.T) {
	priv := withTestKey(t)
	path := withLicensePath(t)

	key, err := Sign(priv, Info{Plan: PlanTeam, Seats: 3, Email: "team@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Activate(key); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("license file not written: %v", err)
	}

	info, err := Current()
	if err != nil || info == nil {
		t.Fatalf("Current after activate = %+v, %v", info, err)
	}
	if info.Plan != PlanTeam {
		t.Fatalf("plan = %s, want team", info.Plan)
	}

	if err := Deactivate(); err != nil {
		t.Fatalf("Deactivate: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("license file should be removed")
	}
}

func TestActivateRejectsBadKey(t *testing.T) {
	withTestKey(t)
	path := withLicensePath(t)

	if _, err := Activate("garbage-key"); err == nil {
		t.Fatal("Activate should reject invalid key")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("invalid key must not be persisted")
	}
}

func TestPathHonoursEnv(t *testing.T) {
	path := withLicensePath(t)
	if got := Path(); got != path {
		t.Fatalf("Path() = %q, want %q", got, path)
	}
}

func TestLimitsTable(t *testing.T) {
	cases := []struct {
		plan Plan
		docs int64
	}{
		{PlanFree, 5_000},
		{PlanPro, 500_000},
		{PlanTeam, 1_000_000},
		{PlanEnterprise, -1},
		{Plan("unknown"), 5_000}, // 未知按 Free
	}
	for _, c := range cases {
		if got := Limits(c.plan).Documents; got != c.docs {
			t.Errorf("Limits(%s).Documents = %d, want %d", c.plan, got, c.docs)
		}
	}
}

func TestCheckQuota(t *testing.T) {
	free := &Info{Plan: PlanFree}
	if err := CheckQuota(free, 100, 1024); err != nil {
		t.Fatalf("under free limits should pass: %v", err)
	}
	err := CheckQuota(free, 5_000, 0)
	if err == nil {
		t.Fatal("reaching document limit should fail")
	}
	if !IsQuotaExceeded(err) {
		t.Fatalf("expected quota error, got %v", err)
	}

	ent := &Info{Plan: PlanEnterprise}
	if err := CheckQuota(ent, 10_000_000, 1<<50); err != nil {
		t.Fatalf("enterprise should be unlimited: %v", err)
	}

	if err := CheckQuota(nil, 10_000, 0); err == nil {
		t.Fatal("nil license should be treated as free and exceed limit")
	}
}
