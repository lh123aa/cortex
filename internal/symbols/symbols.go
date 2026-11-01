// Package symbols 从源码中提取符号定义（函数/类/方法/类型），供代码检索使用。
//
// 为什么不用 tree-sitter：
// tree-sitter 的 Go 绑定需要 CGO，而 Cortex 的核心卖点是「纯 Go、单二进制、无 CGO」。
// 引入 CGO 会让交叉编译、Alpine 容器和 Windows 无工具链安装全部失效。
// 因此这里用「逐行词法 + 轻量状态机」提取符号：精度低于 AST，
// 但对「找函数/类定义」这个主场景足够，且零依赖、可交叉编译。
package symbols

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// Kind 是符号类型。
type Kind string

const (
	KindFunction  Kind = "function"
	KindMethod    Kind = "method"
	KindClass     Kind = "class"
	KindType      Kind = "type"
	KindInterface Kind = "interface"
	KindStruct    Kind = "struct"
	KindVariable  Kind = "variable"
)

// Symbol 是一条符号定义。
type Symbol struct {
	Name      string `json:"name"`
	Kind      Kind   `json:"kind"`
	Line      int    `json:"line"`      // 1-based
	Signature string `json:"signature"` // 定义所在行（截断）
	Language  string `json:"language"`
	Path      string `json:"path"`
}

// LanguageOf 根据扩展名判断语言，未知返回空串。
func LanguageOf(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "go"
	case ".py", ".pyi":
		return "python"
	case ".js", ".jsx", ".mjs", ".cjs":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".java":
		return "java"
	case ".rs":
		return "rust"
	default:
		return ""
	}
}

// IsCodeFile 判断是否是可提取符号的代码文件。
func IsCodeFile(path string) bool { return LanguageOf(path) != "" }

var (
	// Go
	goFunc   = regexp.MustCompile(`^func\s+(?:\([^)]*\)\s*)?([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	goMethod = regexp.MustCompile(`^func\s+\(([^)]*)\)\s*([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	goType   = regexp.MustCompile(`^type\s+([A-Za-z_][A-Za-z0-9_]*)\s+(struct|interface)?`)

	// Python
	pyFunc  = regexp.MustCompile(`^\s*(?:async\s+)?def\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	pyClass = regexp.MustCompile(`^\s*class\s+([A-Za-z_][A-Za-z0-9_]*)\s*[\(:]`)

	// JavaScript / TypeScript
	jsFunc      = regexp.MustCompile(`^\s*(?:export\s+)?(?:default\s+)?(?:async\s+)?function\s*\*?\s*([A-Za-z_$][A-Za-z0-9_$]*)\s*\(`)
	jsArrow     = regexp.MustCompile(`^\s*(?:export\s+)?(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*(?::\s*[^=]+)?=\s*(?:async\s*)?(?:\([^)]*\)|[A-Za-z_$][A-Za-z0-9_$]*)\s*=>`)
	jsClass     = regexp.MustCompile(`^\s*(?:export\s+)?(?:default\s+)?(?:abstract\s+)?class\s+([A-Za-z_$][A-Za-z0-9_$]*)`)
	jsMethod    = regexp.MustCompile(`^\s{2,}(?:public\s+|private\s+|protected\s+|static\s+|async\s+|readonly\s+)*([A-Za-z_$][A-Za-z0-9_$]*)\s*\([^)]*\)\s*(?::\s*[^{]+)?\{`)
	tsInterface = regexp.MustCompile(`^\s*(?:export\s+)?interface\s+([A-Za-z_$][A-Za-z0-9_$]*)`)
	tsType      = regexp.MustCompile(`^\s*(?:export\s+)?type\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*[=<]`)

	// Java
	javaMethod = regexp.MustCompile(`^\s+(?:public|private|protected)\s+(?:static\s+)?(?:final\s+)?[A-Za-z_<>\[\],\s]+\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	javaClass  = regexp.MustCompile(`^\s*(?:public\s+|private\s+|protected\s+)?(?:static\s+)?(?:final\s+)?(?:abstract\s+)?(class|interface|enum)\s+([A-Za-z_][A-Za-z0-9_]*)`)

	// Rust
	rustFn   = regexp.MustCompile(`^\s*(?:pub(?:\([^)]*\))?\s+)?(?:async\s+)?(?:unsafe\s+)?fn\s+([A-Za-z_][A-Za-z0-9_]*)`)
	rustType = regexp.MustCompile(`^\s*(?:pub(?:\([^)]*\))?\s+)?(struct|enum|trait|type)\s+([A-Za-z_][A-Za-z0-9_]*)`)
)

// Extract 从源码内容中提取符号定义。
// path 用于判断语言并写入结果的 Path 字段；content 是完整源码。
//
// Go 走官方解析器（准确率 100%）；解析失败（例如文件有语法错误）时
// 自动回退到逐行规则法，保证「至少给点结果」而不是直接空手而归。
func Extract(path, content string) []Symbol {
	lang := LanguageOf(path)
	if lang == "" {
		return nil
	}

	if lang == "go" {
		if syms, err := extractGoAST(path, content); err == nil {
			return syms
		}
		// 语法错误：回退到规则法
	}

	return extractRegex(path, content, lang)
}

// extractRegex 是逐行规则法，用于非 Go 语言以及 Go 语法错误时的兜底。
// 单独抽出来便于测试对照：Go 场景下它的精度明显低于 extractGoAST。
func extractRegex(path, content, lang string) []Symbol {
	lines := strings.Split(content, "\n")
	out := make([]Symbol, 0, 32)

	add := func(name string, kind Kind, lineNo int, sig string) {
		name = strings.TrimSpace(name)
		if name == "" || name == "_" {
			return
		}
		out = append(out, Symbol{
			Name:      name,
			Kind:      kind,
			Line:      lineNo,
			Signature: truncateLine(sig, 200),
			Language:  lang,
			Path:      path,
		})
	}

	for i, raw := range lines {
		lineNo := i + 1
		line := expandLeadingTabs(raw)
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			continue
		}

		switch lang {
		case "go":
			if m := goMethod.FindStringSubmatch(trimmed); m != nil {
				add(m[2], KindMethod, lineNo, trimmed)
				continue
			}
			if m := goFunc.FindStringSubmatch(trimmed); m != nil {
				add(m[1], KindFunction, lineNo, trimmed)
				continue
			}
			if m := goType.FindStringSubmatch(trimmed); m != nil {
				kind := KindType
				switch m[2] {
				case "struct":
					kind = KindStruct
				case "interface":
					kind = KindInterface
				}
				add(m[1], kind, lineNo, trimmed)
			}
		case "python":
			if m := pyClass.FindStringSubmatch(line); m != nil {
				add(m[1], KindClass, lineNo, trimmed)
				continue
			}
			if m := pyFunc.FindStringSubmatch(line); m != nil {
				kind := KindFunction
				if isIndented(line) {
					kind = KindMethod
				}
				add(m[1], kind, lineNo, trimmed)
			}
		case "javascript", "typescript":
			if m := tsInterface.FindStringSubmatch(line); m != nil {
				add(m[1], KindInterface, lineNo, trimmed)
				continue
			}
			if m := tsType.FindStringSubmatch(line); m != nil {
				add(m[1], KindType, lineNo, trimmed)
				continue
			}
			if m := jsClass.FindStringSubmatch(line); m != nil {
				add(m[1], KindClass, lineNo, trimmed)
				continue
			}
			if m := jsFunc.FindStringSubmatch(line); m != nil {
				add(m[1], KindFunction, lineNo, trimmed)
				continue
			}
			if m := jsArrow.FindStringSubmatch(line); m != nil {
				add(m[1], KindFunction, lineNo, trimmed)
				continue
			}
			if m := jsMethod.FindStringSubmatch(line); m != nil {
				if !isControlKeyword(m[1]) {
					add(m[1], KindMethod, lineNo, trimmed)
				}
			}
		case "java":
			if m := javaClass.FindStringSubmatch(line); m != nil {
				kind := KindClass
				if m[1] == "interface" {
					kind = KindInterface
				}
				add(m[2], kind, lineNo, trimmed)
				continue
			}
			if m := javaMethod.FindStringSubmatch(line); m != nil {
				add(m[1], KindMethod, lineNo, trimmed)
			}
		case "rust":
			if m := rustType.FindStringSubmatch(line); m != nil {
				kind := KindType
				switch m[1] {
				case "struct":
					kind = KindStruct
				case "trait":
					kind = KindInterface
				}
				add(m[2], kind, lineNo, trimmed)
				continue
			}
			if m := rustFn.FindStringSubmatch(line); m != nil {
				add(m[1], KindFunction, lineNo, trimmed)
			}
		}
	}

	return dedupe(out)
}

// dedupe 去掉同一行同一名字的重复符号（例如重载/误匹配）。
// dedupe 去掉重复符号。
//
// 去重键必须包含行号：同名符号完全合法且常见，
// 例如不同接收者的方法 `(A) Name()` 与 `(B) Name()` 会落在不同行。
// 早期键只有 Path|Name|Kind，会把它们误合并成一个。
func dedupe(in []Symbol) []Symbol {
	seen := make(map[string]bool, len(in))
	out := in[:0]
	for _, s := range in {
		key := fmt.Sprintf("%s|%s|%s|%d", s.Path, s.Name, s.Kind, s.Line)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, s)
	}
	return out
}

func expandLeadingTabs(line string) string {
	// 用 4 空格表示一级缩进，便于统一的「是否缩进」判断
	return strings.ReplaceAll(line, "\t", "    ")
}

func isIndented(line string) bool {
	return len(line) > 0 && (line[0] == ' ' || line[0] == '\t')
}

func shortenLine(s string, n int) string { return truncateLine(s, n) }

func truncateLine(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}

var controlKeywords = map[string]bool{
	"if": true, "for": true, "while": true, "switch": true, "catch": true,
	"return": true, "function": true, "else": true, "try": true, "do": true,
	"new": true, "typeof": true, "await": true, "yield": true,
}

func isControlKeyword(name string) bool { return controlKeywords[name] }

// Find 在已提取的符号里按名称查找（精确优先，其次包含匹配）。
func Find(all []Symbol, query string, limit int) []Symbol {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	if limit <= 0 {
		limit = 20
	}

	var exact, prefix, contains []Symbol
	for _, s := range all {
		name := strings.ToLower(s.Name)
		switch {
		case name == q:
			exact = append(exact, s)
		case strings.HasPrefix(name, q):
			prefix = append(prefix, s)
		case strings.Contains(name, q):
			contains = append(contains, s)
		}
	}

	out := make([]Symbol, 0, limit)
	for _, group := range [][]Symbol{exact, prefix, contains} {
		for _, s := range group {
			if len(out) >= limit {
				return out
			}
			out = append(out, s)
		}
	}
	return out
}
