package storage

import (
	"fmt"
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

func TestSQLiteStorage_EmptyDB(t *testing.T) {
	s := newTestDB(t)

	// Empty database should return zero counts
	docCount, err := s.GetDocumentsCount("")
	if err != nil {
		t.Fatalf("GetDocumentsCount on empty DB failed: %v", err)
	}
	if docCount != 0 {
		t.Errorf("doc count on empty DB = %d, want 0", docCount)
	}

	chunkCount, err := s.GetChunksCount("")
	if err != nil {
		t.Fatalf("GetChunksCount on empty DB failed: %v", err)
	}
	if chunkCount != 0 {
		t.Errorf("chunk count on empty DB = %d, want 0", chunkCount)
	}

	// Search on empty DB should return empty results
	results, err := s.FTSSearch("test", "", 10)
	if err != nil {
		t.Fatalf("FTSSearch on empty DB failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("search on empty DB returned %d results, want 0", len(results))
	}

	// List documents on empty DB
	docs, err := s.ListDocuments("", 10, 0)
	if err != nil {
		t.Fatalf("ListDocuments on empty DB failed: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("list on empty DB returned %d docs, want 0", len(docs))
	}

	// Get nonexistent document
	doc, err := s.GetDocumentByID("nonexistent", "")
	if err != nil {
		t.Fatalf("GetDocumentByID on empty DB failed: %v", err)
	}
	if doc != nil {
		t.Error("expected nil for nonexistent document")
	}

	// Get nonexistent chunk
	chunk, err := s.GetChunk("nonexistent", "")
	if err != nil {
		t.Fatalf("GetChunk on empty DB failed: %v", err)
	}
	if chunk != nil {
		t.Error("expected nil for nonexistent chunk")
	}
}

func TestSQLiteStorage_SpecialChars(t *testing.T) {
	s := newTestDB(t)

	// Document with special characters in path
	doc := &models.Document{
		ID:          "special-doc",
		UserID:      "user-1",
		Path:        "/test/special chars' \"and more.md",
		FileType:    "md",
		ContentHash: "special-hash",
		Status:      "indexed",
	}
	if err := s.SaveDocument(doc); err != nil {
		t.Fatalf("SaveDocument with special chars failed: %v", err)
	}

	// Retrieve by path with special chars
	got, err := s.GetDocumentByPath("/test/special chars' \"and more.md", "user-1")
	if err != nil {
		t.Fatalf("GetDocumentByPath with special chars failed: %v", err)
	}
	if got == nil {
		t.Fatal("document with special chars not found")
	}

	// Delete document by path with special chars
	if err := s.DeleteDocumentByPath("/test/special chars' \"and more.md", "user-1"); err != nil {
		t.Fatalf("DeleteDocumentByPath with special chars failed: %v", err)
	}

	got, err = s.GetDocumentByID("special-doc", "user-1")
	if err != nil {
		t.Fatalf("GetDocumentByID after delete failed: %v", err)
	}
	if got != nil {
		t.Error("document should have been deleted")
	}
}

func TestSQLiteStorage_ConcurrentWrites(t *testing.T) {
	s := newTestDB(t)

	for i := 0; i < 10; i++ {
		doc := &models.Document{
			ID:          fmt.Sprintf("concurrent-doc-%d", i),
			UserID:      "user-1",
			Path:        fmt.Sprintf("/test/concurrent/%d.md", i),
			FileType:    "md",
			ContentHash: fmt.Sprintf("hash-%d", i),
			Status:      "indexed",
		}
		if err := s.SaveDocument(doc); err != nil {
			t.Fatalf("save failed for doc %d: %v", i, err)
		}
	}

	count, err := s.GetDocumentsCount("user-1")
	if err != nil {
		t.Fatalf("GetDocumentsCount failed: %v", err)
	}
	if count != 10 {
		t.Errorf("expected 10 documents, got %d", count)
	}
}

func TestSQLiteStorage_DocumentLifecycle(t *testing.T) {
	s := newTestDB(t)

	// Create
	doc := &models.Document{
		ID:          "lifecycle-doc",
		UserID:      "user-1",
		Path:        "/test/lifecycle.md",
		FileType:    "md",
		ContentHash: "lifecycle-hash",
		FileSize:    2048,
		ChunkCount:  1,
		Status:      "indexed",
	}
	if err := s.SaveDocument(doc); err != nil {
		t.Fatalf("SaveDocument failed: %v", err)
	}

	// Update
	doc.FileSize = 4096
	doc.ChunkCount = 2
	if err := s.SaveDocument(doc); err != nil {
		t.Fatalf("SaveDocument update failed: %v", err)
	}

	got, err := s.GetDocumentByID("lifecycle-doc", "user-1")
	if err != nil {
		t.Fatalf("GetDocumentByID failed: %v", err)
	}
	if got.FileSize != 4096 {
		t.Errorf("file size after update = %d, want 4096", got.FileSize)
	}
	if got.ChunkCount != 2 {
		t.Errorf("chunk count after update = %d, want 2", got.ChunkCount)
	}

	// Delete
	if err := s.DeleteDocument("lifecycle-doc", "user-1"); err != nil {
		t.Fatalf("DeleteDocument failed: %v", err)
	}

	got, err = s.GetDocumentByID("lifecycle-doc", "user-1")
	if err != nil {
		t.Fatalf("GetDocumentByID after delete failed: %v", err)
	}
	if got != nil {
		t.Error("document should be nil after delete")
	}
}

func TestSQLiteStorage_DeleteUserData(t *testing.T) {
	s := newTestDB(t)

	// Insert data for two users
	for _, uid := range []string{"user-a", "user-b"} {
		doc := &models.Document{
			ID:          "doc-" + uid,
			UserID:      uid,
			Path:        "/test/" + uid + ".md",
			FileType:    "md",
			ContentHash: "hash-" + uid,
			Status:      "indexed",
		}
		if err := s.SaveDocument(doc); err != nil {
			t.Fatalf("SaveDocument failed: %v", err)
		}
	}

	// Delete user-a's data
	if err := s.DeleteUserData("user-a"); err != nil {
		t.Fatalf("DeleteUserData failed: %v", err)
	}

	// user-a's document should be gone
	got, _ := s.GetDocumentByID("doc-user-a", "user-a")
	if got != nil {
		t.Error("user-a's document should be deleted")
	}

	// user-b's document should still exist
	got, _ = s.GetDocumentByID("doc-user-b", "user-b")
	if got == nil {
		t.Error("user-b's document should still exist")
	}
}

func TestSQLiteStorage_SaveChunks_Empty(t *testing.T) {
	s := newTestDB(t)

	// Saving empty chunk slice should not error
	err := s.SaveChunks(nil)
	if err != nil {
		t.Fatalf("SaveChunks(nil) failed: %v", err)
	}

	err = s.SaveChunks([]*models.Chunk{})
	if err != nil {
		t.Fatalf("SaveChunks(empty) failed: %v", err)
	}
}
