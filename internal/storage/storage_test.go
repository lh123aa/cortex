package storage

import (
	"os"
	"testing"
	"time"

	"github.com/lh123aa/cortex/internal/models"
)

// setupTestDB 创建一个测试用临时数据库
func setupTestDB(t *testing.T) (*SQLiteStorage, func()) {
	tmpFile := t.TempDir() + "/test.db"
	storage, err := NewSQLiteStorage(tmpFile)
	if err != nil {
		t.Fatalf("failed to create test storage: %v", err)
	}
	cleanup := func() {
		storage.Close()
		os.Remove(tmpFile)
	}
	return storage, cleanup
}

func TestSaveDocument(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	doc := &models.Document{
		ID:          "test-doc-1",
		UserID:      "user-1",
		Path:        "/test/doc1.md",
		Title:       "Test Document",
		FileType:    "md",
		ContentHash: "abc123",
		ChunkCount:  3,
		Status:      "indexed",
		IndexedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := storage.SaveDocument(doc)
	if err != nil {
		t.Fatalf("SaveDocument failed: %v", err)
	}

	// 验证保存
	retrieved, err := storage.GetDocumentByID("test-doc-1", "user-1")
	if err != nil {
		t.Fatalf("GetDocumentByID failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Document not found")
	}
	if retrieved.Title != "Test Document" {
		t.Errorf("Expected title 'Test Document', got '%s'", retrieved.Title)
	}
}

func TestGetDocumentByID_UserIsolation(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	doc := &models.Document{
		ID:          "test-doc-2",
		UserID:      "user-1",
		Path:        "/test/doc2.md",
		ContentHash: "def456",
		Status:      "indexed",
	}

	err := storage.SaveDocument(doc)
	if err != nil {
		t.Fatalf("SaveDocument failed: %v", err)
	}

	// User-1 应该能访问
	retrieved, err := storage.GetDocumentByID("test-doc-2", "user-1")
	if err != nil {
		t.Fatalf("GetDocumentByID failed: %v", err)
	}
	if retrieved == nil {
		t.Error("User-1 should be able to access the document")
	}

	// User-2 不应该能访问
	retrieved, err = storage.GetDocumentByID("test-doc-2", "user-2")
	if err != nil {
		t.Fatalf("GetDocumentByID failed: %v", err)
	}
	if retrieved != nil {
		t.Error("User-2 should NOT be able to access User-1's document")
	}
}

func TestDeleteDocument(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	doc := &models.Document{
		ID:          "test-doc-3",
		UserID:      "user-1",
		Path:        "/test/doc3.md",
		ContentHash: "ghi789",
		Status:      "indexed",
	}

	err := storage.SaveDocument(doc)
	if err != nil {
		t.Fatalf("SaveDocument failed: %v", err)
	}

	// 验证存在
	retrieved, _ := storage.GetDocumentByID("test-doc-3", "user-1")
	if retrieved == nil {
		t.Fatal("Document should exist before deletion")
	}

	// 删除
	err = storage.DeleteDocument("test-doc-3", "user-1")
	if err != nil {
		t.Fatalf("DeleteDocument failed: %v", err)
	}

	// 验证已删除
	retrieved, _ = storage.GetDocumentByID("test-doc-3", "user-1")
	if retrieved != nil {
		t.Error("Document should not exist after deletion")
	}
}

func TestSaveChunks(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	// 先保存一个文档
	doc := &models.Document{
		ID:          "test-doc-4",
		UserID:      "user-1",
		Path:        "/test/doc4.md",
		ContentHash: "jkl012",
		Status:      "indexed",
	}
	err := storage.SaveDocument(doc)
	if err != nil {
		t.Fatalf("SaveDocument failed: %v", err)
	}

	// 保存 chunks
	chunks := []*models.Chunk{
		{
			ID:           "chunk-1",
			UserID:       "user-1",
			DocumentID:   "test-doc-4",
			HeadingPath:  "Introduction",
			Content:      "This is the introduction",
			ContentRaw:    "This is the introduction",
			TokenCount:   5,
		},
		{
			ID:           "chunk-2",
			UserID:       "user-1",
			DocumentID:   "test-doc-4",
			HeadingPath:  "Chapter 1",
			Content:      "This is chapter 1",
			ContentRaw:    "This is chapter 1",
			TokenCount:   4,
		},
	}

	err = storage.SaveChunks(chunks)
	if err != nil {
		t.Fatalf("SaveChunks failed: %v", err)
	}

	// 验证 chunks 数量
	count, err := storage.GetChunksCount("user-1")
	if err != nil {
		t.Fatalf("GetChunksCount failed: %v", err)
	}
	if count < 2 {
		t.Errorf("Expected at least 2 chunks, got %d", count)
	}
}

func TestGetChunksCount(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	// 无用户隔离 - 初始应该为 0
	count, err := storage.GetChunksCount("")
	if err != nil {
		t.Fatalf("GetChunksCount failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 chunks, got %d", count)
	}

	// 保存文档和 chunks
	doc := &models.Document{
		ID:          "test-doc-5",
		UserID:      "user-1",
		Path:        "/test/doc5.md",
		ContentHash: "mno345",
		Status:      "indexed",
	}
	storage.SaveDocument(doc)

	chunks := []*models.Chunk{
		{ID: "chunk-5-1", UserID: "user-1", DocumentID: "test-doc-5", Content: "Content 1", ContentRaw: "Content 1"},
		{ID: "chunk-5-2", UserID: "user-1", DocumentID: "test-doc-5", Content: "Content 2", ContentRaw: "Content 2"},
	}
	storage.SaveChunks(chunks)

	// 验证计数
	count, err = storage.GetChunksCount("user-1")
	if err != nil {
		t.Fatalf("GetChunksCount failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 chunks, got %d", count)
	}
}

func TestGetVectorsCount(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	// 初始应该为 0
	count, err := storage.GetVectorsCount("")
	if err != nil {
		t.Fatalf("GetVectorsCount failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 vectors, got %d", count)
	}
}

func TestUserCRUD(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	// 创建用户
	user := &models.User{
		ID:           "user-test-1",
		Username:     "testuser",
		PasswordHash: "hashed_password",
		Role:         "user",
		CreatedAt:    time.Now(),
		IsActive:     true,
	}

	err := storage.SaveUser(user)
	if err != nil {
		t.Fatalf("SaveUser failed: %v", err)
	}

	// 获取用户
	retrieved, err := storage.GetUserByID("user-test-1")
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("User not found")
	}
	if retrieved.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", retrieved.Username)
	}

	// 按用户名获取
	retrieved, err = storage.GetUserByUsername("testuser")
	if err != nil {
		t.Fatalf("GetUserByUsername failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("User not found by username")
	}

	// 删除用户
	err = storage.DeleteUser("user-test-1")
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	// 验证已删除
	retrieved, _ = storage.GetUserByID("user-test-1")
	if retrieved != nil {
		t.Error("User should not exist after deletion")
	}
}

func TestTokenCRUD(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	// 先创建用户
	user := &models.User{
		ID:           "user-token-test",
		Username:     "tokenuser",
		PasswordHash: "hashed",
		Role:         "user",
		CreatedAt:    time.Now(),
		IsActive:     true,
	}
	storage.SaveUser(user)

	// 创建 token
	token := &models.AuthToken{
		Token:     "test-token-123",
		UserID:    "user-token-test",
		Username:  "tokenuser",
		ExpiresIn: 3600,
	}

	err := storage.SaveToken(token)
	if err != nil {
		t.Fatalf("SaveToken failed: %v", err)
	}

	// 获取 token
	retrieved, err := storage.GetToken("test-token-123")
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Token not found")
	}
	if retrieved.Username != "tokenuser" {
		t.Errorf("Expected username 'tokenuser', got '%s'", retrieved.Username)
	}

	// 删除 token
	err = storage.DeleteToken("test-token-123")
	if err != nil {
		t.Fatalf("DeleteToken failed: %v", err)
	}

	// 验证已删除
	retrieved, _ = storage.GetToken("test-token-123")
	if retrieved != nil {
		t.Error("Token should not exist after deletion")
	}
}

func TestCacheOperations(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	results := []*models.SearchResult{
		{
			Score: 0.95,
			Chunk: &models.Chunk{ID: "chunk-1", Content: "Test content"},
		},
	}

	// 写入缓存
	err := storage.SetCachedSearch("test query", "user-1", "hybrid", 10, results, 5*time.Minute)
	if err != nil {
		t.Fatalf("SetCachedSearch failed: %v", err)
	}

	// 读取缓存
	cached, found := storage.GetCachedSearch("test query", "user-1", "hybrid", 10)
	if !found {
		t.Fatal("Cache miss - expected cache hit")
	}
	if len(cached) != 1 {
		t.Errorf("Expected 1 result, got %d", len(cached))
	}
	if cached[0].Score != 0.95 {
		t.Errorf("Expected score 0.95, got %f", cached[0].Score)
	}

	// 失效缓存
	err = storage.InvalidateSearchCache()
	if err != nil {
		t.Fatalf("InvalidateSearchCache failed: %v", err)
	}

	// 验证缓存已清空
	cached, found = storage.GetCachedSearch("test query", "user-1", "hybrid", 10)
	if found {
		t.Error("Cache should be empty after invalidation")
	}
}

func TestMetadataOperations(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	// 设置元数据
	err := storage.SetMetadata("last_index_time", "2024-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("SetMetadata failed: %v", err)
	}

	// 获取元数据
	value, err := storage.GetMetadata("last_index_time")
	if err != nil {
		t.Fatalf("GetMetadata failed: %v", err)
	}
	if value != "2024-01-01T00:00:00Z" {
		t.Errorf("Expected '2024-01-01T00:00:00Z', got '%s'", value)
	}
}
