package chunker

import "github.com/lh123aa/cortex/internal/models"

// ChunkConfig 负责管理数据处理与分块规则的配置
type ChunkConfig struct {
	MaxTokens        int    `yaml:"max_tokens"`         // 单块最大Token
	OverlapTokens    int    `yaml:"overlap_tokens"`     // 重叠Token
	MinChars         int    `yaml:"min_chars"`          // 最小字符数
	IncludeBreadcrumb bool   `yaml:"include_breadcrumb"` // 是否注入标题路径前缀
	Tokenizer        string `yaml:"tokenizer"`          // 比如 cl100k_base
}

// Chunker 接口，根据不同文件格式提供具体拆块支持
type Chunker interface {
	Chunk(content string, path string) ([]*models.Chunk, error)
	Name() string
}
