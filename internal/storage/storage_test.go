package storage

import (
	"os"
	"testing"

	"github.com/lh123aa/cortex/internal/models"
)

func newTestDB(t *testing.T) *SQLiteStorage {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "cortex-test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp db: %v", err)
	}
	tmpFile.Close()
	db, err := NewSQLiteStorage(tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to create storage: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
		os.Remove(tmpFile.Name())
	})
	return db
}

func TestSQLiteStorage_SaveAndGetDocument(t *testing.T) {
	s := newTestDB(t)

	doc := &models.Document{
		ID:          "test-doc-1",
		UserID:      "user-1",
		Path:        "/test/doc.md",
		FileType:    "md",
		ContentHash: "abc123",
		FileSize:    1024,
		ChunkCount:  3,
		Status:      "indexed",
	}

	err := s.SaveDocument(doc)
	if err != nil {
		t.Fatalf("SaveDocument failed: %v", err)
	}

	got, err := s.GetDocumentByID("test-doc-1", "user-1")
	if err != nil {
		t.Fatalf("GetDocumentByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("document not found")
	}
	if got.Path != "/test/doc.md" {
		t.Errorf("path = %q, want %q", got.Path, "/test/doc.md")
	}
	if got.ContentHash != "abc123" {
		t.Errorf("content_hash = %q, want %q", got.ContentHash, "abc123")
	}
}

func TestSQLiteStorage_DocumentUserIsolation(t *testing.T) {
	s := newTestDB(t)

	doc1 := &models.Document{ID: "doc-1", UserID: "user-a", Path: "/doc.md", Status: "indexed"}
	s.SaveDocument(doc1)

	// user-b should not see user-a's document
	got, err := s.GetDocumentByID("doc-1", "user-b")
	if err != nil {
		t.Fatalf("GetDocumentByID failed: %v", err)
	}
	if got != nil {
		t.Error("user-b should not see user-a's document")
	}

	// user-a should see it
	got, err = s.GetDocumentByID("doc-1", "user-a")
	if err != nil {
		t.Fatalf("GetDocumentByID failed: %v", err)
	}
	if got == nil {
		t.Error("user-a should see their document")
	}
}

func TestSQLiteStorage_SaveAndGetChunks(t *testing.T) {
	s := newTestDB(t)

	doc := &models.Document{ID: "chunk-doc", UserID: "user-1", Path: "/chunks.md", Status: "indexed"}
	s.SaveDocument(doc)

	chunks := []*models.Chunk{
		{ID: "chunk-1", UserID: "user-1", DocumentID: "chunk-doc", ContentRaw: "first chunk", ContentHash: "hash-1"},
		{ID: "chunk-2", UserID: "user-1", DocumentID: "chunk-doc", ContentRaw: "second chunk", ContentHash: "hash-2"},
	}

	err := s.SaveChunks(chunks)
	if err != nil {
		t.Fatalf("SaveChunks failed: %v", err)
	}

	// GetChunkByHash
	c1, err := s.GetChunkByHash("hash-1", "user-1")
	if err != nil {
		t.Fatalf("GetChunkByHash failed: %v", err)
	}
	if c1 == nil {
		t.Fatal("chunk not found by hash")
	}
	if c1.ContentHash != "hash-1" {
		t.Errorf("content_hash = %q, want %q", c1.ContentHash, "hash-1")
	}
}

func TestSQLiteStorage_GetChunkByHash_NotFound(t *testing.T) {
	s := newTestDB(t)
	c, err := s.GetChunkByHash("nonexistent-hash", "user-1")
	if err != nil {
		t.Fatalf("GetChunkByHash failed: %v", err)
	}
	if c != nil {
		t.Error("expected nil for nonexistent hash")
	}
}

func TestSQLiteStorage_Metadata(t *testing.T) {
	s := newTestDB(t)

	err := s.SetMetadata("version", "2.4.0")
	if err != nil {
		t.Fatalf("SetMetadata failed: %v", err)
	}

	val, err := s.GetMetadata("version")
	if err != nil {
		t.Fatalf("GetMetadata failed: %v", err)
	}
	if val != "2.4.0" {
		t.Errorf("version = %q, want %q", val, "2.4.0")
	}

	// nonexistent key
	val, err = s.GetMetadata("nonexistent")
	if err != nil {
		t.Fatalf("GetMetadata failed: %v", err)
	}
	if val != "" {
		t.Errorf("expected empty for nonexistent key, got %q", val)
	}
}
