package storage

import (
	"os"
	"testing"

	"github.com/lh123aa/cortex/internal/models"
)

func setupTestDB(t *testing.T) (*SQLiteStorage, func()) {
	tmpFile, err := os.CreateTemp("", "cortex-test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()

	db, err := NewSQLiteStorage(tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("Failed to create storage: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.Remove(tmpFile.Name())
	}
	return db, cleanup
}

func TestSQLiteStorage_SaveDocument(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	doc := &models.Document{
		ID:          "test-doc-1",
		Path:        "/test/path.md",
		Title:       "Test Document",
		FileType:    "md",
		ContentHash: "abc123",
		FileSize:    1024,
		ChunkCount:  5,
		Status:      "indexed",
	}

	err := db.SaveDocument(doc)
	if err != nil {
		t.Fatalf("SaveDocument failed: %v", err)
	}

	// Verify by retrieving
	retrieved, err := db.GetDocumentByID("test-doc-1")
	if err != nil {
		t.Fatalf("GetDocumentByID failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Expected document, got nil")
	}
	if retrieved.Path != "/test/path.md" {
		t.Errorf("Expected path /test/path.md, got %s", retrieved.Path)
	}
	if retrieved.ContentHash != "abc123" {
		t.Errorf("Expected hash abc123, got %s", retrieved.ContentHash)
	}
}

func TestSQLiteStorage_GetDocumentByPath(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	doc := &models.Document{
		ID:          "test-doc-2",
		Path:        "/test/path2.md",
		FileType:    "md",
		ContentHash: "hash456",
		Status:      "indexed",
	}

	err := db.SaveDocument(doc)
	if err != nil {
		t.Fatalf("SaveDocument failed: %v", err)
	}

	retrieved, err := db.GetDocumentByPath("/test/path2.md")
	if err != nil {
		t.Fatalf("GetDocumentByPath failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Expected document, got nil")
	}
	if retrieved.ID != "test-doc-2" {
		t.Errorf("Expected ID test-doc-2, got %s", retrieved.ID)
	}
}

func TestSQLiteStorage_SaveChunks(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// First save a document
	doc := &models.Document{
		ID:          "doc-chunks-test",
		Path:        "/test/chunks.md",
		FileType:    "md",
		ContentHash: "chunks-hash",
		Status:      "indexed",
	}
	if err := db.SaveDocument(doc); err != nil {
		t.Fatalf("SaveDocument failed: %v", err)
	}

	chunks := []*models.Chunk{
		{
			ID:           "chunk-1",
			DocumentID:   "doc-chunks-test",
			HeadingPath:  "Header 1",
			HeadingLevel: 1,
			Content:      "Section: Header 1\n\nThis is chunk 1 content.",
			ContentRaw:   "This is chunk 1 content.",
			TokenCount:   5,
		},
		{
			ID:           "chunk-2",
			DocumentID:   "doc-chunks-test",
			HeadingPath:  "Header 2",
			HeadingLevel: 2,
			Content:      "Section: Header 2\n\nThis is chunk 2 content.",
			ContentRaw:   "This is chunk 2 content.",
			TokenCount:   5,
		},
	}

	err := db.SaveChunks(chunks)
	if err != nil {
		t.Fatalf("SaveChunks failed: %v", err)
	}

	// Verify retrieval
	retrieved, err := db.GetChunk("chunk-1")
	if err != nil {
		t.Fatalf("GetChunk failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Expected chunk, got nil")
	}
	if retrieved.ContentRaw != "This is chunk 1 content." {
		t.Errorf("Expected content, got %s", retrieved.ContentRaw)
	}
}

func TestSQLiteStorage_DeleteChunksByDocument(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	doc := &models.Document{
		ID:          "doc-delete-test",
		Path:        "/test/delete.md",
		FileType:    "md",
		ContentHash: "delete-hash",
		Status:      "indexed",
	}
	if err := db.SaveDocument(doc); err != nil {
		t.Fatalf("SaveDocument failed: %v", err)
	}

	chunks := []*models.Chunk{
		{
			ID:         "chunk-to-delete-1",
			DocumentID: "doc-delete-test",
			ContentRaw: "Content 1",
			TokenCount: 2,
		},
		{
			ID:         "chunk-to-delete-2",
			DocumentID: "doc-delete-test",
			ContentRaw: "Content 2",
			TokenCount: 2,
		},
	}
	if err := db.SaveChunks(chunks); err != nil {
		t.Fatalf("SaveChunks failed: %v", err)
	}

	// Delete chunks
	err := db.DeleteChunksByDocument("doc-delete-test")
	if err != nil {
		t.Fatalf("DeleteChunksByDocument failed: %v", err)
	}

	// Verify chunks are deleted
	for _, id := range []string{"chunk-to-delete-1", "chunk-to-delete-2"} {
		retrieved, err := db.GetChunk(id)
		if err != nil {
			t.Fatalf("GetChunk failed for %s: %v", id, err)
		}
		if retrieved != nil {
			t.Errorf("Expected nil for deleted chunk %s, got %v", id, retrieved)
		}
	}
}

func TestSQLiteStorage_ListDocuments(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Save multiple documents
	for i := 0; i < 5; i++ {
		doc := &models.Document{
			ID:          "list-doc-" + string(rune('a'+i)),
			Path:        "/test/list" + string(rune('0'+i)) + ".md",
			FileType:    "md",
			ContentHash: "hash-list",
			Status:      "indexed",
		}
		if err := db.SaveDocument(doc); err != nil {
			t.Fatalf("SaveDocument failed: %v", err)
		}
	}

	docs, err := db.ListDocuments(10, 0)
	if err != nil {
		t.Fatalf("ListDocuments failed: %v", err)
	}
	if len(docs) != 5 {
		t.Errorf("Expected 5 documents, got %d", len(docs))
	}

	// Test pagination
	docs, err = db.ListDocuments(2, 0)
	if err != nil {
		t.Fatalf("ListDocuments failed: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("Expected 2 documents with limit, got %d", len(docs))
	}
}

func TestSQLiteStorage_Metadata(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	err := db.SetMetadata("test-key", "test-value")
	if err != nil {
		t.Fatalf("SetMetadata failed: %v", err)
	}

	val, err := db.GetMetadata("test-key")
	if err != nil {
		t.Fatalf("GetMetadata failed: %v", err)
	}
	if val != "test-value" {
		t.Errorf("Expected test-value, got %s", val)
	}
}
