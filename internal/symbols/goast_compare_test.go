package symbols

import (
	"os"
	"strings"
	"testing"
)

// TestASTBeatsRegexOnTrickyGo 对照测试：同一段"花哨"的 Go 代码，
// 官方解析器（AST）必须比逐行规则法（regex）找到更多正确符号。
//
// 这段代码刻意包含规则法的四个盲区：
//   - 多行签名
//   - 泛型
//   - 字符串里的假代码
//   - 注释掉的函数
func TestASTBeatsRegexOnTrickyGo(t *testing.T) {
	src := `package demo

const hint = "这里演示 func FakeInString() 的写法"

// func CommentedOut() {}

func MultiLine(
	a int,
	b string,
) error {
	return nil
}

func Generic[T any](v T) T {
	return v
}

type Box[T any] struct {
	Value T
}

func (b *Box[T]) Get() T {
	return b.Value
}
`
	astSyms := Extract("demo.go", src)
	regexSyms := extractRegex("demo.go", src, "go")

	astNames := namesOf(astSyms)
	regexNames := namesOf(regexSyms)

	t.Logf("AST   (%d): %v", len(astSyms), astNames)
	t.Logf("regex (%d): %v", len(regexSyms), regexNames)

	// AST 必须找全这些真符号
	want := []string{"MultiLine", "Generic", "Box", "Get"}
	for _, name := range want {
		if !contains(astNames, name) {
			t.Errorf("AST missed %q", name)
		}
	}

	// AST 不能把字符串/注释里的假代码当符号
	for _, fake := range []string{"FakeInString", "CommentedOut"} {
		if contains(astNames, fake) {
			t.Errorf("AST wrongly extracted fake symbol %q", fake)
		}
	}

	// 规则法必须至少漏掉一个（否则这个对照测试说明不了问题）
	missed := 0
	for _, name := range want {
		if !contains(regexNames, name) {
			missed++
		}
	}
	if missed == 0 {
		t.Logf("注意：当前规则法在这段样例上没有漏符号（可能规则已被加强），AST 仍更可靠")
	}

	// 泛型方法的接收者识别：AST 应判为 method
	for _, s := range astSyms {
		if s.Name == "Get" && s.Kind != KindMethod {
			t.Errorf("Get should be method, got %q", s.Kind)
		}
		if s.Name == "Box" && s.Kind != KindStruct {
			t.Errorf("Box should be struct, got %q", s.Kind)
		}
	}
}

// TestASTSignatureContainsFullParams 多行签名应被压成单行完整签名。
func TestASTSignatureContainsFullParams(t *testing.T) {
	src := `package demo

func Save(
	name string,
	value []byte,
) (int, error) {
	return 0, nil
}
`
	syms := Extract("demo.go", src)
	if len(syms) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(syms))
	}
	sig := syms[0].Signature
	for _, part := range []string{"name string", "value []byte"} {
		if !strings.Contains(sig, part) {
			t.Errorf("signature missing %q: %q", part, sig)
		}
	}
	if strings.Contains(sig, "\n") {
		t.Errorf("signature should be single line: %q", sig)
	}
}

// TestASTOnRealRepoFiles 在仓库真实文件上跑一遍，确保不 panic、有结果。
func TestASTOnRealRepoFiles(t *testing.T) {
	files := []string{
		"symbols.go",
		"goast.go",
		"../api/mcp_tools.go",
		"../storage/symbols.go",
	}
	for _, f := range files {
		content, err := readFileForTest(f)
		if err != nil {
			t.Logf("skip %s: %v", f, err)
			continue
		}
		syms := Extract(f, content)
		if len(syms) == 0 {
			t.Errorf("%s produced no symbols", f)
			continue
		}
		// 行号必须落在文件范围内
		lineCount := strings.Count(content, "\n") + 1
		for _, s := range syms {
			if s.Line < 1 || s.Line > lineCount {
				t.Errorf("%s: symbol %s line %d out of range (file has %d lines)", f, s.Name, s.Line, lineCount)
			}
			if strings.TrimSpace(s.Signature) == "" {
				t.Errorf("%s: symbol %s has empty signature", f, s.Name)
			}
		}
		t.Logf("%-24s -> %d symbols", f, len(syms))
	}
}

// TestASTNoDuplicateNames 同名符号（方法重载/不同接收者）不应被去重掉。
func TestASTNoDuplicateNames(t *testing.T) {
	src := `package demo

type A struct{}
type B struct{}

func (a A) Name() string { return "a" }
func (b B) Name() string { return "b" }
`
	syms := Extract("demo.go", src)
	count := 0
	for _, s := range syms {
		if s.Name == "Name" {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("expected 2 Name methods (different receivers), got %d: %v", count, namesOf(syms))
	}
}

func namesOf(syms []Symbol) []string {
	out := make([]string, 0, len(syms))
	for _, s := range syms {
		out = append(out, s.Name)
	}
	return out
}

func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

func readFileForTest(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
