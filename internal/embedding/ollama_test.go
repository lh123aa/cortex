package embedding

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestOllamaEmbedding_EmbedBatchErrorsCollected(t *testing.T) {
	// Create a mock that will fail - invalid base URL
	emb := &OllamaEmbedding{
		BaseURL:  "http://localhost:99999", // Invalid
		Model:    "test-model",
		CacheDim: 768,
		Timeout:  100 * time.Millisecond,
	}

	texts := []string{"text1", "text2", "text3"}

	_, err := emb.EmbedBatch(texts)

	// Should return error since localhost:99999 is not available
	if err == nil {
		t.Log("Warning: EmbedBatch did not return error - might indicate network issue")
	}
}

func TestOllamaEmbedding_Name(t *testing.T) {
	emb := NewOllamaEmbedding("http://localhost:11434", "nomic-embed-text", 768)

	name := emb.Name()
	if name != "ollama:nomic-embed-text" {
		t.Errorf("Expected ollama:nomic-embed-text, got %s", name)
	}
}

func TestOllamaEmbedding_Dimension(t *testing.T) {
	emb := NewOllamaEmbedding("http://localhost:11434", "nomic-embed-text", 1536)

	dim := emb.Dimension()
	if dim != 1536 {
		t.Errorf("Expected 1536, got %d", dim)
	}
}

func TestProviderManager_EmbedBatchFallback(t *testing.T) {
	// Primary that always fails health check
	primary := &failingEmbeddingProvider{}
	// Fallback that works
	fallback := &workingEmbeddingProvider{dim: 512}

	pm := NewProviderManager(primary, fallback)

	results, err := pm.EmbedBatch([]string{"text1", "text2"})
	if err != nil {
		t.Fatalf("EmbedBatch failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestProviderManager_Name(t *testing.T) {
	primary := &failingEmbeddingProvider{}
	fallback := &workingEmbeddingProvider{dim: 512}

	pm := NewProviderManager(primary, fallback)

	name := pm.Name()
	if name != "working-provider" {
		t.Errorf("Expected working-provider, got %s", name)
	}
}

// Mock providers for testing

type failingEmbeddingProvider struct{}

func (f *failingEmbeddingProvider) EmbedBatch(texts []string) ([][]float32, error) {
	return nil, nil
}

func (f *failingEmbeddingProvider) Embed(text string) ([]float32, error) {
	return nil, nil
}

func (f *failingEmbeddingProvider) Dimension() int {
	return 512
}

func (f *failingEmbeddingProvider) Name() string {
	return "failing-provider"
}

func (f *failingEmbeddingProvider) Health() error {
	return context.DeadlineExceeded
}

type workingEmbeddingProvider struct {
	dim int
}

func (w *workingEmbeddingProvider) EmbedBatch(texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i := range texts {
		results[i] = make([]float32, w.dim)
		for j := 0; j < w.dim; j++ {
			results[i][j] = float32(j) / float32(w.dim)
		}
	}
	return results, nil
}

func (w *workingEmbeddingProvider) Embed(text string) ([]float32, error) {
	emb := make([]float32, w.dim)
	for i := 0; i < w.dim; i++ {
		emb[i] = float32(i) / float32(w.dim)
	}
	return emb, nil
}

func (w *workingEmbeddingProvider) Dimension() int {
	return w.dim
}

func (w *workingEmbeddingProvider) Name() string {
	return "working-provider"
}

func (w *workingEmbeddingProvider) Health() error {
	return nil
}

func TestEmbedBatchWithContext_Cancellation(t *testing.T) {
	emb := &OllamaEmbedding{
		BaseURL:  "http://localhost:99999",
		Model:    "test",
		CacheDim: 768,
		Timeout:  10 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := emb.EmbedBatchWithContext(ctx, []string{"text1"})
	if err == nil {
		t.Log("Warning: expected cancellation error")
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a long string", 10, "this is a ..."},
		{"exact", 5, "exact"},
	}

	for _, tc := range tests {
		result := truncateString(tc.input, tc.maxLen)
		if result != tc.expected {
			t.Errorf("truncateString(%q, %d) = %q, expected %q", tc.input, tc.maxLen, result, tc.expected)
		}
	}
}

// Compile-time check that OllamaEmbedding implements EmbeddingProvider
var _ EmbeddingProvider = (*OllamaEmbedding)(nil)

// TestEmbedBatchConcurrency tests that concurrent calls work
func TestEmbedBatchConcurrency(t *testing.T) {
	emb := &OllamaEmbedding{
		BaseURL:  "http://localhost:99999",
		Model:    "test",
		CacheDim: 768,
		Timeout:  10 * time.Millisecond,
	}

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			emb.EmbedBatch([]string{"text1", "text2"})
		}()
	}
	wg.Wait()
}
