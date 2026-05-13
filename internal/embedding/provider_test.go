package embedding

import (
	"testing"
)

func TestNewProviderFromConfig_None(t *testing.T) {
	cfg := ProviderConfig{Provider: "none"}
	p, err := NewProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewProviderFromConfig(none) failed: %v", err)
	}
	if p != nil {
		t.Error("expected nil provider for 'none'")
	}
}

func TestNewProviderFromConfig_Ollama(t *testing.T) {
	cfg := ProviderConfig{
		Provider: "ollama",
	}
	cfg.Ollama.BaseURL = "http://localhost:11434"
	cfg.Ollama.Model = "nomic-embed-text"

	p, err := NewProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewProviderFromConfig(ollama) failed: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider for ollama")
	}
	if p.Name() == "" {
		t.Error("expected non-empty name")
	}
}

func TestNewProviderFromConfig_OpenAI(t *testing.T) {
	cfg := ProviderConfig{
		Provider: "openai",
	}
	cfg.OpenAI.APIKey = "sk-test-key"
	cfg.OpenAI.Model = "text-embedding-3-small"
	cfg.OpenAI.BaseURL = "https://api.openai.com/v1"

	p, err := NewProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewProviderFromConfig(openai) failed: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider for openai")
	}
	if p.Name() == "" {
		t.Error("expected non-empty name for openai")
	}
}

func TestNewProviderFromConfig_Invalid(t *testing.T) {
	cfg := ProviderConfig{Provider: "invalid_provider_xyz"}
	p, err := NewProviderFromConfig(cfg)
	if err == nil {
		t.Error("expected error for invalid provider")
	}
	if p != nil {
		t.Error("expected nil provider for invalid provider")
	}
}

func TestGetProviderByID(t *testing.T) {
	p := GetProviderByID("ollama")
	if p == nil {
		t.Fatal("expected ollama provider in registry")
	}
	if p.ID != "ollama" {
		t.Errorf("expected id 'ollama', got %q", p.ID)
	}

	p = GetProviderByID("nonexistent")
	if p != nil {
		t.Error("expected nil for nonexistent provider")
	}
}

func TestRegisteredProviders(t *testing.T) {
	providers := RegisteredProviders
	if len(providers) == 0 {
		t.Fatal("expected at least one registered provider")
	}

	foundNone := false
	foundOllama := false
	for _, p := range providers {
		if p.ID == "none" {
			foundNone = true
		}
		if p.ID == "ollama" {
			foundOllama = true
		}
	}
	if !foundNone {
		t.Error("expected 'none' provider to be registered")
	}
	if !foundOllama {
		t.Error("expected 'ollama' provider to be registered")
	}
}

func TestEmbedError(t *testing.T) {
	err := NewEmbedError("ollama", "embed", nil, true)
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if !err.Retryable {
		t.Error("expected retryable error")
	}
	if err.Provider != "ollama" {
		t.Errorf("expected provider 'ollama', got %q", err.Provider)
	}
	if err.Op != "embed" {
		t.Errorf("expected op 'embed', got %q", err.Op)
	}

	err2 := NewEmbedError("openai", "health", nil, false)
	if err2.Retryable {
		t.Error("expected non-retryable error")
	}
}

func TestDetectNetwork(t *testing.T) {
	// This should always return without error (may return false)
	result := DetectNetwork()
	t.Logf("DetectNetwork returned: %v", result)
}

func TestNewOllamaEmbedding(t *testing.T) {
	emb := NewOllamaEmbedding("http://localhost:11434", "nomic-embed-text", 768)
	if emb == nil {
		t.Fatal("NewOllamaEmbedding returned nil")
	}
	if emb.Model != "nomic-embed-text" {
		t.Errorf("model = %q, want nomic-embed-text", emb.Model)
	}
	if emb.CacheDim != 768 {
		t.Errorf("dim = %d, want 768", emb.CacheDim)
	}
}
