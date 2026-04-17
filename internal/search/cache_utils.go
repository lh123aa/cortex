package search

// 占位确保被 indexer 调用到，真正完整的REST引擎可以在此基础上延展
import (
	"context"
)

// InvalidateSearchCacheForDocument 配合文件监测用于精准废弃
func InvalidateSearchCacheForDocument(cacheLayer *SearchCache) {
	if cacheLayer != nil {
		cacheLayer.InvalidateAll()
	}
}
