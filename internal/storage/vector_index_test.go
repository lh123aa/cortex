//go:build cgo

package storage

import (
	"math"
	"os"
	"testing"

	"github.com/lh123aa/cortex/internal/models"
)

func setupVectorTestDB(t *testing.T) *SQLiteStorage {
	t.Helper()
	s := newTestDB(t)

	doc := &models.Document{
		ID:          "vec-test-doc",
		UserID:      "user-1",
		Path:        "/test/vectors.md",
		FileType:    "md",
		ContentHash: "vec-test-hash",
		Status:      "indexed",
	}
	if err := s.SaveDocument(doc); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	chunks := []*models.Chunk{
		{ID: "chunk-a", UserID: "user-1", DocumentID: "vec-test-doc", ContentRaw: "vector A content about Go", ContentHash: "hash-a"},
		{ID: "chunk-b", UserID: "user-1", DocumentID: "vec-test-doc", ContentRaw: "vector B content about Rust", ContentHash: "hash-b"},
		{ID: "chunk-c", UserID: "user-1", DocumentID: "vec-test-doc", ContentRaw: "vector C content about Python", ContentHash: "hash-c"},
		{ID: "chunk-d", UserID: "user-2", DocumentID: "vec-test-doc", ContentRaw: "vector D other user", ContentHash: "hash-d"},
	}
	if err := s.SaveChunks(chunks); err != nil {
		t.Fatalf("SaveChunks: %v", err)
	}

	dim := 4
	vecData := [][]float32{
		{1.0, 0.0, 0.0, 0.0},
		{0.0, 1.0, 0.0, 0.0},
		{0.0, 0.0, 1.0, 0.0},
		{1.0, 1.0, 0.0, 0.0},
	}
	for i, chunkID := range []string{"chunk-a", "chunk-b", "chunk-c", "chunk-d"} {
		userID := "user-1"
		if i == 3 {
			userID = "user-2"
		}
		embBytes := Float32ArrayToBytes(vecData[i])
		_, err := s.db.Exec(
			`INSERT INTO vectors (chunk_id, user_id, embedding, dimension, model) VALUES (?, ?, ?, ?, ?)`,
			chunkID, userID, embBytes, dim, "test-model",
		)
		if err != nil {
			t.Fatalf("insert vector %s: %v", chunkID, err)
		}
	}

	return s
}

func TestBuildHNSWIndex_CreateAndSearch(t *testing.T) {
	s := setupVectorTestDB(t)

	if err := s.BuildHNSWIndex(); err != nil {
		t.Fatalf("BuildHNSWIndex: %v", err)
	}

	if !s.flatReady {
		t.Fatal("flatReady should be true after BuildHNSWIndex")
	}
	if len(s.flatChunkIDs) != 4 {
		t.Fatalf("expected 4 chunks, got %d", len(s.flatChunkIDs))
	}
	if s.flatDim != 4 {
		t.Fatalf("expected dim=4, got %d", s.flatDim)
	}
	if len(s.flatData) != 16 {
		t.Fatalf("expected 16 flat values (4*4), got %d", len(s.flatData))
	}

	queryVec := []float32{1.0, 0.0, 0.0, 0.0}
	results, err := s.VectorSearch(queryVec, "user-1", 2)
	if err != nil {
		t.Fatalf("VectorSearch: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 result")
	}
	if results[0].Chunk.ID != "chunk-a" {
		t.Fatalf("expected chunk-a as top result (cosine 1.0), got %s", results[0].Chunk.ID)
	}
	if math.Abs(results[0].Score-1.0) > 0.001 {
		t.Fatalf("expected score ~1.0, got %f", results[0].Score)
	}
}

func TestBuildHNSWIndex_SaveLoadBinary(t *testing.T) {
	s := setupVectorTestDB(t)

	if err := s.BuildHNSWIndex(); err != nil {
		t.Fatalf("BuildHNSWIndex: %v", err)
	}

	tmpIdx, err := os.CreateTemp("", "cortex-test-*.idx")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	origPath := s.flatPath
	s.flatPath = tmpIdx.Name()
	tmpIdx.Close()

	if err := s.saveFlatVectorIndex(); err != nil {
		t.Fatalf("saveFlatVectorIndex: %v", err)
	}

	s2 := newTestDB(t)
	s2.flatPath = s.flatPath
	if err := s2.loadFlatVectorIndex(); err != nil {
		t.Fatalf("loadFlatVectorIndex: %v", err)
	}

	if !s2.flatReady {
		t.Fatal("flatReady should be true after load")
	}
	if len(s2.flatChunkIDs) != 4 {
		t.Fatalf("expected 4 chunk IDs after load, got %d", len(s2.flatChunkIDs))
	}
	if s2.flatDim != 4 {
		t.Fatalf("expected dim=4 after load, got %d", s2.flatDim)
	}
	if s2.flatChunkIDs[0] != "chunk-a" {
		t.Fatalf("expected chunk-a first, got %s", s2.flatChunkIDs[0])
	}
	if math.Abs(float64(s2.flatData[0]-1.0)) > 0.001 {
		t.Fatalf("expected flatData[0]=1.0, got %f", s2.flatData[0])
	}

	s.flatPath = origPath
	os.Remove(s.flatPath)
}

func TestBuildHNSWIndex_EmptyDB(t *testing.T) {
	s := newTestDB(t)

	if err := s.BuildHNSWIndex(); err != nil {
		t.Fatalf("BuildHNSWIndex on empty DB: %v", err)
	}

	if len(s.flatChunkIDs) != 0 {
		t.Fatalf("expected 0 chunks for empty DB, got %d", len(s.flatChunkIDs))
	}
	if s.flatReady {
		t.Log("flatReady is true after empty index build (expected, harmless)")
	}
}

func TestVectorSearchInMemory_UserIsolation(t *testing.T) {
	s := setupVectorTestDB(t)

	if err := s.BuildHNSWIndex(); err != nil {
		t.Fatalf("BuildHNSWIndex: %v", err)
	}

	queryVec := []float32{1.0, 0.0, 0.0, 0.0}
	results, err := s.VectorSearch(queryVec, "user-2", 10)
	if err != nil {
		t.Fatalf("VectorSearch: %v", err)
	}
	for _, r := range results {
		if r.Chunk.UserID != "user-2" {
			t.Fatalf("expected user-2 results only, got chunk %s with userID %s", r.Chunk.ID, r.Chunk.UserID)
		}
	}
}

func TestVectorSearchInMemory_ExactTopK(t *testing.T) {
	s := setupVectorTestDB(t)

	if err := s.BuildHNSWIndex(); err != nil {
		t.Fatalf("BuildHNSWIndex: %v", err)
	}

	queryVec := []float32{1.0, 0.0, 0.0, 0.0}
	for k := 1; k <= 3; k++ {
		results, err := s.VectorSearch(queryVec, "user-1", k)
		if err != nil {
			t.Fatalf("VectorSearch k=%d: %v", k, err)
		}
		if len(results) != k {
			t.Fatalf("expected %d results for k=%d, got %d", k, k, len(results))
		}
	}
}
