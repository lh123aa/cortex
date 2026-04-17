package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds all configuration for Cortex
type Config struct {
	Cortex     CortexConfig     `mapstructure:"cortex"`
	Embedding EmbeddingConfig   `mapstructure:"embedding"`
	Index     IndexConfig       `mapstructure:"index"`
	Search    SearchConfig      `mapstructure:"search"`
	Backup    BackupConfig      `mapstructure:"backup"`
}

// CortexConfig holds core Cortex settings
type CortexConfig struct {
	DBPath  string `mapstructure:"db_path"`
	LogLevel string `mapstructure:"log_level"`
}

// EmbeddingConfig holds embedding provider settings
type EmbeddingConfig struct {
	Provider string             `mapstructure:"provider"`
	Ollama  OllamaConfig        `mapstructure:"ollama"`
	ONNX    ONNXConfig         `mapstructure:"onnx"`
}

// OllamaConfig holds Ollama-specific settings
type OllamaConfig struct {
	BaseURL string `mapstructure:"base_url"`
	Model   string `mapstructure:"model"`
}

// ONNXConfig holds ONNX-specific settings
type ONNXConfig struct {
	BaseURL string `mapstructure:"base_url"`
	Model   string `mapstructure:"model"`
	Dim     int    `mapstructure:"dim"`
}

// IndexConfig holds indexing settings
type IndexConfig struct {
	MaxTokens     int `mapstructure:"max_tokens"`
	OverlapTokens int `mapstructure:"overlap_tokens"`
	MinChars      int `mapstructure:"min_chars"`
	Workers       int `mapstructure:"workers"`
}

// SearchConfig holds search settings
type SearchConfig struct {
	CacheTTL     string `mapstructure:"cache_ttl"`
	DefaultTopK  int    `mapstructure:"default_top_k"`
}

// BackupConfig holds backup settings
type BackupConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	Dir        string `mapstructure:"dir"`
	MaxBackups int    `mapstructure:"max_backups"`
	AutoBackup bool   `mapstructure:"auto_backup"`
}

// Global config instance
var cfg *Config

// Load loads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	home, _ := os.UserHomeDir()
	defaultDir := filepath.Join(home, ".cortex")

	viper.SetDefault("cortex.db_path", filepath.Join(defaultDir, "cortex.db"))
	viper.SetDefault("cortex.log_level", "info")
	viper.SetDefault("embedding.provider", "ollama")
	viper.SetDefault("embedding.ollama.base_url", "http://localhost:11434")
	viper.SetDefault("embedding.ollama.model", "nomic-embed-text")
	viper.SetDefault("index.max_tokens", 512)
	viper.SetDefault("index.overlap_tokens", 64)
	viper.SetDefault("index.min_chars", 50)
	viper.SetDefault("index.workers", 4)
	viper.SetDefault("search.cache_ttl", "5m")
	viper.SetDefault("search.default_top_k", 10)
	viper.SetDefault("backup.enabled", true)
	viper.SetDefault("backup.dir", filepath.Join(defaultDir, "backups"))
	viper.SetDefault("backup.max_backups", 10)
	viper.SetDefault("backup.auto_backup", false)

	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(filepath.Join(defaultDir, "config"))
		viper.AddConfigPath(".")
	}

	// Environment variable overrides
	viper.AutomaticEnv()
	viper.SetEnvPrefix("CORTEX")
	viper.SetEnvKeyReplacer(newReplacer())

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
		// Config file not found, use defaults only
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}

// Get returns the global config instance
func Get() *Config {
	return cfg
}

// replacer for env variable names
type replacer struct{}

func newReplacer() *replacer {
	return &replacer{}
}

func (r *replacer) Replace(s string) string {
	// Already handled by viper
	return s
}
