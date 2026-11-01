package symbols

import (
	"strings"
	"testing"
)

// TestGoASTMultilineSignature 多行函数声明：规则法只能看到第一行，
// 官方解析器能拿到完整签名。
func TestGoASTMultilineSignature(t *testing.T) {
	src := `package demo

import "context"

func (s *Server) Start(
	ctx context.Context,
	port int,
) error {
	return nil
}
`
	syms := Extract("a.go", src)
	byName := indexByName(syms)

	s, ok := byName["Start"]
	if !ok {
		t.Fatalf("multiline func not extracted; got %v", keysOf(byName))
	}
	if s.Kind != KindMethod {
		t.Errorf("Start should be a method, got %q", s.Kind)
	}
	if s.Line != 5 {
		t.Errorf("Start line = %d, want 5", s.Line)
	}
	if !strings.Contains(s.Signature, "context.Context") || !strings.Contains(s.Signature, "port int") {
		t.Errorf("signature should span multiple lines, got %q", s.Signature)
	}
}

// TestGoASTGenerics 泛型函数：规则法的正则没考虑方括号，容易漏。
func TestGoASTGenerics(t *testing.T) {
	src := `package demo

func Map[T any, R any](items []T, f func(T) R) []R {
	out := make([]R, 0, len(items))
	for _, it := range items {
		out = append(out, f(it))
	}
	return out
}

type List[T any] struct {
	items []T
}
`
	syms := Extract("a.go", src)
	byName := indexByName(syms)

	if _, ok := byName["Map"]; !ok {
		t.Fatalf("generic func Map not extracted; got %v", keysOf(byName))
	}
	if s, ok := byName["List"]; !ok || s.Kind != KindStruct {
		t.Fatalf("generic struct List not extracted as struct; got %+v", byName)
	}
}

// TestGoASTIgnoresCodeInStrings 字符串里的「假代码」不能当成符号。
func TestGoASTIgnoresCodeInStrings(t *testing.T) {
	src := `package demo

const msg = "调用 func FakeFromString() 即可"
const raw = ` + "`" + `type FakeFromRawString struct{}` + "`" + `

func Real() {}
`
	syms := Extract("a.go", src)
	byName := indexByName(syms)

	if _, ok := byName["FakeFromString"]; ok {
		t.Error("code inside a string literal must not be a symbol")
	}
	if _, ok := byName["FakeFromRawString"]; ok {
		t.Error("code inside a raw string must not be a symbol")
	}
	if _, ok := byName["Real"]; !ok {
		t.Error("Real() should be extracted")
	}
}

// TestGoASTIgnoresCommentedOut 被注释掉的函数不能算符号。
func TestGoASTIgnoresCommentedOut(t *testing.T) {
	src := `package demo

// func CommentedOut() {}
/* func BlockCommented() {} */

func Alive() {}
`
	syms := Extract("a.go", src)
	byName := indexByName(syms)

	if _, ok := byName["CommentedOut"]; ok {
		t.Error("line-commented func must not be a symbol")
	}
	if _, ok := byName["BlockCommented"]; ok {
		t.Error("block-commented func must not be a symbol")
	}
	if _, ok := byName["Alive"]; !ok {
		t.Error("Alive() should be extracted")
	}
}

// TestGoASTGroupedTypes 分组类型声明：type ( A struct{} B interface{} )。
func TestGoASTGroupedTypes(t *testing.T) {
	src := `package demo

type (
	Config struct {
		Name string
	}
	Store interface {
		Get(k string) string
	}
	Alias = Config
)
`
	syms := Extract("a.go", src)
	byName := indexByName(syms)

	if s, ok := byName["Config"]; !ok || s.Kind != KindStruct {
		t.Errorf("Config should be struct, got %+v", s)
	}
	if s, ok := byName["Store"]; !ok || s.Kind != KindInterface {
		t.Errorf("Store should be interface, got %+v", s)
	}
	if s, ok := byName["Alias"]; !ok || s.Kind != KindType {
		t.Errorf("Alias should be type, got %+v", s)
	}
}

// TestGoASTLineNumbers 行号必须精确到声明行。
func TestGoASTLineNumbers(t *testing.T) {
	src := `package demo

// 第 3 行是注释

func First() {}   // 第 5 行

type Second struct{} // 第 7 行
`
	syms := Extract("a.go", src)
	for _, s := range syms {
		switch s.Name {
		case "First":
			if s.Line != 5 {
				t.Errorf("First line = %d, want 5", s.Line)
			}
		case "Second":
			if s.Line != 7 {
				t.Errorf("Second line = %d, want 7", s.Line)
			}
		}
	}
}

// TestGoASTAnonymousFuncNotSymbol 匿名函数（闭包）不应产生符号。
func TestGoASTAnonymousFuncNotSymbol(t *testing.T) {
	src := `package demo

func Outer() {
	f := func(x int) int { return x + 1 }
	_ = f
}
`
	syms := Extract("a.go", src)
	if len(syms) != 1 || syms[0].Name != "Outer" {
		t.Fatalf("expected only Outer, got %+v", syms)
	}
}

// TestGoASTSyntaxErrorFallsBackToRegex 语法错误的文件应回退到规则法，
// 而不是返回空结果。
func TestGoASTSyntaxErrorFallsBackToRegex(t *testing.T) {
	src := `package demo

func Broken( {
}
`
	syms := Extract("a.go", src)
	// 语法错误时解析器失败，回退规则法；至少应识别出 Broken
	found := false
	for _, s := range syms {
		if s.Name == "Broken" {
			found = true
		}
	}
	if !found {
		t.Fatalf("syntax-error file should fall back to regex and still find Broken; got %+v", syms)
	}
}

// TestGoASTMatchesRegexOnSimpleCase 简单场景下两种实现结果应一致（回归护栏）。
func TestGoASTMatchesRegexOnSimpleCase(t *testing.T) {
	src := `package demo

type A struct{}
type B interface{}
func C() {}
func (a A) D() {}
`
	syms := Extract("a.go", src)
	byName := indexByName(syms)

	expect := map[string]Kind{
		"A": KindStruct,
		"B": KindInterface,
		"C": KindFunction,
		"D": KindMethod,
	}
	if len(byName) != len(expect) {
		t.Errorf("symbol count = %d, want %d (%v)", len(byName), len(expect), keysOf(byName))
	}
	for name, kind := range expect {
		s, ok := byName[name]
		if !ok {
			t.Errorf("missing %q", name)
			continue
		}
		if s.Kind != kind {
			t.Errorf("%q kind = %q, want %q", name, s.Kind, kind)
		}
	}
}
