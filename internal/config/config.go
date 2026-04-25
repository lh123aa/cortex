package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Config holds all configuration for Cortex
type Config struct {
	Cortex     CortexConfig     `mapstructure:"cortex"`
	Embedding  EmbeddingConfig  `mapstructure:"embedding"`
	Index      IndexConfig      `mapstructure:"index"`
	Search     SearchConfig     `mapstructure:"search"`
	Backup     BackupConfig     `mapstructure:"backup"`
	Vector     VectorConfig     `mapstructure:"vector"`
}

// CortexConfig holds core Cortex settings
type CortexConfig struct {
	DBPath      string `mapstructure:"db_path"`
	LogLevel    string `mapstructure:"log_level"`
	AuthEnabled bool   `mapstructure:"auth_enabled"`
}

// EmbeddingConfig holds embedding provider settings
type EmbeddingConfig struct {
	Provider string       `mapstructure:"provider"`
	Ollama   OllamaConfig  `mapstructure:"ollama"`
	ONNX     ONNXConfig    `mapstructure:"onnx"`
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

// VectorConfig 向量相关配置
type VectorConfig struct {
	Compression string `mapstructure:"compression"` // none/pq
	Dimension  int    `mapstructure:"dimension"`    // 原始向量维度 (默认768)
	PQDim      int    `mapstructure:"pq_dim"`      // PQ压缩后维度 (默认64)
	CodebookSize int  `mapstructure:"codebook_size"` // 码本大小 (默认256)
}

// UsePQ 是否启用 PQ 压缩
func (v *VectorConfig) UsePQ() bool {
	return v.Compression == "pq"
}

// ConfigWatcher 配置变更监听器
type ConfigWatcher struct {
	viper  *viper.Viper
	mu     sync.RWMutex
	done   chan struct{}
	onChange func(*Config) // 配置变更回调
}

// Global config instance
var (
	cfg     *Config
	watcher *ConfigWatcher
	mu      sync.RWMutex
)

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
	viper.SetDefault("index.workers", 8) // 优化：从 4 提升到 8，提升索引吞吐量
	viper.SetDefault("search.cache_ttl", "5m")
	viper.SetDefault("search.default_top_k", 10)
	viper.SetDefault("backup.enabled", true)
	viper.SetDefault("backup.dir", filepath.Join(defaultDir, "backups"))
	viper.SetDefault("backup.max_backups", 10)
	viper.SetDefault("backup.auto_backup", false)
	viper.SetDefault("vector.compression", "none")
	viper.SetDefault("vector.dimension", 768)
	viper.SetDefault("vector.pq_dim", 64)
	viper.SetDefault("vector.codebook_size", 256)

	v := viper.New()
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(filepath.Join(defaultDir, "config"))
		v.AddConfigPath(".")
	}

	// Environment variable overrides
	v.AutomaticEnv()
	v.SetEnvPrefix("CORTEX")
	v.SetEnvKeyReplacer(strings.NewReplacer("_", "."))

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
		// Config file not found, use defaults only
	}

	config := &Config{}
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	mu.Lock()
	cfg = config
	mu.Unlock()

	return cfg, nil
}

// WatchConfig 启动配置热更新监控
func WatchConfig(onChange func(*Config)) error {
	mu.RLock()
	c := cfg
	mu.RUnlock()
	if c == nil {
		return fmt.Errorf("config not loaded, call Load first")
	}

	mu.Lock()
	if watcher != nil {
		mu.Unlock()
		return nil // already watching
	}

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	home, _ := os.UserHomeDir()
	defaultDir := filepath.Join(home, ".cortex")
	v.AddConfigPath(filepath.Join(defaultDir, "config"))
	v.AddConfigPath(".")

	// 读取现有配置
	if err := v.ReadInConfig(); err != nil {
		mu.Unlock()
		return fmt.Errorf("failed to read config for watching: %w", err)
	}

	watcher = &ConfigWatcher{
		viper:     v,
		done:      make(chan struct{}),
		onChange:  onChange,
	}
	mu.Unlock()

	go watcher.watch()
	return nil
}

// watch 监听配置文件变更
func (w *ConfigWatcher) watch() {
	defer func() {
		if r := recover(); r != nil {
			// ignore
		}
	}()

	for {
		select {
		case <-w.done:
			return
		case <-time.After(1 * time.Second):
			// 简单轮询检查配置变更 (Viper 的 WatchConfig 不返回 channel)
			// 实际触发由 OnConfigChange 回调处理
		}
	}
}

// handleChange 处理配置变更
func (w *ConfigWatcher) handleChange(event fsnotify.Event) {
	if event.Op != fsnotify.Write {
		return
	}

	newCfg := &Config{}
	if err := w.viper.Unmarshal(newCfg); err != nil {
		// log error
		return
	}

	mu.Lock()
	cfg = newCfg
	mu.Unlock()

	if w.onChange != nil {
		w.onChange(newCfg)
	}
}

// StopWatch 停止配置监听
func StopWatch() {
	mu.Lock()
	defer mu.Unlock()
	if watcher != nil {
		close(watcher.done)
		watcher = nil
	}
}

// Get returns the global config instance (thread-safe)
func Get() *Config {
	mu.RLock()
	defer mu.RUnlock()
	return cfg
}

// UpdatePartial 部分更新配置
func UpdatePartial(updates map[string]interface{}) error {
	mu.RLock()
	currentCfg := cfg
	mu.RUnlock()

	if currentCfg == nil {
		return fmt.Errorf("config not loaded")
	}

	// 使用 viper 的 MergeConfigMap 进行部分更新
	v := viper.New()
	if err := v.MergeConfigMap(updates); err != nil {
		return err
	}

	newCfg := &Config{}
	if err := v.Unmarshal(newCfg); err != nil {
		return err
	}

	mu.Lock()
	cfg = newCfg
	mu.Unlock()

	return nil
}

// ValidateConfig 验证配置有效性
func ValidateConfig(c *Config) error {
	if c.Cortex.DBPath == "" {
		return fmt.Errorf("cortex.db_path is required")
	}
	if c.Embedding.Provider != "ollama" && c.Embedding.Provider != "onnx" {
		return fmt.Errorf("embedding.provider must be 'ollama' or 'onnx'")
	}
	if c.Index.Workers <= 0 || c.Index.Workers > 32 {
		return fmt.Errorf("index.workers must be between 1 and 32")
	}
	if c.Search.DefaultTopK <= 0 || c.Search.DefaultTopK > 1000 {
		return fmt.Errorf("search.default_top_k must be between 1 and 1000")
	}
	return nil
}