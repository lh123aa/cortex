// Package snippet 从长文本中截取「围绕命中词」的展示片段。
//
// 背景：早期实现统一取正文前 300 个字符。当命中词出现在第 500 个字符之后时，
// 用户看到的是与查询无关的开头，会以为「搜得不准」——实际是展示层把证据截掉了。
package snippet

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// DefaultWidth 是默认片段宽度（rune 数）。
const DefaultWidth = 300

// Options 控制片段截取行为。
type Options struct {
	// Width 是片段最大 rune 数（含省略号）。<=0 时用 DefaultWidth。
	Width int
	// Context 是命中词前后保留的 rune 数，默认 Width/4。
	Context int
	// Ellipsis 是截断标记，默认 "…"。
	Ellipsis string
	// PreserveNewlines 为 true 时保留换行（适合代码），否则折叠为空格。
	PreserveNewlines bool
}

func (o Options) normalized() Options {
	if o.Width <= 0 {
		o.Width = DefaultWidth
	}
	if o.Context <= 0 {
		o.Context = o.Width / 4
		if o.Context < 20 {
			o.Context = 20
		}
	}
	if o.Ellipsis == "" {
		o.Ellipsis = "…"
	}
	return o
}

// Build 返回围绕 query 中任意命中词的最优窗口。
//
// 行为：
//   - 文本短于 Width：原样返回（折叠空白）。
//   - 能找到命中词：以第一个命中位置为中心开窗，窗口落在文本边界时自动贴边。
//   - 找不到命中词：退回取开头 Width 个字符。
//
// 返回的字符串在需要截断时以 Ellipsis 结尾/开头，便于前端识别。
func Build(text, query string, opts Options) string {
	o := opts.normalized()
	text = normalizeSpace(text, o.PreserveNewlines)
	if text == "" {
		return ""
	}

	runes := []rune(text)
	if len(runes) <= o.Width {
		return text
	}

	terms := QueryTerms(query)
	if len(terms) == 0 {
		return trimToWidth(runes, o)
	}

	// 找最早出现的命中位置（rune 偏移）
	lowerRunes := []rune(strings.ToLower(text))
	bestPos := -1
	for _, term := range terms {
		termRunes := []rune(strings.ToLower(term))
		if len(termRunes) == 0 {
			continue
		}
		if idx := indexOf(lowerRunes, termRunes); idx >= 0 && (bestPos < 0 || idx < bestPos) {
			bestPos = idx
		}
	}
	if bestPos < 0 {
		return trimToWidth(runes, o)
	}

	// 以命中位置为中心开窗，并贴边修正
	start := bestPos - o.Context
	if start < 0 {
		start = 0
	}
	end := start + o.Width
	if end > len(runes) {
		end = len(runes)
		start = end - o.Width
		if start < 0 {
			start = 0
		}
	}

	out := string(runes[start:end])
	if start > 0 {
		out = o.Ellipsis + out
	}
	if end < len(runes) {
		out += o.Ellipsis
	}
	return out
}

// QueryTerms 把查询拆成用于定位的候选词。
//
// 中文查询按整词与单字都试（因为索引侧是单字分词），英文按空格拆词。
func QueryTerms(query string) []string {
	fields := strings.Fields(strings.TrimSpace(query))
	seen := make(map[string]bool)
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || utf8.RuneCountInString(s) < 2 {
			return
		}
		key := strings.ToLower(s)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, s)
	}
	for _, f := range fields {
		add(f)
	}
	// 中文长词补充按 2-gram 切分，提高定位命中率
	for _, f := range fields {
		runes := []rune(f)
		if len(runes) < 3 {
			continue
		}
		hasCJK := false
		for _, r := range runes {
			if unicode.Is(unicode.Han, r) {
				hasCJK = true
				break
			}
		}
		if !hasCJK {
			continue
		}
		for i := 0; i+2 <= len(runes); i++ {
			add(string(runes[i : i+2]))
		}
	}
	return out
}

func normalizeSpace(s string, keepNewlines bool) string {
	s = strings.TrimSpace(s)
	if keepNewlines {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	lastSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !lastSpace {
				b.WriteRune(' ')
				lastSpace = true
			}
			continue
		}
		b.WriteRune(r)
		lastSpace = false
	}
	return b.String()
}

func trimToWidth(runes []rune, o Options) string {
	if len(runes) <= o.Width {
		return string(runes)
	}
	return string(runes[:o.Width]) + o.Ellipsis
}

func indexOf(haystack, needle []rune) int {
	if len(needle) == 0 || len(haystack) < len(needle) {
		return -1
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
