package snippet

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestBuildShortTextUnchanged(t *testing.T) {
	text := "短文本不需要截断。"
	got := Build(text, "文本", Options{})
	if got != text {
		t.Fatalf("short text should be unchanged, got %q", got)
	}
}

func TestBuildFocusesOnHitNotPrefix(t *testing.T) {
	// 命中词在第 600 字符之后，旧实现会取开头 300 字 → 用户看不到命中内容
	prefix := strings.Repeat("无关前缀内容。", 100) // ~700 runes
	text := prefix + "这里才出现目标关键词"
	got := Build(text, "目标关键词", Options{})
	if !strings.Contains(got, "目标关键词") {
		t.Fatalf("snippet should contain the matched term, got %q", got)
	}
	if strings.Contains(got, "无关前缀内容。无关前缀内容。无关前缀内容。") && !strings.Contains(got, "目标") {
		t.Fatalf("snippet still shows only the prefix")
	}
}

func TestBuildRespectsWidth(t *testing.T) {
	text := strings.Repeat("一二三四五六七八九十", 100)
	got := Build(text, "", Options{Width: 50})
	if n := utf8.RuneCountInString(got); n > 60 {
		t.Fatalf("snippet too long: %d runes", n)
	}
}

func TestBuildAddsEllipsisWhenTruncated(t *testing.T) {
	text := strings.Repeat("内容", 300)
	got := Build(text, "", Options{Width: 100, Ellipsis: "…"})
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("truncated snippet should end with ellipsis, got %q", got)
	}
}

func TestBuildNoEllipsisWhenWholeTextFits(t *testing.T) {
	text := "短文本"
	got := Build(text, "", Options{Width: 100})
	if strings.Contains(got, "…") {
		t.Fatalf("should not add ellipsis, got %q", got)
	}
}

func TestBuildWindowsMidText(t *testing.T) {
	left := strings.Repeat("左", 500)
	right := strings.Repeat("右", 500)
	text := left + "命中词" + right
	got := Build(text, "命中词", Options{Width: 100, Context: 30})
	if !strings.Contains(got, "命中词") {
		t.Fatalf("mid-text match lost: %q", got)
	}
	if !strings.HasPrefix(got, "…") {
		t.Fatalf("should indicate leading truncation with ellipsis, got %q", got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("should indicate trailing truncation with ellipsis, got %q", got)
	}
}

func TestBuildCollapsesWhitespace(t *testing.T) {
	text := "第一行\n\n\n第二行\t\t第三行"
	got := Build(text, "", Options{})
	if strings.ContainsAny(got, "\t") || strings.Contains(got, "\n\n") {
		t.Fatalf("whitespace should be collapsed, got %q", got)
	}
}

func TestBuildPreservesNewlinesForCode(t *testing.T) {
	text := "line1\n\nline2"
	got := Build(text, "", Options{PreserveNewlines: true})
	if !strings.Contains(got, "\n") {
		t.Fatalf("newlines should be preserved in code mode, got %q", got)
	}
}

func TestBuildEmptyInput(t *testing.T) {
	if got := Build("", "query", Options{}); got != "" {
		t.Fatalf("empty input should return empty, got %q", got)
	}
}

func TestQueryTermsExtractsEnglishAndCJK(t *testing.T) {
	terms := QueryTerms("License 激活")
	hasEnglish := false
	for _, term := range terms {
		if strings.EqualFold(term, "License") {
			hasEnglish = true
		}
	}
	if !hasEnglish {
		t.Fatalf("expected English term to be kept, got %v", terms)
	}
	if len(terms) < 2 {
		t.Fatalf("expected multiple terms, got %v", terms)
	}
}

func TestQueryTermsIgnoresSingleChar(t *testing.T) {
	for _, term := range QueryTerms("a") {
		if term == "a" {
			t.Fatalf("single char term should be ignored, got %v", QueryTerms("a"))
		}
	}
}

func TestBuildNeverExceedsWidthMuch(t *testing.T) {
	for _, width := range []int{50, 100, 300} {
		text := strings.Repeat("x", 5000)
		got := Build(text, "x", Options{Width: width})
		if n := utf8.RuneCountInString(got); n > width+3 {
			t.Fatalf("width=%d produced %d runes", width, n)
		}
	}
}

func TestBuildCJKCenteredWindow(t *testing.T) {
	text := strings.Repeat("前文", 200) + "关键的检索结果" + strings.Repeat("后文", 200)
	got := Build(text, "检索结果", Options{Width: 120})
	if !strings.Contains(got, "检索结果") {
		t.Fatalf("CJK term not found in snippet: %q", got)
	}
}
