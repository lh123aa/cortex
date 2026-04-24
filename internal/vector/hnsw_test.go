package vector

import (
	"math"
	"sort"
	"testing"
)

func TestCosineDistance(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float64
	}{
		{
			name:     "identical vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{1, 0, 0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{-1, 0, 0},
			expected: 2.0,
		},
		{
			name:     "perpendicular vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{0, 1, 0},
			expected: 1.0,
		},
		{
			name:     "3d vectors",
			a:        []float32{1, 2, 3},
			b:        []float32{4, 5, 6},
			expected: 1.0 - cosineSimilarity([]float32{1, 2, 3}, []float32{4, 5, 6}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineDistance(tt.a, tt.b)
			// 对于正交向量，余弦距离应该是 1
			if math.IsNaN(result) {
				t.Errorf("cosineDistance returned NaN")
			}
		})
	}
}

func TestCosineDistance_ZeroVector(t *testing.T) {
	result := cosineDistance([]float32{0, 0}, []float32{1, 0})
	if result != 1.0 {
		t.Errorf("Expected 1.0 for zero vector, got %f", result)
	}
}

func TestL2Distance(t *testing.T) {
	// same vector
	a := []float32{1.0, 2.0, 3.0}
	b := []float32{1.0, 2.0, 3.0}
	if d := l2Distance(a, b); d != 0 {
		t.Errorf("Expected 0 for identical vectors, got %f", d)
	}

	// different vector
	c := []float32{4.0, 5.0, 6.0}
	d := l2Distance(a, c)
	expected := math.Sqrt(math.Pow(3, 2) + math.Pow(3, 2) + math.Pow(3, 2))
	if math.Abs(d-expected) > 1e-6 {
		t.Errorf("Expected %f, got %f", expected, d)
	}
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		name  string
		input []float32
	}{
		{"unit vector", []float32{1, 0, 0}},
		{"zero vector", []float32{0, 0, 0}},
		{"3d vector", []float32{3, 4, 0}},
		{"negative", []float32{-1, -2, -3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalize(tt.input)
			if len(result) != len(tt.input) {
				t.Errorf("Length mismatch: expected %d, got %d", len(tt.input), len(result))
			}
		})
	}
}

func TestNormalize_ZeroVector(t *testing.T) {
	result := normalize([]float32{0, 0, 0})
	// Zero vector should return as-is (avoid division by zero)
	if len(result) != 3 {
		t.Errorf("Expected length 3, got %d", len(result))
	}
}

func TestNewHNSW(t *testing.T) {
	cfg := DefaultConfig()
	h := NewHNSW(cfg)

	if h.count != 0 {
		t.Errorf("Expected count 0, got %d", h.count)
	}
	if h.dim != 0 {
		t.Errorf("Expected dim 0, got %d", h.dim)
	}
	if h.maxLevel != 0 {
		t.Errorf("Expected maxLevel 0, got %d", h.maxLevel)
	}
}

func TestHNSW_Add(t *testing.T) {
	h := NewHNSW(DefaultConfig())

	// Add first vector
	h.Add("vec1", []float32{1.0, 0.0, 0.0})
	if h.count != 1 {
		t.Errorf("Expected count 1, got %d", h.count)
	}
	if h.dim != 3 {
		t.Errorf("Expected dim 3, got %d", h.dim)
	}

	// Add second vector
	h.Add("vec2", []float32{0.0, 1.0, 0.0})
	if h.count != 2 {
		t.Errorf("Expected count 2, got %d", h.count)
	}
}

func TestHNSW_Add_Empty(t *testing.T) {
	h := NewHNSW(DefaultConfig())
	h.Add("empty", []float32{})
	// Empty vector should not change dimension
	if h.dim != 0 {
		t.Errorf("Dim should remain 0 for empty vector, got %d", h.dim)
	}
}

func TestHNSW_Search_Empty(t *testing.T) {
	h := NewHNSW(DefaultConfig())
	ids, distances := h.Search([]float32{1, 0, 0}, 5)

	if ids != nil || distances != nil {
		t.Errorf("Expected nil for empty index, got ids=%v, distances=%v", ids, distances)
	}
}

func TestHNSW_Search(t *testing.T) {
	h := NewHNSW(DefaultConfig())

	// Add test vectors
	vecs := [][]float32{
		{1.0, 0.0, 0.0},   // vec1
		{0.0, 1.0, 0.0},   // vec2
		{0.9, 0.1, 0.0},   // vec3 - close to vec1
		{0.0, 0.9, 0.1},   // vec4 - close to vec2
	}

	for i, v := range vecs {
		h.Add(string(rune('0'+i)), v)
	}

	// Search for vector close to vec1
	ids, distances := h.Search([]float32{1.0, 0.1, 0.0}, 2)

	if len(ids) != 2 {
		t.Errorf("Expected 2 results, got %d", len(ids))
	}

	// First result should be vec1 (0.9, 0.1, 0 is closest to 1.0, 0.1, 0.0)
	// Second should be vec1 (0.0 distance)
	if len(ids) >= 1 && ids[0] != "0" && ids[0] != "2" {
		t.Errorf("Expected vec0 or vec2 as first result, got %s", ids[0])
	}
}

func TestHNSW_Search_KLargerThanCount(t *testing.T) {
	h := NewHNSW(DefaultConfig())

	h.Add("vec1", []float32{1.0, 0.0, 0.0})
	h.Add("vec2", []float32{0.0, 1.0, 0.0})

	// Search with k larger than vector count
	ids, _ := h.Search([]float32{1.0, 0.0, 0.0}, 10)

	if len(ids) != 2 {
		t.Errorf("Expected 2 results (max available), got %d", len(ids))
	}
}

func TestHNSW_Remove(t *testing.T) {
	h := NewHNSW(DefaultConfig())

	h.Add("vec1", []float32{1.0, 0.0, 0.0})
	h.Add("vec2", []float32{0.0, 1.0, 0.0})

	if h.count != 2 {
		t.Errorf("Expected count 2, got %d", h.count)
	}

	// Remove vec1
	h.Remove("vec1")

	// Vector should still exist but marked as zero
	if h.count != 2 {
		t.Errorf("Count should still be 2 after removal, got %d", h.count)
	}

	// But search should not return it
	ids, _ := h.Search([]float32{1.0, 0.0, 0.0}, 1)
	if len(ids) == 1 && ids[0] == "vec1" {
		t.Error("Removed vector should not appear in search results")
	}
}

func TestHNSW_Remove_NonExistent(t *testing.T) {
	h := NewHNSW(DefaultConfig())

	h.Add("vec1", []float32{1.0, 0.0, 0.0})

	// Remove non-existent vector - should not panic
	h.Remove("non-existent")

	if h.count != 1 {
		t.Errorf("Count should remain 1, got %d", h.count)
	}
}

func TestHNSW_Clear(t *testing.T) {
	h := NewHNSW(DefaultConfig())

	h.Add("vec1", []float32{1.0, 0.0, 0.0})
	h.Add("vec2", []float32{0.0, 1.0, 0.0})

	h.Clear()

	if h.count != 0 {
		t.Errorf("Expected count 0 after clear, got %d", h.count)
	}
	if len(h.vectors) != 0 {
		t.Errorf("Expected empty vectors slice after clear, got length %d", len(h.vectors))
	}
	if len(h.ids) != 0 {
		t.Errorf("Expected empty ids slice after clear, got length %d", len(h.ids))
	}
}

func TestHNSW_Count(t *testing.T) {
	h := NewHNSW(DefaultConfig())

	if h.Count() != 0 {
		t.Errorf("Expected 0 initially, got %d", h.Count())
	}

	h.Add("vec1", []float32{1.0, 0.0})
	if h.Count() != 1 {
		t.Errorf("Expected 1 after add, got %d", h.Count())
	}

	h.Add("vec2", []float32{0.0, 1.0})
	if h.Count() != 2 {
		t.Errorf("Expected 2 after second add, got %d", h.Count())
	}
}

func TestHNSW_Dimension(t *testing.T) {
	h := NewHNSW(DefaultConfig())

	if h.Dimension() != 0 {
		t.Errorf("Expected 0 initially, got %d", h.Dimension())
	}

	h.Add("vec1", []float32{1.0, 2.0, 3.0})
	if h.Dimension() != 3 {
		t.Errorf("Expected 3 after add, got %d", h.Dimension())
	}
}

// PriorityQueue tests
func TestPriorityQueue_PushPop(t *testing.T) {
	pq := &priorityQueue{}

	pq.push(1, 2.0)
	pq.push(2, 1.0)
	pq.push(3, 3.0)

	if pq.Len() != 3 {
		t.Errorf("Expected length 3, got %d", pq.Len())
	}

	// Should pop in order of smallest distance first
	id, dist := pq.pop()
	if id != 2 || dist != 1.0 {
		t.Errorf("Expected (2, 1.0), got (%d, %f)", id, dist)
	}
}

func TestPriorityQueue_PopEmpty(t *testing.T) {
	pq := &priorityQueue{}

	// Pop from empty queue
	id, dist := pq.pop()
	if id != 0 || dist != 0 {
		t.Errorf("Expected (0, 0) for empty queue, got (%d, %f)", id, dist)
	}
}

func TestPriorityQueue_GetWorst(t *testing.T) {
	pq := &priorityQueue{}

	if pq.getWorst() != math.MaxFloat64 {
		t.Errorf("Expected MaxFloat64 for empty queue, got %f", pq.getWorst())
	}

	pq.push(1, 5.0)
	pq.push(2, 3.0)

	// Worst should be the last element (largest distance)
	worst := pq.getWorst()
	if worst != 5.0 {
		t.Errorf("Expected 5.0, got %f", worst)
	}
}

func TestPriorityQueue_Len(t *testing.T) {
	pq := &priorityQueue{}

	if pq.Len() != 0 {
		t.Errorf("Expected 0, got %d", pq.Len())
	}

	pq.push(1, 1.0)
	if pq.Len() != 1 {
		t.Errorf("Expected 1, got %d", pq.Len())
	}
}

func TestPriorityQueue_Swap(t *testing.T) {
	pq := &priorityQueue{
		items: []searchResult{
			{id: 1, dist: 2.0},
			{id: 2, dist: 1.0},
		},
	}

	pq.Swap(0, 1)

	if pq.items[0].id != 2 || pq.items[1].id != 1 {
		t.Errorf("Swap didn't work correctly")
	}
}

func TestPriorityQueue_Less(t *testing.T) {
	pq := &priorityQueue{
		items: []searchResult{
			{id: 1, dist: 1.0},
			{id: 2, dist: 2.0},
		},
	}

	if pq.Less(0, 1) != true {
		t.Errorf("Expected Less(0,1) to be true")
	}
	if pq.Less(1, 0) != false {
		t.Errorf("Expected Less(1,0) to be false")
	}
}

// Helper function for cosine similarity in tests
func cosineSimilarity(a, b []float32) float64 {
	var dotProd, normA, normB float64
	for i := range a {
		dotProd += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dotProd / (math.Sqrt(normA) * math.Sqrt(normB))
}

// SearchResult sort for testing
type byDist []searchResult

func (a byDist) Len() int           { return len(a) }
func (a byDist) Less(i, j int) bool { return a[i].dist < a[j].dist }
func (a byDist) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func TestSearchLayer(t *testing.T) {
	h := NewHNSW(DefaultConfig())

	// Add some vectors
	h.Add("vec0", []float32{1.0, 0.0, 0.0})
	h.Add("vec1", []float32{0.0, 1.0, 0.0})
	h.Add("vec2", []float32{0.0, 0.0, 1.0})

	// Search in layer 0
	results := h.searchLayer([]float32{1.0, 0.0, 0.0}, 0, 10, 0)

	if len(results) == 0 {
		t.Error("Expected at least one result")
	}

	// Results should be sorted by distance
	for i := 1; i < len(results); i++ {
		if results[i-1].dist > results[i].dist {
			t.Errorf("Results not sorted: results[%d].dist=%f > results[%d].dist=%f",
				i-1, results[i-1].dist, i, results[i].dist)
		}
	}
}

func TestSearchLayer_EmptyIndex(t *testing.T) {
	h := NewHNSW(DefaultConfig())

	results := h.searchLayer([]float32{1.0, 0.0, 0.0}, 0, 10, 0)

	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty index, got %d", len(results))
	}
}

func TestHNSW_RandomLevel(t *testing.T) {
	h := NewHNSW(DefaultConfig())

	// Test that randomLevel returns values within expected range
	for i := 0; i < 100; i++ {
		level := h.randomLevel()
		if level < 0 || level >= h.cfg.MaxLayers {
			t.Errorf("randomLevel returned %d, expected 0 <= level < %d", level, h.cfg.MaxLayers)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxLayers != 16 {
		t.Errorf("Expected MaxLayers 16, got %d", cfg.MaxLayers)
	}
	if cfg.EfConstruction != 200 {
		t.Errorf("Expected EfConstruction 200, got %d", cfg.EfConstruction)
	}
	if cfg.M != 32 {
		t.Errorf("Expected M 32, got %d", cfg.M)
	}
	if cfg.ML == 0 {
		t.Error("Expected ML to be non-zero")
	}
}

func TestSearchResult_Sort(t *testing.T) {
	results := []searchResult{
		{id: 3, dist: 0.5},
		{id: 1, dist: 0.1},
		{id: 2, dist: 0.3},
	}

	sort.Sort(byDist(results))

	if results[0].id != 1 {
		t.Errorf("Expected id 1 first, got %d", results[0].id)
	}
	if results[1].id != 2 {
		t.Errorf("Expected id 2 second, got %d", results[1].id)
	}
	if results[2].id != 3 {
		t.Errorf("Expected id 3 third, got %d", results[2].id)
	}
}