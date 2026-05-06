package chunker

import (
	"testing"
)

func TestContainsChinese(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"Hello World", false},
		{"Go语言", true},
		{"并发编程", true},
		{"English only", false},
		{"混合text", true},
	}
	for _, tt := range tests {
		got := containsChinese(tt.input)
		if got != tt.want {
			t.Errorf("containsChinese(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestSegmentChinese(t *testing.T) {
	tests := []struct {
		input string
		want  string // empty means check contains, not exact
	}{
		{"Hello World", "Hello World"},
		{"", ""},
		{"a", "a"},
		{"Go语言并发", ""}, // 只验证不崩溃
	}
	for _, tt := range tests {
		got := SegmentChinese(tt.input)
		if tt.want != "" && got != tt.want {
			t.Errorf("SegmentChinese(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSegmentChinese_Roundtrip(t *testing.T) {
	// 验证中文文本被分词后长度合理
	text := "Go语言并发编程使用goroutine和channel"
	seg := SegmentChinese(text)
	if len(seg) < len(text) {
		t.Errorf("Segmented text should not be shorter: %q -> %q", text, seg)
	}
	if !containsChinese(seg) {
		t.Errorf("Segmented text should still contain Chinese: %q", seg)
	}
}

func TestSegmentChinese_NonChinese(t *testing.T) {
	text := "The quick brown fox"
	seg := SegmentChinese(text)
	if seg != text {
		t.Errorf("Non-Chinese text should not change: %q -> %q", text, seg)
	}
}

func TestSegmentChinese_Mixed(t *testing.T) {
	text := "学习Go语言并发编程"
	seg := SegmentChinese(text)
	if seg == text {
		t.Errorf("Chinese text should be segmented: %q -> %q", text, seg)
	}
}

func TestOptimizeForChineseSearch(t *testing.T) {
	text := "Go语言并发编程"
	opt := OptimizeForChineseSearch(text)
	if len(opt) < len(text) {
		t.Errorf("Optimized text should not be shorter: %q", opt)
	}
}
