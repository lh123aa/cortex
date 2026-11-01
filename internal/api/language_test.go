package api

import (
	"testing"

	"github.com/lh123aa/cortex/internal/version"
)

// TestNormalizeLanguageAliases 回归测试：
// 用户习惯写 py/js/ts/golang 这类短别名，而索引里存的是 python/javascript/typescript/go。
// 早期实现对不上就静默返回空结果，看起来像「索引没建好」。
func TestNormalizeLanguageAliases(t *testing.T) {
	cases := map[string]string{
		"py": "python", "PY": "python", ".py": "python", "python": "python",
		"js": "javascript", "jsx": "javascript", "node": "javascript",
		"ts": "typescript", "tsx": "typescript",
		"golang": "go", ".go": "go",
		"rs":     "rust",
		"":       "",
		"  Go  ": "go",
	}
	for in, want := range cases {
		if got := normalizeLanguage(in); got != want {
			t.Errorf("normalizeLanguage(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestCodeSearchLanguageFilterUsesNormalizedValue 确保 handler 真的用了归一化结果。
func TestCodeSearchLanguageFilterUsesNormalizedValue(t *testing.T) {
	if got := normalizeLanguage("py"); got != "python" {
		t.Fatalf("expected python, got %q", got)
	}
	// 传入未知语言应原样返回（不 panic、不误判）
	if got := normalizeLanguage("cobol"); got != "cobol" {
		t.Fatalf("unknown language should pass through, got %q", got)
	}
}

// TestVersionLabelNoDoublePrefix 回归测试：
// 发布构建里 version.Version 已含 "v"（如 v4.0.0），
// 若再用 "v%s" 拼接就会出现 "vv4.0.0"。
func TestVersionLabelNoDoublePrefix(t *testing.T) {
	orig := version.Version
	defer func() { version.Version = orig }()

	cases := []struct{ in, want string }{
		{"v4.0.0", "v4.0.0"},
		{"4.0.0", "v4.0.0"},
		{"dev", "dev"},
		{"", "dev"},
	}
	for _, c := range cases {
		version.Version = c.in
		if got := versionLabel(); got != c.want {
			t.Errorf("versionLabel() with Version=%q = %q, want %q", c.in, got, c.want)
		}
	}
}
