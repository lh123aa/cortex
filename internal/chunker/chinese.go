package chunker

import (
	"strings"
)

// isCJK 判断是否为中日韩统一表意文字
func isCJK(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
		(r >= 0x3400 && r <= 0x4DBF) || // CJK Extension A
		(r >= 0x2E80 && r <= 0x2EFF) || // CJK Radicals Supplement
		(r >= 0x3000 && r <= 0x303F) || // CJK Symbols and Punctuation
		(r >= 0xFF00 && r <= 0xFFEF) || // Halfwidth and Fullwidth Forms
		(r >= 0x20000 && r <= 0x2A6DF) || // CJK Extension B
		(r >= 0xF900 && r <= 0xFAFF) // CJK Compatibility Ideographs
}

// SegmentChinese 对中文文本进行分词，在词之间插入空格
// 使用滑动窗口最大匹配 + 单字成词的后备策略
func SegmentChinese(text string) string {
	if !containsChinese(text) {
		return text
	}

	var result strings.Builder
	var buf []rune
	inChinese := false

	for _, r := range text {
		if isCJK(r) {
			if !inChinese && len(buf) > 0 {
				result.WriteString(string(buf))
				buf = buf[:0]
			}
			inChinese = true
			buf = append(buf, r)
		} else {
			if inChinese && len(buf) > 0 {
				// 输出中文分词结果
				result.WriteString(segmentChineseRunes(buf))
				result.WriteByte(' ')
				buf = buf[:0]
			}
			inChinese = false
			buf = append(buf, r)
		}
	}

	// 处理剩余缓冲区
	if inChinese && len(buf) > 0 {
		result.WriteString(segmentChineseRunes(buf))
	} else if len(buf) > 0 {
		result.WriteString(string(buf))
	}

	return result.String()
}

// segmentChineseRunes 将中文字符序列分词（2-gram + 单字后备）
func segmentChineseRunes(runes []rune) string {
	if len(runes) == 0 {
		return ""
	}
	if len(runes) == 1 {
		return string(runes)
	}

	var parts []string
	i := 0
	for i < len(runes) {
		if i+1 < len(runes) {
			// 输出2-gram
			parts = append(parts, string(runes[i:i+2]))
			i++
		} else {
			parts = append(parts, string(runes[i]))
			i++
		}
	}
	return strings.Join(parts, " ")
}

// containsChinese 检查文本是否包含中文字符
func containsChinese(text string) bool {
	for _, r := range text {
		if isCJK(r) {
			return true
		}
	}
	return false
}

// OptimizeForChineseSearch 对文本进行中文搜索优化（预处理）
// 在索引和搜索前调用，使 FTS5 能正确匹配中文
func OptimizeForChineseSearch(text string) string {
	return SegmentChinese(text)
}
