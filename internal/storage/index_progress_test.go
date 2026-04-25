//go:build cgo

package storage

import (
	"testing"
	"time"

	"github.com/lh123aa/cortex/internal/models"
)

func setupTestDBForIndexProgress(t *testing.T) (*SQLiteStorage, func()) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	db, err := NewSQLiteStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

func TestSaveAndGetIndexProgress(t *testing.T) {
	db, cleanup := setupTestDBForIndexProgress(t)
	defer cleanup()

	progress := &models.IndexProgress{
		RootPath:      "/test/path",
		LastFilePath:  "/test/path/file3.md",
		LastFileIndex: 5,
		TotalFiles:    100,
		IndexedFiles:  50,
		IndexedChunks: 200,
		FailedFiles:   2,
		Status:        "running",
		StartedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Save progress
	err := db.SaveIndexProgress(progress)
	if err != nil {
		t.Fatalf("SaveIndexProgress failed: %v", err)
	}

	// Get progress
	saved, err := db.GetIndexProgress("/test/path")
	if err != nil {
		t.Fatalf("GetIndexProgress failed: %v", err)
	}

	if saved == nil {
		t.Fatal("Expected saved progress, got nil")
	}

	if saved.RootPath != "/test/path" {
		t.Errorf("Expected RootPath '/test/path', got '%s'", saved.RootPath)
	}
	if saved.TotalFiles != 100 {
		t.Errorf("Expected TotalFiles 100, got %d", saved.TotalFiles)
	}
	if saved.IndexedFiles != 50 {
		t.Errorf("Expected IndexedFiles 50, got %d", saved.IndexedFiles)
	}
	if saved.Status != "running" {
		t.Errorf("Expected Status 'running', got '%s'", saved.Status)
	}
}

func TestSaveIndexProgress_Update(t *testing.T) {
	db, cleanup := setupTestDBForIndexProgress(t)
	defer cleanup()

	// Save initial progress
	progress := &models.IndexProgress{
		RootPath:      "/test/path",
		LastFilePath:  "/test/path/file1.md",
		LastFileIndex: 1,
		TotalFiles:    100,
		IndexedFiles:  10,
		Status:        "running",
		StartedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err := db.SaveIndexProgress(progress)
	if err != nil {
		t.Fatalf("SaveIndexProgress failed: %v", err)
	}

	// Update progress
	progress.LastFileIndex = 5
	progress.IndexedFiles = 50
	progress.UpdatedAt = time.Now()

	err = db.SaveIndexProgress(progress)
	if err != nil {
		t.Fatalf("SaveIndexProgress update failed: %v", err)
	}

	// Verify update
	saved, err := db.GetIndexProgress("/test/path")
	if err != nil {
		t.Fatalf("GetIndexProgress failed: %v", err)
	}

	if saved.LastFileIndex != 5 {
		t.Errorf("Expected LastFileIndex 5, got %d", saved.LastFileIndex)
	}
	if saved.IndexedFiles != 50 {
		t.Errorf("Expected IndexedFiles 50, got %d", saved.IndexedFiles)
	}
}

func TestGetIndexProgress_NotFound(t *testing.T) {
	db, cleanup := setupTestDBForIndexProgress(t)
	defer cleanup()

	saved, err := db.GetIndexProgress("/non/existent/path")
	if err != nil {
		t.Fatalf("GetIndexProgress failed unexpectedly: %v", err)
	}

	if saved != nil {
		t.Error("Expected nil for non-existent path")
	}
}

func TestGetIndexProgress_OnlyReturnsRunning(t *testing.T) {
	db, cleanup := setupTestDBForIndexProgress(t)
	defer cleanup()

	// Save a completed progress
	completed := &models.IndexProgress{
		RootPath:      "/completed/path",
		LastFileIndex: 100,
		TotalFiles:    100,
		IndexedFiles:  100,
		Status:        "completed",
		StartedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		CompletedAt:   time.Now(),
	}

	db.SaveIndexProgress(completed)

	// Save a running progress
	running := &models.IndexProgress{
		RootPath:      "/running/path",
		LastFileIndex: 50,
		TotalFiles:    100,
		IndexedFiles:  50,
		Status:        "running",
		StartedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	db.SaveIndexProgress(running)

	// GetIndexProgress should only return running one
	saved, err := db.GetIndexProgress("/running/path")
	if err != nil {
		t.Fatalf("GetIndexProgress failed: %v", err)
	}

	if saved == nil {
		t.Fatal("Expected to find running progress")
	}
	if saved.Status != "running" {
		t.Errorf("Expected Status 'running', got '%s'", saved.Status)
	}
}

func TestCompleteIndexProgress(t *testing.T) {
	db, cleanup := setupTestDBForIndexProgress(t)
	defer cleanup()

	// Save running progress
	progress := &models.IndexProgress{
		RootPath:      "/to/complete",
		LastFileIndex: 100,
		TotalFiles:    100,
		IndexedFiles:  100,
		Status:        "running",
		StartedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	db.SaveIndexProgress(progress)

	// Complete it
	err := db.CompleteIndexProgress("/to/complete")
	if err != nil {
		t.Fatalf("CompleteIndexProgress failed: %v", err)
	}

	// Get should now return nil (only returns running)
	saved, err := db.GetIndexProgress("/to/complete")
	if err != nil {
		t.Fatalf("GetIndexProgress failed: %v", err)
	}

	if saved != nil {
		t.Error("Expected nil after completion (only returns running)")
	}
}

func TestFailIndexProgress(t *testing.T) {
	db, cleanup := setupTestDBForIndexProgress(t)
	defer cleanup()

	// Save running progress
	progress := &models.IndexProgress{
		RootPath:      "/to/fail",
		LastFileIndex: 50,
		TotalFiles:    100,
		IndexedFiles:  50,
		Status:        "running",
		StartedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	db.SaveIndexProgress(progress)

	// Fail it
	err := db.FailIndexProgress("/to/fail", "disk full")
	if err != nil {
		t.Fatalf("FailIndexProgress failed: %v", err)
	}

	// Get should now return nil (only returns running)
	saved, err := db.GetIndexProgress("/to/fail")
	if err != nil {
		t.Fatalf("GetIndexProgress failed: %v", err)
	}

	if saved != nil {
		t.Error("Expected nil after failure (only returns running)")
	}
}

func TestListIndexProgress(t *testing.T) {
	db, cleanup := setupTestDBForIndexProgress(t)
	defer cleanup()

	// Create multiple progress entries
	for i := 0; i < 5; i++ {
		progress := &models.IndexProgress{
			RootPath:      "/test/path" + string(rune('0'+i)),
			LastFileIndex: i * 10,
			TotalFiles:    100,
			IndexedFiles:  i * 10,
			Status:        "running",
			StartedAt:     time.Now(),
			UpdatedAt:     time.Now().Add(time.Duration(i) * time.Minute), // different times
		}
		db.SaveIndexProgress(progress)
	}

	// List all (limit=10, offset=0)
	results, err := db.ListIndexProgress(10, 0)
	if err != nil {
		t.Fatalf("ListIndexProgress failed: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 results, got %d", len(results))
	}

	// Should be ordered by updated_at DESC
	for i := 0; i < len(results)-1; i++ {
		if results[i].UpdatedAt.Before(results[i+1].UpdatedAt) {
			t.Errorf("Results not ordered by UpdatedAt DESC")
		}
	}
}

func TestListIndexProgress_Pagination(t *testing.T) {
	db, cleanup := setupTestDBForIndexProgress(t)
	defer cleanup()

	// Create 10 progress entries
	for i := 0; i < 10; i++ {
		progress := &models.IndexProgress{
			RootPath:      "/paginated/" + string(rune('0'+i)),
			LastFileIndex: i,
			TotalFiles:    100,
			IndexedFiles:  i,
			Status:        "running",
			StartedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		db.SaveIndexProgress(progress)
	}

	// Test offset=0, limit=5
	results, err := db.ListIndexProgress(5, 0)
	if err != nil {
		t.Fatalf("ListIndexProgress failed: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 results, got %d", len(results))
	}

	// Test offset=5, limit=5
	results2, err := db.ListIndexProgress(5, 5)
	if err != nil {
		t.Fatalf("ListIndexProgress failed: %v", err)
	}

	if len(results2) != 5 {
		t.Errorf("Expected 5 results for page 2, got %d", len(results2))
	}

	// Test offset=10 (beyond data)
	results3, err := db.ListIndexProgress(5, 10)
	if err != nil {
		t.Fatalf("ListIndexProgress failed: %v", err)
	}

	if len(results3) != 0 {
		t.Errorf("Expected 0 results for offset beyond data, got %d", len(results3))
	}
}

func TestDeleteIndexProgress(t *testing.T) {
	db, cleanup := setupTestDBForIndexProgress(t)
	defer cleanup()

	// Save progress
	progress := &models.IndexProgress{
		ID:            0, // auto-increment
		RootPath:      "/to/delete",
		LastFileIndex: 50,
		TotalFiles:    100,
		Status:        "running",
		StartedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	db.SaveIndexProgress(progress)

	// Get to find the ID
	saved, _ := db.GetIndexProgress("/to/delete")
	if saved == nil {
		t.Fatal("Expected to find saved progress")
	}
	id := saved.ID

	// Delete
	err := db.DeleteIndexProgress(id)
	if err != nil {
		t.Fatalf("DeleteIndexProgress failed: %v", err)
	}

	// Should no longer exist
	saved2, _ := db.GetIndexProgress("/to/delete")
	if saved2 != nil {
		t.Error("Expected progress to be deleted")
	}
}

func TestIndexProgress_CRUD_Integration(t *testing.T) {
	db, cleanup := setupTestDBForIndexProgress(t)
	defer cleanup()

	// Create
	progress := &models.IndexProgress{
		RootPath:      "/integration/test",
		LastFilePath:  "/integration/test/file.md",
		LastFileIndex: 0,
		TotalFiles:    50,
		IndexedFiles:  0,
		IndexedChunks: 0,
		FailedFiles:   0,
		Status:        "running",
		StartedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err := db.SaveIndexProgress(progress)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Read
	saved, err := db.GetIndexProgress("/integration/test")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if saved.IndexedChunks != 0 {
		t.Errorf("Expected IndexedChunks 0, got %d", saved.IndexedChunks)
	}

	// Update
	progress.IndexedFiles = 25
	progress.IndexedChunks = 100
	progress.LastFileIndex = 25
	progress.UpdatedAt = time.Now()

	err = db.SaveIndexProgress(progress)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update
	saved, _ = db.GetIndexProgress("/integration/test")
	if saved.IndexedFiles != 25 {
		t.Errorf("Expected IndexedFiles 25, got %d", saved.IndexedFiles)
	}
	if saved.IndexedChunks != 100 {
		t.Errorf("Expected IndexedChunks 100, got %d", saved.IndexedChunks)
	}

	// Complete
	err = db.CompleteIndexProgress("/integration/test")
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	// Verify completion
	_, err = db.GetIndexProgress("/integration/test")
	// Should not find since only running are returned
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestIndexProgress_Model(t *testing.T) {
	now := time.Now()

	progress := &models.IndexProgress{
		ID:            1,
		RootPath:      "/test",
		LastFilePath:  "/test/file.md",
		LastFileIndex: 10,
		TotalFiles:    100,
		IndexedFiles:  50,
		IndexedChunks: 200,
		FailedFiles:   2,
		Status:        "running",
		StartedAt:     now,
		UpdatedAt:     now,
		CompletedAt:   time.Time{}, // zero time
		ErrorMessage:  "",
	}

	if progress.ID != 1 {
		t.Errorf("Expected ID 1, got %d", progress.ID)
	}
	if progress.RootPath != "/test" {
		t.Errorf("Expected RootPath '/test', got '%s'", progress.RootPath)
	}
	if progress.Status != "running" {
		t.Errorf("Expected Status 'running', got '%s'", progress.Status)
	}
	if !progress.CompletedAt.IsZero() {
		t.Error("Expected CompletedAt to be zero time")
	}
}

func TestIndexProgress_WithCompletedAt(t *testing.T) {
	now := time.Now()
	completedAt := now.Add(5 * time.Minute)

	progress := &models.IndexProgress{
		ID:            2,
		RootPath:      "/completed",
		LastFileIndex: 100,
		TotalFiles:    100,
		IndexedFiles:  100,
		Status:        "completed",
		StartedAt:     now,
		UpdatedAt:     completedAt,
		CompletedAt:   completedAt,
	}

	if progress.Status != "completed" {
		t.Errorf("Expected Status 'completed', got '%s'", progress.Status)
	}
	if progress.CompletedAt != completedAt {
		t.Errorf("Expected CompletedAt %v, got %v", completedAt, progress.CompletedAt)
	}
}

func TestIndexProgress_WithError(t *testing.T) {
	progress := &models.IndexProgress{
		ID:            3,
		RootPath:      "/failed",
		LastFileIndex: 50,
		TotalFiles:    100,
		IndexedFiles:  50,
		FailedFiles:   10,
		Status:        "failed",
		StartedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		ErrorMessage:  "disk full",
	}

	if progress.Status != "failed" {
		t.Errorf("Expected Status 'failed', got '%s'", progress.Status)
	}
	if progress.ErrorMessage != "disk full" {
		t.Errorf("Expected ErrorMessage 'disk full', got '%s'", progress.ErrorMessage)
	}
}
