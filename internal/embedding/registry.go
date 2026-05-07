package embedding

import (
	"fmt"
	"net"
	"sort"
	"time"
)

// ProviderInfo 提供者信息（用于 setup 向导展示）
type ProviderInfo struct {
	ID           string      // 配置标识
	Name         string      // 显示名称
	Category     string      // local / international / domestic
	Description  string      // 简短说明
	Models       []ModelInfo // 可用模型列表
	NeedsAPIKey  bool        // 是否需要 API Key
	NeedsURL     bool        // 是否需要自定义 URL
	NeedsSecret  bool        // 是否需要 Secret Key（百度专用）
	DefaultModel string      // 默认选中的模型 ID
	ModelURL     string      // 模型列表查询 API（用于自动更新检测）
}

// ModelInfo 模型信息
type ModelInfo struct {
	ID          string // 模型标识
	Name        string // 显示名称
	Dimension   int    // 向量维度
	Description string // 说明
	IsDefault   bool   // 是否为默认
}

// NewProviderFromConfig 根据配置创建对应的 EmbeddingProvider
func NewProviderFromConfig(cfg ProviderConfig) (EmbeddingProvider, error) {
	switch cfg.Provider {
	case "", "none":
		return nil, nil
	case "ollama":
		url := cfg.Ollama.BaseURL
		if url == "" {
			url = "http://localhost:11434"
		}
		model := cfg.Ollama.Model
		if model == "" {
			model = "all-minilm"
		}
		return NewOllamaEmbedding(url, model, 384), nil
	case "openai":
		return NewOpenAIEmbedding(cfg.OpenAI.APIKey, cfg.OpenAI.Model, cfg.OpenAI.BaseURL, cfg.OpenAI.Dimension), nil
	case "cohere":
		return NewCohereEmbedding(cfg.Cohere.APIKey, cfg.Cohere.Model, cfg.Cohere.BaseURL, cfg.Cohere.Dimension), nil
	case "voyage":
		return NewVoyageEmbedding(cfg.Voyage.APIKey, cfg.Voyage.Model, cfg.Voyage.BaseURL, cfg.Voyage.Dimension), nil
	case "dashscope":
		return NewDashScopeEmbedding(cfg.DashScope.APIKey, cfg.DashScope.Model, cfg.DashScope.BaseURL, cfg.DashScope.Dimension), nil
	case "zhipu":
		return NewZhipuEmbedding(cfg.Zhipu.APIKey, cfg.Zhipu.Model, cfg.Zhipu.BaseURL, cfg.Zhipu.Dimension), nil
	case "baidu":
		return NewBaiduEmbedding(cfg.Baidu.APIKey, cfg.Baidu.SecretKey, cfg.Baidu.Model, cfg.Baidu.BaseURL, cfg.Baidu.Dimension), nil
	default:
		return nil, fmt.Errorf("unknown embedding provider: %s", cfg.Provider)
	}
}

// ProviderConfig 平铺的配置参数（从 config.Config 映射而来）
type ProviderConfig struct {
	Provider  string
	Ollama    struct{ BaseURL, Model string }
	OpenAI    struct{ APIKey, Model, BaseURL string; Dimension int }
	Cohere    struct{ APIKey, Model, BaseURL string; Dimension int }
	Voyage    struct{ APIKey, Model, BaseURL string; Dimension int }
	DashScope struct{ APIKey, Model, BaseURL string; Dimension int }
	Zhipu     struct{ APIKey, Model, BaseURL string; Dimension int }
	Baidu     struct{ APIKey, SecretKey, Model, BaseURL string; Dimension int }
}

// RegisteredProviders 所有注册的 Provider
var RegisteredProviders = []ProviderInfo{
	// ── 本地模式 ──
	{
		ID: "none", Name: "纯离线模式", Category: "local",
		Description: "FTS5 全文搜索，零外部依赖，最快最轻",
		NeedsAPIKey: false,
	},
	{
		ID: "ollama", Name: "Ollama + 本地模型", Category: "local",
		Description: "本地运行 embedding 模型，推荐 all-minilm（23MB，快速）",
		Models: []ModelInfo{
			{ID: "all-minilm", Name: "all-minilm (L6-v2)", Dimension: 384, Description: "23MB · 推荐 · 速度最快", IsDefault: true},
			{ID: "nomic-embed-text", Name: "nomic-embed-text", Dimension: 768, Description: "137MB · 精度更高 · 速度较慢"},
			{ID: "bge-small-en-v1.5", Name: "bge-small-en-v1.5", Dimension: 384, Description: "33MB · 英文优化"},
			{ID: "bge-m3", Name: "bge-m3", Dimension: 1024, Description: "2.2GB · 多语言 · 精度最高"},
		},
		NeedsAPIKey: false, NeedsURL: true,
		DefaultModel: "all-minilm",
	},

	// ── 国外 API ──
	{
		ID: "openai", Name: "OpenAI", Category: "international",
		Description: "text-embedding-3-small/large · 行业标杆 · 需 API Key",
		Models: []ModelInfo{
			{ID: "text-embedding-3-small", Name: "text-embedding-3-small", Dimension: 512, Description: "512维 · $0.02/1M tokens · 推荐", IsDefault: true},
			{ID: "text-embedding-3-large", Name: "text-embedding-3-large", Dimension: 256, Description: "256维 · $0.13/1M tokens · 高精度"},
			{ID: "text-embedding-ada-002", Name: "text-embedding-ada-002", Dimension: 1536, Description: "1536维 · $0.10/1M tokens · 经典"},
		},
		NeedsAPIKey: true,
		DefaultModel: "text-embedding-3-small",
	},
	{
		ID: "cohere", Name: "Cohere", Category: "international",
		Description: "embed-multilingual-v3.0 · 多语言支持好 · 需 API Key",
		Models: []ModelInfo{
			{ID: "embed-multilingual-v3.0", Name: "embed-multilingual-v3.0", Dimension: 1024, Description: "1024维 · 多语言 · 推荐", IsDefault: true},
			{ID: "embed-english-v3.0", Name: "embed-english-v3.0", Dimension: 1024, Description: "1024维 · 英文专用 · 精度更高"},
			{ID: "embed-english-light-v3.0", Name: "embed-english-light-v3.0", Dimension: 384, Description: "384维 · 轻量 · 速度快"},
		},
		NeedsAPIKey: true,
		DefaultModel: "embed-multilingual-v3.0",
	},
	{
		ID: "voyage", Name: "Voyage AI", Category: "international",
		Description: "voyage-3-lite / voyage-code-3 · 代码搜索特化 · 需 API Key",
		Models: []ModelInfo{
			{ID: "voyage-3-lite", Name: "voyage-3-lite", Dimension: 512, Description: "512维 · 快速 · 推荐", IsDefault: true},
			{ID: "voyage-3", Name: "voyage-3", Dimension: 1024, Description: "1024维 · 高精度"},
			{ID: "voyage-code-3", Name: "voyage-code-3", Dimension: 1536, Description: "1536维 · 代码搜索特化"},
		},
		NeedsAPIKey: true, NeedsURL: true,
		DefaultModel: "voyage-3-lite",
	},

	// ── 国内 API ──
	{
		ID: "dashscope", Name: "阿里云 DashScope", Category: "domestic",
		Description: "通义千问 text-embedding-v2 · 国内延迟低 · 需 API Key",
		Models: []ModelInfo{
			{ID: "text-embedding-v2", Name: "text-embedding-v2", Dimension: 1536, Description: "1536维 · 推荐", IsDefault: true},
			{ID: "text-embedding-v1", Name: "text-embedding-v1", Dimension: 1536, Description: "1536维 · 经典"},
		},
		NeedsAPIKey: true,
		DefaultModel: "text-embedding-v2",
	},
	{
		ID: "zhipu", Name: "智谱 GLM", Category: "domestic",
		Description: "GLM embedding-2/3 · 国产自研 · 需 API Key",
		Models: []ModelInfo{
			{ID: "embedding-3", Name: "embedding-3", Dimension: 2048, Description: "2048维 · 最新 · 推荐", IsDefault: true},
			{ID: "embedding-2", Name: "embedding-2", Dimension: 1024, Description: "1024维 · 经典"},
		},
		NeedsAPIKey: true,
		DefaultModel: "embedding-3",
	},
	{
		ID: "baidu", Name: "百度文心 ERNIE", Category: "domestic",
		Description: "Embedding-V1 · 百度生态 · 需 API Key + Secret Key",
		Models: []ModelInfo{
			{ID: "Embedding-V1", Name: "Embedding-V1", Dimension: 384, Description: "384维 · 推荐", IsDefault: true},
		},
		NeedsAPIKey: true, NeedsSecret: true,
		DefaultModel: "Embedding-V1",
	},
}

// DetectNetwork 检测网络连通性（超时 3 秒）
func DetectNetwork() bool {
	timeout := 3 * time.Second
	// 尝试连接多个常见域名，任一可达即判定为有网
	targets := []string{"api.openai.com:443", "api.cohere.com:443", "dashscope.aliyuncs.com:443"}
	for _, t := range targets {
		conn, err := net.DialTimeout("tcp", t, timeout)
		if err == nil {
			conn.Close()
			return true
		}
	}
	return false
}

// GetProviderByID 根据 ID 查找 Provider
func GetProviderByID(id string) *ProviderInfo {
	for _, p := range RegisteredProviders {
		if p.ID == id {
			return &p
		}
	}
	return nil
}

// GetProvidersByCategory 按类别获取 Provider
func GetProvidersByCategory(category string) []ProviderInfo {
	var result []ProviderInfo
	for _, p := range RegisteredProviders {
		if p.Category == category {
			result = append(result, p)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// GetOnlineProviders 获取需要联网的 Provider
func GetOnlineProviders() []ProviderInfo {
	var result []ProviderInfo
	for _, p := range RegisteredProviders {
		if p.Category != "local" {
			result = append(result, p)
		}
	}
	return result
}
