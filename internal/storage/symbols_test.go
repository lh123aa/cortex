package storage

import (
	"os"
	"testing"

	"github.com/lh123aa/cortex/internal/symbols"
)

// newSymbolTestDB 创建独立测试库。
// 不复用 crud_test.go 的 setupTestDB：后者带 //go:build cgo 标签，
// 在纯 Go 构建（CGO_ENABLED=0）下不可见，而本项目主打无 CGO。
func newSymbolTestDB(t *testing.T) (*SQLiteStorage, func()) {
	t.Helper()
	f, err := os.CreateTemp("", "cortex-sym-*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	_ = f.Close()

	st, err := NewSQLiteStorage(f.Name())
	if err != nil {
		_ = os.Remove(f.Name())
		t.Fatalf("open storage: %v", err)
	}
	return st, func() {
		_ = st.Close()
		_ = os.Remove(f.Name())
	}
}

func TestSaveAndSearchSymbols(t *testing.T) {
	st, cleanup := newSymbolTestDB(t)
	defer cleanup()

	syms := []symbols.Symbol{
		{Name: "StartServer", Kind: symbols.KindFunction, Line: 12, Signature: "func StartServer(port int) error", Language: "go", Path: "/a/server.go"},
		{Name: "StopServer", Kind: symbols.KindFunction, Line: 30, Signature: "func StopServer()", Language: "go", Path: "/a/server.go"},
		{Name: "Config", Kind: symbols.KindStruct, Line: 5, Signature: "type Config struct", Language: "go", Path: "/a/config.go"},
	}
	if err := st.SaveSymbols("doc1", syms); err != nil {
		t.Fatalf("SaveSymbols failed: %v", err)
	}

	got, err := st.SearchSymbols("StartServer", "", "", 10)
	if err != nil {
		t.Fatalf("SearchSymbols failed: %v", err)
	}
	if len(got) != 1 || got[0].Name != "StartServer" {
		t.Fatalf("exact lookup failed: %+v", got)
	}
	if got[0].Line != 12 {
		t.Fatalf("line number lost: %+v", got[0])
	}
}

func TestSearchSymbolsRanking(t *testing.T) {
	st, cleanup := newSymbolTestDB(t)
	defer cleanup()

	if err := st.SaveSymbols("doc1", []symbols.Symbol{
		{Name: "startBackground", Kind: symbols.KindFunction, Language: "go", Path: "/a.go"},
		{Name: "Start", Kind: symbols.KindFunction, Language: "go", Path: "/b.go"},
		{Name: "Restart", Kind: symbols.KindFunction, Language: "go", Path: "/c.go"},
	}); err != nil {
		t.Fatal(err)
	}

	got, err := st.SearchSymbols("start", "", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Fatal("no results")
	}
	if got[0].Name != "Start" {
		t.Fatalf("exact match should rank first, got %q", got[0].Name)
	}
	// 前缀 startBackground 应排在后缀包含 Restart 之前
	order := map[string]int{}
	for i, s := range got {
		order[s.Name] = i
	}
	if order["startBackground"] > order["Restart"] {
		t.Fatalf("prefix match should beat contains match: %v", order)
	}
}

func TestSearchSymbolsFilters(t *testing.T) {
	st, cleanup := newSymbolTestDB(t)
	defer cleanup()

	if err := st.SaveSymbols("doc1", []symbols.Symbol{
		{Name: "load", Kind: symbols.KindFunction, Language: "go", Path: "/go/a.go"},
		{Name: "load", Kind: symbols.KindFunction, Language: "python", Path: "/py/b.py"},
	}); err != nil {
		t.Fatal(err)
	}

	goOnly, err := st.SearchSymbols("load", "go", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(goOnly) != 1 || goOnly[0].Language != "go" {
		t.Fatalf("language filter failed: %+v", goOnly)
	}

	pathOnly, err := st.SearchSymbols("load", "", "/py/", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(pathOnly) != 1 || pathOnly[0].Language != "python" {
		t.Fatalf("path filter failed: %+v", pathOnly)
	}
}

func TestSaveSymbolsReplacesOld(t *testing.T) {
	st, cleanup := newSymbolTestDB(t)
	defer cleanup()

	if err := st.SaveSymbols("doc1", []symbols.Symbol{{Name: "old", Kind: symbols.KindFunction}}); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveSymbols("doc1", []symbols.Symbol{{Name: "new", Kind: symbols.KindFunction}}); err != nil {
		t.Fatal(err)
	}

	if n, _ := st.CountSymbols(); n != 1 {
		t.Fatalf("expected 1 symbol after replace, got %d", n)
	}
	got, _ := st.SearchSymbols("old", "", "", 10)
	if len(got) != 0 {
		t.Fatalf("old symbol should be gone: %+v", got)
	}
}

func TestSearchSymbolsEmptyQuery(t *testing.T) {
	st, cleanup := newSymbolTestDB(t)
	defer cleanup()

	got, err := st.SearchSymbols("  ", "", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("empty query should return nothing, got %+v", got)
	}
}

func TestSearchSymbolsLimit(t *testing.T) {
	st, cleanup := newSymbolTestDB(t)
	defer cleanup()

	var batch []symbols.Symbol
	for i := 0; i < 30; i++ {
		batch = append(batch, symbols.Symbol{Name: "fn", Kind: symbols.KindFunction})
	}
	if err := st.SaveSymbols("doc1", batch); err != nil {
		t.Fatal(err)
	}
	got, err := st.SearchSymbols("fn", "", "", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Fatalf("limit not respected: %d", len(got))
	}
}
