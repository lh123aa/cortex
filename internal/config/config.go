package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Config holds all configuration for Cortex
type Config struct {
	Cortex    CortexConfig    `mapstructure:"cortex"`
	Embedding EmbeddingConfig `mapstructure:"embedding"`
	Index     IndexConfig     `mapstructure:"index"`
	Search    SearchConfig    `mapstructure:"search"`
	Backup    BackupConfig    `mapstructure:"backup"`
	Vector    VectorConfig    `mapstructure:"vector"`
}

// CortexConfig holds core Cortex settings
type CortexConfig struct {
	DBPath      string `mapstructure:"db_path"`
	LogLevel    string `mapstructure:"log_level"`
	AuthEnabled bool   `mapstructure:"auth_enabled"`
}

// EmbeddingConfig holds embedding provider settings
type EmbeddingConfig struct {
	Provider    string          `mapstructure:"provider"`
	AutoUpdate  bool            `mapstructure:"auto_update"`
	Ollama      OllamaConfig    `mapstructure:"ollama"`
	ONNX        ONNXConfig      `mapstructure:"onnx"`
	OpenAI      APIConfig       `mapstructure:"openai"`
	Cohere      APIConfig       `mapstructure:"cohere"`
	Voyage      APIConfig       `mapstructure:"voyage"`
	DashScope   APIConfig       `mapstructure:"dashscope"`
	Zhipu       APIConfig       `mapstructure:"zhipu"`
	Baidu       BaiduConfig     `mapstructure:"baidu"`
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

// APIConfig generic API-based embedding provider config
type APIConfig struct {
	APIKey    string `mapstructure:"api_key"`
	Model     string `mapstructure:"model"`
	BaseURL   string `mapstructure:"base_url"`
	Dimension int    `mapstructure:"dimension"`
}

// BaiduConfig 百度文心特有配置（需 access_key + secret_key 换取 token）
type BaiduConfig struct {
	APIKey      string `mapstructure:"api_key"`
	SecretKey   string `mapstructure:"secret_key"`
	Model       string `mapstructure:"model"`
	Dimension   int    `mapstructure:"dimension"`
	BaseURL     string `mapstructure:"base_url"`
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
	CacheTTL    string `mapstructure:"cache_ttl"`
	DefaultTopK int    `mapstructure:"default_top_k"`
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
	Compression  string `mapstructure:"compression"`   // none/pq
	Dimension    int    `mapstructure:"dimension"`     // 原始向量维度 (默认768)
	PQDim        int    `mapstructure:"pq_dim"`        // PQ压缩后维度 (默认64)
	CodebookSize int    `mapstructure:"codebook_size"` // 码本大小 (默认256)
}

// UsePQ 是否启用 PQ 压缩
func (v *VectorConfig) UsePQ() bool {
	return v.Compression == "pq"
}

// ConfigWatcher 配置变更监听器
type ConfigWatcher struct {
	viper    *viper.Viper
	mu       sync.RWMutex
	done     chan struct{}
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

	v := viper.New()

	// 设置默认值
	v.SetDefault("cortex.db_path", filepath.Join(defaultDir, "cortex.db"))
	v.SetDefault("cortex.log_level", "info")
	v.SetDefault("embedding.provider", "none")
	v.SetDefault("embedding.auto_update", true)
	v.SetDefault("embedding.ollama.base_url", "http://localhost:11434")
	v.SetDefault("embedding.ollama.model", "all-minilm")
	v.SetDefault("index.max_tokens", 512)
	v.SetDefault("index.overlap_tokens", 64)
	v.SetDefault("index.min_chars", 50)
	v.SetDefault("index.workers", 8)
	v.SetDefault("search.cache_ttl", "5m")
	v.SetDefault("search.default_top_k", 10)
	v.SetDefault("backup.enabled", true)
	v.SetDefault("backup.dir", filepath.Join(defaultDir, "backups"))
	v.SetDefault("backup.max_backups", 10)
	v.SetDefault("backup.auto_backup", false)
	v.SetDefault("vector.compression", "none")
	v.SetDefault("vector.dimension", 768)
	v.SetDefault("vector.pq_dim", 64)
	v.SetDefault("vector.codebook_size", 256)

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(defaultDir)               // ~/.cortex/config.yaml
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
	v.AddConfigPath(defaultDir)
	v.AddConfigPath(".")

	// 读取现有配置
	if err := v.ReadInConfig(); err != nil {
		mu.Unlock()
		return fmt.Errorf("failed to read config for watching: %w", err)
	}

	watcher = &ConfigWatcher{
		viper:    v,
		done:     make(chan struct{}),
		onChange: onChange,
	}
	mu.Unlock()

	go watcher.watch()
	return nil
}

// watch 监听配置文件变更（使用 fsnotify 实现热加载）
func (w *ConfigWatcher) watch() {
	defer func() {
		if r := recover(); r != nil {
			// ignore
		}
	}()

	// 使用 viper 的 WatchConfig 实现热加载
	w.viper.WatchConfig()
	w.viper.OnConfigChange(func(e fsnotify.Event) {
		if e.Op != fsnotify.Write {
			return
		}
		w.handleChange(e)
	})

	// 保持 goroutine 运行直到收到停止信号
	<-w.done
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

	// 直接修改当前配置的副本
	newCfg := currentCfg

	// 应用更新
	applyUpdate(newCfg, updates)

	mu.Lock()
	cfg = newCfg
	mu.Unlock()

	return nil
}

// applyUpdate 递归应用更新到配置结构体
func applyUpdate(cfg *Config, updates map[string]interface{}) {
	for key, value := range updates {
		switch key {
		case "cortex.db_path":
			if v, ok := value.(string); ok {
				cfg.Cortex.DBPath = v
			}
		case "cortex.log_level":
			if v, ok := value.(string); ok {
				cfg.Cortex.LogLevel = v
			}
		case "cortex.auth_enabled":
			if v, ok := value.(bool); ok {
				cfg.Cortex.AuthEnabled = v
			}
		case "embedding.provider":
			if v, ok := value.(string); ok {
				cfg.Embedding.Provider = v
			}
		case "embedding.ollama.base_url":
			if v, ok := value.(string); ok {
				cfg.Embedding.Ollama.BaseURL = v
			}
		case "embedding.ollama.model":
			if v, ok := value.(string); ok {
				cfg.Embedding.Ollama.Model = v
			}
		case "embedding.onnx.base_url":
			if v, ok := value.(string); ok {
				cfg.Embedding.ONNX.BaseURL = v
			}
		case "embedding.onnx.model":
			if v, ok := value.(string); ok {
				cfg.Embedding.ONNX.Model = v
			}
		case "embedding.onnx.dim":
			if v, ok := value.(int); ok {
				cfg.Embedding.ONNX.Dim = v
			}
		case "index.max_tokens":
			if v, ok := value.(int); ok {
				cfg.Index.MaxTokens = v
			}
		case "index.overlap_tokens":
			if v, ok := value.(int); ok {
				cfg.Index.OverlapTokens = v
			}
		case "index.min_chars":
			if v, ok := value.(int); ok {
				cfg.Index.MinChars = v
			}
		case "index.workers":
			if v, ok := value.(int); ok {
				cfg.Index.Workers = v
			}
		case "search.cache_ttl":
			if v, ok := value.(string); ok {
				cfg.Search.CacheTTL = v
			}
		case "search.default_top_k":
			if v, ok := value.(int); ok {
				cfg.Search.DefaultTopK = v
			}
		case "backup.enabled":
			if v, ok := value.(bool); ok {
				cfg.Backup.Enabled = v
			}
		case "backup.dir":
			if v, ok := value.(string); ok {
				cfg.Backup.Dir = v
			}
		case "backup.max_backups":
			if v, ok := value.(int); ok {
				cfg.Backup.MaxBackups = v
			}
		case "backup.auto_backup":
			if v, ok := value.(bool); ok {
				cfg.Backup.AutoBackup = v
			}
		case "vector.compression":
			if v, ok := value.(string); ok {
				cfg.Vector.Compression = v
			}
		case "vector.dimension":
			if v, ok := value.(int); ok {
				cfg.Vector.Dimension = v
			}
		case "vector.pq_dim":
			if v, ok := value.(int); ok {
				cfg.Vector.PQDim = v
			}
		case "vector.codebook_size":
			if v, ok := value.(int); ok {
				cfg.Vector.CodebookSize = v
			}
		}
	}
}

// ValidateConfig 验证配置有效性
func ValidateConfig(c *Config) error {
	if c.Cortex.DBPath == "" {
		return fmt.Errorf("cortex.db_path is required")
	}
	validProviders := map[string]bool{
		"ollama": true, "onnx": true, "none": true,
		"openai": true, "cohere": true, "voyage": true,
		"dashscope": true, "zhipu": true, "baidu": true,
	}
	if !validProviders[c.Embedding.Provider] {
		return fmt.Errorf("embedding.provider must be one of: ollama, onnx, none, openai, cohere, voyage, dashscope, zhipu, baidu")
	}
	if c.Index.Workers <= 0 || c.Index.Workers > 32 {
		return fmt.Errorf("index.workers must be between 1 and 32")
	}
	if c.Search.DefaultTopK <= 0 || c.Search.DefaultTopK > 1000 {
		return fmt.Errorf("search.default_top_k must be between 1 and 1000")
	}
	return nil
}
