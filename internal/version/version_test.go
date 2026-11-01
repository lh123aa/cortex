package version

import (
	"strings"
	"testing"
)

// withInjectedVersion 在测试期间临时覆盖注入变量，结束后恢复。
func withInjectedVersion(t *testing.T, ver, commit, date string) {
	t.Helper()
	origVersion, origCommit, origDate := Version, Commit, Date
	t.Cleanup(func() {
		Version, Commit, Date = origVersion, origCommit, origDate
	})
	Version, Commit, Date = ver, commit, date
}

func TestFullContainsVersionAndCommit(t *testing.T) {
	withInjectedVersion(t, "v4.0.0", "abc1234", "2026-10-22T10:00:00Z")

	got := Full()
	for _, want := range []string{"v4.0.0", "abc1234", "2026-10-22T10:00:00Z"} {
		if !strings.Contains(got, want) {
			t.Errorf("Full() = %q, 期望包含 %q", got, want)
		}
	}
}

func TestFullDefaults(t *testing.T) {
	withInjectedVersion(t, "dev", "none", "unknown")

	got := Full()
	for _, want := range []string{"dev", "none", "unknown"} {
		if !strings.Contains(got, want) {
			t.Errorf("Full() = %q, 期望包含 %q", got, want)
		}
	}
}

func TestDefaultVariablesAreNonEmpty(t *testing.T) {
	// 未注入时也必须有可展示的值，避免发布产物出现空版本号。
	for name, value := range map[string]string{
		"Version": Version,
		"Commit":  Commit,
		"Date":    Date,
	} {
		if strings.TrimSpace(value) == "" {
			t.Errorf("%s 不应为空", name)
		}
	}
	if Full() == "" {
		t.Error("Full() 不应为空")
	}
}
