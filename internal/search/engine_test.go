package search

import (
	"math"
	"testing"
	"time"

	"github.com/lh123aa/cortex/internal/models"
)

func TestRRFMerge(t *testing.T) {
	engine := &HybridSearchEngine{}

	// Mock vector results
	vr := []*models.SearchResult{
		{Chunk: &models.Chunk{ID: "chunk-1"}, Score: 0.9},
		{Chunk: &models.Chunk{ID: "chunk-2"}, Score: 0.8},
		{Chunk: &models.Chunk{ID: "chunk-3"}, Score: 0.7},
	}

	// Mock FTS results
	fr := []*models.SearchResult{
		{Chunk: &models.Chunk{ID: "chunk-2"}, Score: 10.5},
		{Chunk: &models.Chunk{ID: "chunk-4"}, Score: 8.0},
		{Chunk: &models.Chunk{ID: "chunk-1"}, Score: 5.0},
	}

	merged := engine.rrfMerge(vr, fr)

	if len(merged) != 4 {
		t.Fatalf("Expected 4 merged results, got %d", len(merged))
	}

	// Because chunk-2 has rank 2 in VR and rank 1 in FR
	// chunk-1 has rank 1 in VR and rank 3 in FR
	// Both chunk-2 and chunk-1 should be at the top. Let's check normalization.

	if merged[0].Score != 1.0 {
		t.Errorf("Top result score should be normalized to 1.0, got %f", merged[0].Score)
	}

	// Ensure sorted order
	for i := 0; i < len(merged)-1; i++ {
		if merged[i].Score < merged[i+1].Score {
			t.Errorf("Results are not strictly descending at index %d and %d", i, i+1)
		}
	}
}

func TestRRFMerge_EmptyInputs(t *testing.T) {
	engine := &HybridSearchEngine{}

	tests := []struct {
		name string
		vr   []*models.SearchResult
		fr   []*models.SearchResult
	}{
		{"empty vr", []*models.SearchResult{}, []*models.SearchResult{{Chunk: &models.Chunk{ID: "1"}, Score: 1.0}}},
		{"empty fr", []*models.SearchResult{{Chunk: &models.Chunk{ID: "1"}, Score: 1.0}}, []*models.SearchResult{}},
		{"both empty", []*models.SearchResult{}, []*models.SearchResult{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged := engine.rrfMerge(tt.vr, tt.fr)
			if merged == nil {
				t.Error("Expected non-nil result")
			}
		})
	}
}

func TestRRFMerge_DuplicateChunks(t *testing.T) {
	engine := &HybridSearchEngine{}

	// Same chunk appears in both result sets
	vr := []*models.SearchResult{
		{Chunk: &models.Chunk{ID: "chunk-1"}, Score: 0.9},
	}
	fr := []*models.SearchResult{
		{Chunk: &models.Chunk{ID: "chunk-1"}, Score: 0.8},
	}

	merged := engine.rrfMerge(vr, fr)

	// Should only have one result for chunk-1
	if len(merged) != 1 {
		t.Errorf("Expected 1 result, got %d", len(merged))
	}
}

func TestNormalizeScores(t *testing.T) {
	engine := &HybridSearchEngine{}
	res := []*models.SearchResult{
		{Score: 2.0},
		{Score: 0.5},
		{Score: 10.0},
	}

	normalized := engine.normalizeScores(res)

	// Since it sorts descending and max is 10.0
	if math.Abs(normalized[0].Score-1.0) > 1e-9 {
		t.Errorf("Expected max score to be 1.0, got %f", normalized[0].Score)
	}
	if math.Abs(normalized[1].Score-0.2) > 1e-9 {
		t.Errorf("Expected second score to be 0.2, got %f", normalized[1].Score)
	}
	if math.Abs(normalized[2].Score-0.05) > 1e-9 {
		t.Errorf("Expected lowest score to be 0.05, got %f", normalized[2].Score)
	}
}

func TestNormalizeScores_EmptyInput(t *testing.T) {
	engine := &HybridSearchEngine{}
	res := []*models.SearchResult{}

	normalized := engine.normalizeScores(res)
	if normalized != nil {
		t.Errorf("Expected nil for empty input, got %v", normalized)
	}
}

func TestNormalizeScores_AllSameScore(t *testing.T) {
	engine := &HybridSearchEngine{}
	res := []*models.SearchResult{
		{Score: 5.0},
		{Score: 5.0},
		{Score: 5.0},
	}

	normalized := engine.normalizeScores(res)

	for _, r := range normalized {
		if r.Score != 1.0 {
			t.Errorf("Expected all scores to be 1.0, got %f", r.Score)
		}
	}
}

func TestNormalizeScores_ZeroMaxScore(t *testing.T) {
	engine := &HybridSearchEngine{}
	res := []*models.SearchResult{
		{Score: 0.0},
		{Score: 0.0},
	}

	normalized := engine.normalizeScores(res)

	// Should not panic and should return zeros
	for _, r := range normalized {
		if r.Score != 0.0 {
			t.Errorf("Expected zero score, got %f", r.Score)
		}
	}
}

func TestHybridSearchEngine_SetCacheTTL(t *testing.T) {
	engine := &HybridSearchEngine{}

	if engine.cacheTTL != 5*time.Minute {
		t.Errorf("Expected default TTL of 5 minutes, got %v", engine.cacheTTL)
	}

	engine.SetCacheTTL(10 * time.Minute)

	if engine.cacheTTL != 10*time.Minute {
		t.Errorf("Expected TTL of 10 minutes, got %v", engine.cacheTTL)
	}
}

func TestHybridSearchEngine_DisableCache(t *testing.T) {
	engine := &HybridSearchEngine{}

	if !engine.useCache {
		t.Error("Expected cache to be enabled by default")
	}

	engine.DisableCache()

	if engine.useCache {
		t.Error("Expected cache to be disabled")
	}
}

func TestHybridSearchEngine_NewHybridSearchEngine(t *testing.T) {
	engine := NewHybridSearchEngine(nil, nil)

	if engine == nil {
		t.Fatal("Expected non-nil engine")
	}
	if !engine.useCache {
		t.Error("Expected cache to be enabled by default")
	}
	if engine.cacheTTL != 5*time.Minute {
		t.Errorf("Expected default TTL of 5 minutes, got %v", engine.cacheTTL)
	}
}
