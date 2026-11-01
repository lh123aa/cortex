package storage

import (
	"path/filepath"
	"testing"

	"github.com/lh123aa/cortex/internal/models"
	"github.com/lh123aa/cortex/internal/symbols"
)

// TestSaveDocumentBundleWithSymbols 覆盖「代码文件走 bundle 写入」这条路径。
//
// 这个测试是为了防止一个真实事故复发：bundle.go 曾把符号表列名写成
// file_path（真实 schema 是 path + language），导致所有 .go 文件索引失败，
// 报错 "table symbols has no column named file_path"。
// 之前的测试只覆盖了无符号的 .md 文档，所以没发现。
func TestSaveDocumentBundleWithSymbols(t *testing.T) {
	dir := t.TempDir()
	st, err := NewSQLiteStorage(filepath.Join(dir, "bundle.db"))
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	defer st.Close()
	if err := st.InitSymbolsTable(); err != nil {
		t.Fatalf("init symbols table: %v", err)
	}

	doc := &models.Document{
		ID:       "doc-go-1",
		UserID:   "",
		Path:     "internal/storage/bundle.go",
		FileType: "go",
		Status:   "indexed",
	}
	chunks := []*models.Chunk{{
		ID:         "chunk-1",
		UserID:     "",
		DocumentID: doc.ID,
		Content:    "func SaveDocumentBundle(b *DocumentBundle) error { return nil }",
		ContentRaw: "func SaveDocumentBundle b DocumentBundle error return nil",
	}}
	syms := []symbols.Symbol{
		{Name: "SaveDocumentBundle", Kind: symbols.KindFunction, Line: 3, Signature: "func SaveDocumentBundle(b *DocumentBundle) error"},
		{Name: "DocumentBundle", Kind: symbols.KindType, Line: 15, Signature: "type DocumentBundle struct"},
	}
	doc.ChunkCount = len(chunks)

	if err := st.SaveDocumentBundle(&DocumentBundle{Doc: doc, Chunks: chunks, Symbols: syms}); err != nil {
		t.Fatalf("SaveDocumentBundle with symbols failed: %v", err)
	}

	// 文档、分块、符号都必须真的落库
	got, err := st.GetDocumentByID(doc.ID, "")
	if err != nil || got == nil {
		t.Fatalf("document not saved: %v", err)
	}
	if _, err := st.GetChunk("chunk-1", ""); err != nil {
		t.Fatalf("chunk not saved: %v", err)
	}
	found, err := st.SearchSymbols("SaveDocumentBundle", "", "", 10)
	if err != nil {
		t.Fatalf("query symbols: %v", err)
	}
	if len(found) == 0 {
		t.Fatal("symbols were not persisted (schema column mismatch?)")
	}
	if found[0].Line != 3 {
		t.Fatalf("symbol line = %d, want 3", found[0].Line)
	}
	if found[0].Path != doc.Path {
		t.Fatalf("symbol path = %q, want %q", found[0].Path, doc.Path)
	}
	if found[0].Language == "" {
		t.Fatal("symbol language should be derived from document path")
	}
}

// TestSaveDocumentBundleReplacesSymbols 重复索引同一文档时符号应被替换而不是累积。
func TestSaveDocumentBundleReplacesSymbols(t *testing.T) {
	dir := t.TempDir()
	st, err := NewSQLiteStorage(filepath.Join(dir, "bundle2.db"))
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	defer st.Close()
	if err := st.InitSymbolsTable(); err != nil {
		t.Fatalf("init symbols table: %v", err)
	}

	doc := &models.Document{ID: "doc-1", Path: "a.go", FileType: "go", Status: "indexed"}
	b1 := &DocumentBundle{Doc: doc, Symbols: []symbols.Symbol{
		{Name: "Old", Kind: symbols.KindFunction, Line: 1},
	}}
	if err := st.SaveDocumentBundle(b1); err != nil {
		t.Fatalf("first bundle: %v", err)
	}
	b2 := &DocumentBundle{Doc: doc, Symbols: []symbols.Symbol{
		{Name: "New", Kind: symbols.KindFunction, Line: 2},
	}}
	if err := st.SaveDocumentBundle(b2); err != nil {
		t.Fatalf("second bundle: %v", err)
	}

	old, _ := st.SearchSymbols("Old", "", "", 10)
	if len(old) != 0 {
		t.Fatalf("stale symbols should be replaced, found %d", len(old))
	}
	fresh, _ := st.SearchSymbols("New", "", "", 10)
	if len(fresh) != 1 {
		t.Fatalf("new symbol not persisted")
	}
}
