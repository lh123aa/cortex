package storage

import (
	"testing"
	"time"

	"github.com/lh123aa/cortex/internal/models"
)

func TestSQLiteStorage_Cache(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup: save a document and chunks
	doc := &models.Document{
		ID:          "cache-doc-test",
		UserID:      "test-user",
		Path:        "/test/cache.md",
		FileType:    "md",
		ContentHash: "cache-hash",
		Status:      "indexed",
	}
	if err := db.SaveDocument(doc); err != nil {
		t.Fatalf("SaveDocument failed: %v", err)
	}

	chunks := []*models.Chunk{
		{
			ID:           "cache-chunk-1",
			UserID:       "test-user",
			DocumentID:   "cache-doc-test",
			HeadingPath:  "Header 1",
			HeadingLevel: 1,
			Content:      "Section: Header 1\n\nCache test content.",
			ContentRaw:   "Cache test content.",
			TokenCount:   5,
		},
	}
	if err := db.SaveChunks(chunks); err != nil {
		t.Fatalf("SaveChunks failed: %v", err)
	}

	// Test: Set and get cached search
	results := []*models.SearchResult{
		{
			Chunk: chunks[0],
			Score: 0.95,
		},
	}

	err := db.SetCachedSearch("test query", "test-user", "hybrid", 10, results, 5*time.Minute)
	if err != nil {
		t.Fatalf("SetCachedSearch failed: %v", err)
	}

	cached, found := db.GetCachedSearch("test query", "test-user", "hybrid", 10)
	if !found {
		t.Fatal("Expected to find cached result")
	}
	if len(cached) != 1 {
		t.Errorf("Expected 1 cached result, got %d", len(cached))
	}
	if cached[0].Score != 0.95 {
		t.Errorf("Expected score 0.95, got %f", cached[0].Score)
	}
}

func TestSQLiteStorage_CacheInvalidation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Set cache
	results := []*models.SearchResult{}
	db.SetCachedSearch("query1", "test-user", "hybrid", 10, results, 5*time.Minute)
	db.SetCachedSearch("query2", "test-user", "fts", 5, results, 5*time.Minute)

	// Verify cache exists
	cached, found := db.GetCachedSearch("query1", "test-user", "hybrid", 10)
	if !found {
		t.Fatal("Expected to find cached result before invalidation")
	}

	// Invalidate all cache
	err := db.InvalidateSearchCache()
	if err != nil {
		t.Fatalf("InvalidateSearchCache failed: %v", err)
	}

	// Verify cache is cleared
	cached, found = db.GetCachedSearch("query1", "test-user", "hybrid", 10)
	if found {
		t.Fatal("Expected cache to be invalidated")
	}
}

func TestSQLiteStorage_CacheStats(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	results := []*models.SearchResult{}
	db.SetCachedSearch("query1", "test-user", "hybrid", 10, results, 5*time.Minute)
	db.SetCachedSearch("query2", "test-user", "fts", 5, results, 5*time.Minute)

	total, expired, avgHits, err := db.GetCacheStats()
	if err != nil {
		t.Fatalf("GetCacheStats failed: %v", err)
	}
	if total != 2 {
		t.Errorf("Expected 2 cache entries, got %d", total)
	}
	if expired != 0 {
		t.Errorf("Expected 0 expired entries, got %d", expired)
	}
	if avgHits < 0 {
		t.Errorf("Expected non-negative avgHits, got %f", avgHits)
	}
}

func TestSQLiteStorage_GetDocumentsCount(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	count, err := db.GetDocumentsCount("test-user")
	if err != nil {
		t.Fatalf("GetDocumentsCount failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 documents initially, got %d", count)
	}

	// Add document
	doc := &models.Document{
		ID:          "count-test-doc",
		UserID:      "test-user",
		Path:        "/test/count.md",
		FileType:    "md",
		ContentHash: "count-hash",
		Status:      "indexed",
	}
	db.SaveDocument(doc)

	count, err = db.GetDocumentsCount("test-user")
	if err != nil {
		t.Fatalf("GetDocumentsCount failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 document, got %d", count)
	}
}

func TestSQLiteStorage_GetChunksCount(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	count, err := db.GetChunksCount("test-user")
	if err != nil {
		t.Fatalf("GetChunksCount failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 chunks initially, got %d", count)
	}

	// Add document and chunks
	doc := &models.Document{
		ID:          "chunks-count-doc",
		UserID:      "test-user",
		Path:        "/test/chunks-count.md",
		FileType:    "md",
		ContentHash: "chunks-count-hash",
		Status:      "indexed",
	}
	db.SaveDocument(doc)

	chunks := []*models.Chunk{
		{
			ID:           "cc-chunk-1",
			UserID:       "test-user",
			DocumentID:   "chunks-count-doc",
			ContentRaw:   "Content 1",
			TokenCount:   2,
		},
		{
			ID:           "cc-chunk-2",
			UserID:       "test-user",
			DocumentID:   "chunks-count-doc",
			ContentRaw:   "Content 2",
			TokenCount:   2,
		},
	}
	db.SaveChunks(chunks)

	count, err = db.GetChunksCount("test-user")
	if err != nil {
		t.Fatalf("GetChunksCount failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 chunks, got %d", count)
	}
}

func TestSQLiteStorage_GetVectorsCount(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	count, err := db.GetVectorsCount("test-user")
	if err != nil {
		t.Fatalf("GetVectorsCount failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 vectors initially, got %d", count)
	}
}

func TestSQLiteStorage_DeleteDocument(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	doc := &models.Document{
		ID:          "delete-doc-test",
		UserID:      "test-user",
		Path:        "/test/delete-doc.md",
		FileType:    "md",
		ContentHash: "delete-doc-hash",
		Status:      "indexed",
	}
	db.SaveDocument(doc)

	// Delete document
	err := db.DeleteDocument("delete-doc-test", "test-user")
	if err != nil {
		t.Fatalf("DeleteDocument failed: %v", err)
	}

	// Verify deleted
	retrieved, err := db.GetDocumentByID("delete-doc-test", "test-user")
	if err != nil {
		t.Fatalf("GetDocumentByID failed: %v", err)
	}
	if retrieved != nil {
		t.Error("Expected nil after deletion")
	}
}

func TestSQLiteStorage_MultipleSaveDocument(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	doc := &models.Document{
		ID:          "multi-save-doc",
		UserID:      "test-user",
		Path:        "/test/multi-save.md",
		FileType:    "md",
		ContentHash: "hash-v1",
		Status:      "indexed",
	}

	// Save twice with same ID - should replace
	err := db.SaveDocument(doc)
	if err != nil {
		t.Fatalf("SaveDocument failed: %v", err)
	}

	doc.ContentHash = "hash-v2"
	err = db.SaveDocument(doc)
	if err != nil {
		t.Fatalf("SaveDocument update failed: %v", err)
	}

	retrieved, _ := db.GetDocumentByID("multi-save-doc", "test-user")
	if retrieved.ContentHash != "hash-v2" {
		t.Errorf("Expected hash-v2, got %s", retrieved.ContentHash)
	}
}