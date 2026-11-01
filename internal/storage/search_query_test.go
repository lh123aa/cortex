package storage

import "testing"

func TestExpandChineseQuery(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"pure english unchanged", "License", "License"},
		{"english multi word", "hybrid search", "hybrid search"},
		{"pure chinese single chars", "激活", "激 活"},
		{"chinese phrase", "配额与套餐", "配 额 与 套 餐"},
		{"mixed keeps english word intact", "License 激活", "License 激 活"},
		{"mixed reversed order", "激活 License", "激 活 License"},
		{"mixed no space", "License激活", "License 激 活"},
		{"punctuation becomes boundary", "License，激活", "License 激 活"},
		{"numbers preserved", "v4 版本", "v4 版 本"},
		{"empty stays empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandChineseQuery(tt.input)
			if got != tt.want {
				t.Errorf("expandChineseQuery(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestExpandChineseQueryKeepsEnglishTokensIntact 回归测试：
// 早期实现会把 "License 激活" 拆成 "L i c e n s e 激 活"，
// FTS5 中不存在单字母 token "L"，导致混合中英查询永远 0 命中。
func TestExpandChineseQueryKeepsEnglishTokensIntact(t *testing.T) {
	got := expandChineseQuery("License 激活")
	if got == "L i c e n s e 激 活" {
		t.Fatalf("english token was split into single letters: %q", got)
	}
	// 必须保留完整的 "License" 词项
	found := false
	for _, tok := range splitTokens(got) {
		if tok == "License" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected token %q in %q", "License", got)
	}
}

func splitTokens(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ' ' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
