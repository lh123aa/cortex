package symbols

import (
	"strings"
	"testing"
)

func TestLanguageOf(t *testing.T) {
	cases := map[string]string{
		"a.go": "go", "b.py": "python", "c.js": "javascript", "d.ts": "typescript",
		"e.tsx": "typescript", "f.java": "java", "g.rs": "rust", "h.md": "",
	}
	for path, want := range cases {
		if got := LanguageOf(path); got != want {
			t.Errorf("LanguageOf(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestExtractGo(t *testing.T) {
	src := `package demo

import "fmt"

// Cache 是缓存接口
type Cache interface {
	Get(k string) string
}

type server struct {
	name string
}

func NewServer(name string) *server {
	return &server{name: name}
}

func (s *server) Start(port int) error {
	return nil
}

func (s *server) stop() {}
`
	syms := Extract("demo.go", src)
	byName := indexByName(syms)

	assertKind(t, byName, "Cache", KindInterface)
	assertKind(t, byName, "server", KindStruct)
	assertKind(t, byName, "NewServer", KindFunction)
	assertKind(t, byName, "Start", KindMethod)

	if s, ok := byName["Start"]; ok && s.Line == 0 {
		t.Fatal("symbol line must be 1-based and non-zero")
	}
}

func TestExtractPython(t *testing.T) {
	src := `import os

class Parser:
    def __init__(self, path):
        self.path = path

    def parse(self):
        return []

    @staticmethod
    def helper():
        pass

async def fetch_all(urls):
    return urls
`
	syms := Extract("p.py", src)
	byName := indexByName(syms)

	assertKind(t, byName, "Parser", KindClass)
	assertKind(t, byName, "__init__", KindMethod)
	assertKind(t, byName, "parse", KindMethod)
	assertKind(t, byName, "fetch_all", KindFunction)
}

func TestExtractJavaScript(t *testing.T) {
	src := `import fs from 'fs';

export function loadConfig(path) {
  return {};
}

export const parseArgs = (argv) => argv.slice(2);

export default class App {
  constructor(name) {
    this.name = name;
  }
  run() {
    console.log(this.name);
  }
}
`
	syms := Extract("app.js", src)
	byName := indexByName(syms)

	assertKind(t, byName, "loadConfig", KindFunction)
	assertKind(t, byName, "parseArgs", KindFunction)
	assertKind(t, byName, "App", KindClass)
}

func TestExtractTypeScript(t *testing.T) {
	src := `export interface UserConfig {
  name: string;
}

export type Plan = 'free' | 'pro';

export class Store {
  get(key: string): string {
    return key;
  }
}

export async function save(data: unknown): Promise<void> {}
`
	syms := Extract("store.ts", src)
	byName := indexByName(syms)

	assertKind(t, byName, "UserConfig", KindInterface)
	assertKind(t, byName, "Plan", KindType)
	assertKind(t, byName, "Store", KindClass)
	assertKind(t, byName, "save", KindFunction)
}

func TestExtractRust(t *testing.T) {
	src := `pub struct Config {
    name: String,
}

pub trait Store {
    fn get(&self) -> Option<&str>;
}

impl Config {
    pub async fn load(path: &str) -> Self {
        Config { name: path.into() }
    }
}
`
	syms := Extract("lib.rs", src)
	byName := indexByName(syms)

	assertKind(t, byName, "Config", KindStruct)
	assertKind(t, byName, "Store", KindInterface)
	assertKind(t, byName, "load", KindFunction)
}

func TestExtractIgnoresCommentsAndNonCode(t *testing.T) {
	src := `// func FakeFunc() {}
# def fake_py():
/* function fakeJs() {} */
package real
func RealFunc() {}
`
	syms := Extract("x.go", src)
	byName := indexByName(syms)
	if _, ok := byName["FakeFunc"]; ok {
		t.Error("symbol inside line comment should be ignored")
	}
	if _, ok := byName["RealFunc"]; !ok {
		t.Error("RealFunc should be extracted")
	}
}

func TestExtractSkipsControlKeywords(t *testing.T) {
	src := `class A {
  if (x) {
  }
  run() {
  }
}
`
	syms := Extract("a.js", src)
	for _, s := range syms {
		if s.Name == "if" {
			t.Fatalf("control keyword should not be a symbol: %+v", s)
		}
	}
}

func TestExtractUnknownLanguage(t *testing.T) {
	if got := Extract("readme.md", "# hi\nfunc X()"); len(got) != 0 {
		t.Fatalf("markdown should produce no symbols, got %+v", got)
	}
}

func TestSignatureTruncation(t *testing.T) {
	long := "func VeryLongName(" + strings.Repeat("a, ", 200) + ")"
	syms := Extract("x.go", "pack"+"age x\n"+long)
	if len(syms) == 0 {
		t.Fatal("expected symbol")
	}
	if n := len([]rune(syms[0].Signature)); n > 210 {
		t.Fatalf("signature not truncated: %d runes", n)
	}
}

func TestFindExactBeatsContains(t *testing.T) {
	all := []Symbol{
		{Name: "startServer", Kind: KindFunction},
		{Name: "Start", Kind: KindFunction},
		{Name: "restart", Kind: KindFunction},
	}
	got := Find(all, "start", 10)
	if len(got) == 0 || got[0].Name != "Start" {
		t.Fatalf("exact match should rank first, got %+v", got)
	}
}

func TestFindLimit(t *testing.T) {
	all := make([]Symbol, 50)
	for i := range all {
		all[i] = Symbol{Name: "foo", Kind: KindFunction}
	}
	if got := Find(all, "foo", 5); len(got) != 5 {
		t.Fatalf("limit not respected: %d", len(got))
	}
}

// TestDedupe 验证去重语义：
//   - 同一行被规则重复匹配 → 合并
//   - 同名但不同行（不同接收者的方法）→ 保留
func TestDedupe(t *testing.T) {
	same := []Symbol{
		{Name: "A", Kind: KindFunction, Line: 1, Path: "x.go"},
		{Name: "A", Kind: KindFunction, Line: 1, Path: "x.go"}, // 同行重复
	}
	if got := dedupe(same); len(got) != 1 {
		t.Fatalf("same-line duplicates should be merged, got %d", len(got))
	}

	diffLines := []Symbol{
		{Name: "Name", Kind: KindMethod, Line: 3, Path: "x.go"},
		{Name: "Name", Kind: KindMethod, Line: 4, Path: "x.go"}, // 不同接收者
	}
	if got := dedupe(diffLines); len(got) != 2 {
		t.Fatalf("same name on different lines should be kept, got %d", len(got))
	}
}

func indexByName(syms []Symbol) map[string]Symbol {
	m := make(map[string]Symbol, len(syms))
	for _, s := range syms {
		if _, exists := m[s.Name]; !exists {
			m[s.Name] = s
		}
	}
	return m
}

func assertKind(t *testing.T, byName map[string]Symbol, name string, want Kind) {
	t.Helper()
	s, ok := byName[name]
	if !ok {
		t.Fatalf("missing symbol %q; got %v", name, keysOf(byName))
	}
	if s.Kind != want {
		t.Fatalf("symbol %q kind = %q, want %q", name, s.Kind, want)
	}
}

func keysOf(m map[string]Symbol) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
