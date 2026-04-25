package embedding

// EmbeddingProvider 提供通用 Embedding 封装支持
type EmbeddingProvider interface {
	// EmbedBatch 并发或批量提交多段文本转化为浮点数向量
	EmbedBatch(texts []string) ([][]float32, error)
	// Embed 单独提取
	Embed(text string) ([]float32, error)

	Dimension() int
	Name() string
	Health() error
}

// ProviderManager 用于包装 Primary 与 Fallback 机制 (例如先调 Ollama, 不行调 OpenAI)
type ProviderManager struct {
	primary  EmbeddingProvider
	fallback EmbeddingProvider
}

// NewProviderManager 初始化聚合管理
func NewProviderManager(primary, fallback EmbeddingProvider) *ProviderManager {
	return &ProviderManager{
		primary:  primary,
		fallback: fallback,
	}
}

// Embed 实现 ProviderManager 对接统一层
func (m *ProviderManager) Embed(text string) ([]float32, error) {
	if m.primary != nil && m.primary.Health() == nil {
		return m.primary.Embed(text)
	}
	if m.fallback != nil {
		return m.fallback.Embed(text)
	}
	return nil, nil // Error handling omitted for brevity
}

// EmbedBatch 实现批量兜底
func (m *ProviderManager) EmbedBatch(texts []string) ([][]float32, error) {
	if m.primary != nil && m.primary.Health() == nil {
		return m.primary.EmbedBatch(texts)
	}
	if m.fallback != nil {
		return m.fallback.EmbedBatch(texts)
	}
	return nil, nil
}

func (m *ProviderManager) Dimension() int {
	if m.primary != nil {
		return m.primary.Dimension()
	}
	if m.fallback != nil {
		return m.fallback.Dimension()
	}
	return 0
}

func (m *ProviderManager) Name() string {
	// 返回当前活跃 provider 的名称（与 Embed/Batch 行为一致）
	if m.primary != nil && m.primary.Health() == nil {
		return m.primary.Name()
	}
	if m.fallback != nil {
		return m.fallback.Name()
	}
	return "none"
}

func (m *ProviderManager) Health() error {
	return nil
}
