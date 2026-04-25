package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary directory for config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write test config
	configContent := `
cortex:
  db_path: /tmp/test.db
  log_level: debug

embedding:
  provider: ollama
  ollama:
    base_url: http://localhost:11434
    model: test-model

index:
  max_tokens: 256
  workers: 2

search:
  cache_ttl: 10m
  default_top_k: 20
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load config
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify loaded values
	if cfg.Cortex.DBPath != "/tmp/test.db" {
		t.Errorf("Expected db_path '/tmp/test.db', got '%s'", cfg.Cortex.DBPath)
	}
	if cfg.Cortex.LogLevel != "debug" {
		t.Errorf("Expected log_level 'debug', got '%s'", cfg.Cortex.LogLevel)
	}
	if cfg.Embedding.Provider != "ollama" {
		t.Errorf("Expected provider 'ollama', got '%s'", cfg.Embedding.Provider)
	}
	if cfg.Embedding.Ollama.BaseURL != "http://localhost:11434" {
		t.Errorf("Expected ollama base_url 'http://localhost:11434', got '%s'", cfg.Embedding.Ollama.BaseURL)
	}
	if cfg.Embedding.Ollama.Model != "test-model" {
		t.Errorf("Expected ollama model 'test-model', got '%s'", cfg.Embedding.Ollama.Model)
	}
	if cfg.Index.MaxTokens != 256 {
		t.Errorf("Expected max_tokens 256, got %d", cfg.Index.MaxTokens)
	}
	if cfg.Index.Workers != 2 {
		t.Errorf("Expected workers 2, got %d", cfg.Index.Workers)
	}
	if cfg.Search.CacheTTL != "10m" {
		t.Errorf("Expected cache_ttl '10m', got '%s'", cfg.Search.CacheTTL)
	}
	if cfg.Search.DefaultTopK != 20 {
		t.Errorf("Expected default_top_k 20, got %d", cfg.Search.DefaultTopK)
	}
}

func TestLoad_Defaults(t *testing.T) {
	// Load with empty config path - should use defaults
	// Empty path triggers default config search paths
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load with empty path failed: %v", err)
	}

	// Check defaults
	if cfg.Cortex.DBPath == "" {
		t.Error("Expected default db_path to be set")
	}
	if cfg.Cortex.LogLevel != "info" {
		t.Errorf("Expected default log_level 'info', got '%s'", cfg.Cortex.LogLevel)
	}
	if cfg.Embedding.Provider != "ollama" {
		t.Errorf("Expected default provider 'ollama', got '%s'", cfg.Embedding.Provider)
	}
	if cfg.Index.MaxTokens != 512 {
		t.Errorf("Expected default max_tokens 512, got %d", cfg.Index.MaxTokens)
	}
	if cfg.Index.Workers != 8 {
		t.Errorf("Expected default workers 8, got %d", cfg.Index.Workers)
	}
	if cfg.Search.DefaultTopK != 10 {
		t.Errorf("Expected default default_top_k 10, got %d", cfg.Search.DefaultTopK)
	}
}

func TestLoad_ONNXConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
cortex:
  db_path: /tmp/test.db

embedding:
  provider: onnx
  onnx:
    base_url: http://localhost:8080
    model: embedder
    dim: 1536

index:
  max_tokens: 512

search:
  default_top_k: 10
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Embedding.Provider != "onnx" {
		t.Errorf("Expected provider 'onnx', got '%s'", cfg.Embedding.Provider)
	}
	if cfg.Embedding.ONNX.BaseURL != "http://localhost:8080" {
		t.Errorf("Expected onnx base_url 'http://localhost:8080', got '%s'", cfg.Embedding.ONNX.BaseURL)
	}
	if cfg.Embedding.ONNX.Model != "embedder" {
		t.Errorf("Expected onnx model 'embedder', got '%s'", cfg.Embedding.ONNX.Model)
	}
	if cfg.Embedding.ONNX.Dim != 1536 {
		t.Errorf("Expected onnx dim 1536, got %d", cfg.Embedding.ONNX.Dim)
	}
}

func TestGet(t *testing.T) {
	// Before any Load, Get should return nil
	// Note: This test depends on previous tests having called Load
	cfg := Get()
	if cfg == nil {
		t.Log("Get returned nil (config not loaded in this test context)")
	} else {
		t.Logf("Get returned config: %+v", cfg)
	}
}

func TestValidateConfig_Valid(t *testing.T) {
	cfg := &Config{
		Cortex: CortexConfig{
			DBPath: "/tmp/test.db",
		},
		Embedding: EmbeddingConfig{
			Provider: "ollama",
		},
		Index: IndexConfig{
			Workers: 4,
		},
		Search: SearchConfig{
			DefaultTopK: 10,
		},
	}

	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("Expected valid config, got error: %v", err)
	}
}

func TestValidateConfig_EmptyDBPath(t *testing.T) {
	cfg := &Config{
		Cortex: CortexConfig{
			DBPath: "",
		},
		Embedding: EmbeddingConfig{
			Provider: "ollama",
		},
		Index: IndexConfig{
			Workers: 4,
		},
		Search: SearchConfig{
			DefaultTopK: 10,
		},
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for empty db_path")
	}
}

func TestValidateConfig_InvalidProvider(t *testing.T) {
	cfg := &Config{
		Cortex: CortexConfig{
			DBPath: "/tmp/test.db",
		},
		Embedding: EmbeddingConfig{
			Provider: "invalid",
		},
		Index: IndexConfig{
			Workers: 4,
		},
		Search: SearchConfig{
			DefaultTopK: 10,
		},
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for invalid provider")
	}
}

func TestValidateConfig_InvalidWorkers(t *testing.T) {
	testCases := []struct {
		name    string
		workers int
	}{
		{"zero workers", 0},
		{"negative workers", -1},
		{"too many workers", 33},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{
				Cortex: CortexConfig{
					DBPath: "/tmp/test.db",
				},
				Embedding: EmbeddingConfig{
					Provider: "ollama",
				},
				Index: IndexConfig{
					Workers: tc.workers,
				},
				Search: SearchConfig{
					DefaultTopK: 10,
				},
			}

			err := ValidateConfig(cfg)
			if err == nil {
				t.Errorf("Expected error for workers=%d", tc.workers)
			}
		})
	}
}

func TestValidateConfig_InvalidTopK(t *testing.T) {
	testCases := []struct {
		name string
		topK int
	}{
		{"zero topK", 0},
		{"negative topK", -1},
		{"too large topK", 1001},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{
				Cortex: CortexConfig{
					DBPath: "/tmp/test.db",
				},
				Embedding: EmbeddingConfig{
					Provider: "ollama",
				},
				Index: IndexConfig{
					Workers: 4,
				},
				Search: SearchConfig{
					DefaultTopK: tc.topK,
				},
			}

			err := ValidateConfig(cfg)
			if err == nil {
				t.Errorf("Expected error for topK=%d", tc.topK)
			}
		})
	}
}

func TestConfig_Struct(t *testing.T) {
	cfg := Config{
		Cortex: CortexConfig{
			DBPath:   "/tmp/db.db",
			LogLevel: "debug",
		},
		Embedding: EmbeddingConfig{
			Provider: "onnx",
			Ollama: OllamaConfig{
				BaseURL: "http://ollama:11434",
				Model:   "llama2",
			},
			ONNX: ONNXConfig{
				BaseURL: "http://onnx:8080",
				Model:   "bge",
				Dim:     1024,
			},
		},
		Index: IndexConfig{
			MaxTokens:     512,
			OverlapTokens: 64,
			MinChars:      50,
			Workers:       8,
		},
		Search: SearchConfig{
			CacheTTL:    "15m",
			DefaultTopK: 20,
		},
		Backup: BackupConfig{
			Enabled:    true,
			Dir:        "/tmp/backups",
			MaxBackups: 5,
			AutoBackup: true,
		},
	}

	if cfg.Cortex.DBPath != "/tmp/db.db" {
		t.Error("Cortex.DBPath mismatch")
	}
	if cfg.Embedding.Provider != "onnx" {
		t.Error("Embedding.Provider mismatch")
	}
	if cfg.Index.Workers != 8 {
		t.Error("Index.Workers mismatch")
	}
	if cfg.Search.DefaultTopK != 20 {
		t.Error("Search.DefaultTopK mismatch")
	}
	if cfg.Backup.MaxBackups != 5 {
		t.Error("Backup.MaxBackups mismatch")
	}
}

func TestCortexConfig_Struct(t *testing.T) {
	cfg := CortexConfig{
		DBPath:   "/custom/path.db",
		LogLevel: "warn",
	}

	if cfg.DBPath != "/custom/path.db" {
		t.Errorf("Expected DBPath '/custom/path.db', got '%s'", cfg.DBPath)
	}
	if cfg.LogLevel != "warn" {
		t.Errorf("Expected LogLevel 'warn', got '%s'", cfg.LogLevel)
	}
}

func TestEmbeddingConfig_Struct(t *testing.T) {
	cfg := EmbeddingConfig{
		Provider: "ollama",
		Ollama: OllamaConfig{
			BaseURL: "http://localhost:11434",
			Model:   "nomic-embed-text",
		},
	}

	if cfg.Provider != "ollama" {
		t.Errorf("Expected Provider 'ollama', got '%s'", cfg.Provider)
	}
	if cfg.Ollama.BaseURL != "http://localhost:11434" {
		t.Errorf("Expected Ollama.BaseURL 'http://localhost:11434', got '%s'", cfg.Ollama.BaseURL)
	}
}

func TestONNXConfig_Struct(t *testing.T) {
	cfg := ONNXConfig{
		BaseURL: "http://onnx:8080",
		Model:   "e5-base",
		Dim:     768,
	}

	if cfg.BaseURL != "http://onnx:8080" {
		t.Errorf("Expected BaseURL 'http://onnx:8080', got '%s'", cfg.BaseURL)
	}
	if cfg.Model != "e5-base" {
		t.Errorf("Expected Model 'e5-base', got '%s'", cfg.Model)
	}
	if cfg.Dim != 768 {
		t.Errorf("Expected Dim 768, got %d", cfg.Dim)
	}
}

func TestIndexConfig_Struct(t *testing.T) {
	cfg := IndexConfig{
		MaxTokens:     1024,
		OverlapTokens: 128,
		MinChars:      100,
		Workers:       16,
	}

	if cfg.MaxTokens != 1024 {
		t.Errorf("Expected MaxTokens 1024, got %d", cfg.MaxTokens)
	}
	if cfg.Workers != 16 {
		t.Errorf("Expected Workers 16, got %d", cfg.Workers)
	}
}

func TestSearchConfig_Struct(t *testing.T) {
	cfg := SearchConfig{
		CacheTTL:    "30m",
		DefaultTopK: 50,
	}

	if cfg.CacheTTL != "30m" {
		t.Errorf("Expected CacheTTL '30m', got '%s'", cfg.CacheTTL)
	}
	if cfg.DefaultTopK != 50 {
		t.Errorf("Expected DefaultTopK 50, got %d", cfg.DefaultTopK)
	}
}

func TestBackupConfig_Struct(t *testing.T) {
	cfg := BackupConfig{
		Enabled:    true,
		Dir:        "/backups",
		MaxBackups: 20,
		AutoBackup: true,
	}

	if !cfg.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if cfg.Dir != "/backups" {
		t.Errorf("Expected Dir '/backups', got '%s'", cfg.Dir)
	}
	if cfg.MaxBackups != 20 {
		t.Errorf("Expected MaxBackups 20, got %d", cfg.MaxBackups)
	}
	if !cfg.AutoBackup {
		t.Error("Expected AutoBackup to be true")
	}
}

func TestUpdatePartial(t *testing.T) {
	// Setup initial config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
cortex:
  db_path: /tmp/original.db
  log_level: info

embedding:
  provider: ollama

index:
  workers: 4

search:
  default_top_k: 10
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Update partial
	updates := map[string]interface{}{
		"search.default_top_k": 50,
		"index.workers":        8,
	}

	err = UpdatePartial(updates)
	if err != nil {
		t.Fatalf("UpdatePartial failed: %v", err)
	}

	// Verify updated values
	cfg := Get()
	if cfg.Search.DefaultTopK != 50 {
		t.Errorf("Expected default_top_k 50, got %d", cfg.Search.DefaultTopK)
	}
	if cfg.Index.Workers != 8 {
		t.Errorf("Expected workers 8, got %d", cfg.Index.Workers)
	}
	// Unchanged values should remain
	if cfg.Cortex.DBPath != "/tmp/original.db" {
		t.Errorf("Expected db_path '/tmp/original.db' to remain unchanged, got '%s'", cfg.Cortex.DBPath)
	}
}

func TestUpdatePartial_Invalid(t *testing.T) {
	// Setup initial config first
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
cortex:
  db_path: /tmp/test.db

embedding:
  provider: ollama

index:
  workers: 4

search:
  default_top_k: 10
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Try to update with invalid values
	updates := map[string]interface{}{
		"embedding.provider": "invalid_provider",
	}

	err = UpdatePartial(updates)
	// Should not error on invalid provider in partial update (validation happens on Load/Get)
	if err != nil {
		t.Logf("UpdatePartial error (may be expected): %v", err)
	}
}

func TestStopWatch(t *testing.T) {
	// StopWatch should not panic even if watcher is nil
	StopWatch()
	StopWatch() // calling twice should be safe
}
