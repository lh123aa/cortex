package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// IndexTotal total number of documents indexed
	IndexTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "cortex",
		Name:      "index_total",
		Help:      "Total number of documents indexed",
	})

	// IndexDuration seconds spent indexing
	IndexDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "cortex",
		Name:      "index_duration_seconds",
		Help:      "Time spent indexing documents",
		Buckets:   prometheus.DefBuckets,
	})

	// SearchTotal total number of search requests
	SearchTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "cortex",
		Name:      "search_total",
		Help:      "Total number of search requests",
	})

	// SearchDuration seconds spent searching
	SearchDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "cortex",
		Name:      "search_duration_seconds",
		Help:      "Time spent searching",
		Buckets:   prometheus.DefBuckets,
	})

	// SearchCacheHits total cache hits
	SearchCacheHits = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "cortex",
		Name:      "search_cache_hits_total",
		Help:      "Total number of search cache hits",
	})

	// ChunkTotal total number of chunks
	ChunkTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "cortex",
		Name:      "chunks_total",
		Help:      "Total number of indexed chunks",
	})

	// VectorDimension current embedding vector dimension
	VectorDimension = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "cortex",
		Name:      "vector_dimension",
		Help:      "Embedding vector dimension",
	})

	// EmbeddingDuration seconds spent generating embeddings
	EmbeddingDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "cortex",
		Name:      "embedding_duration_seconds",
		Help:      "Time spent generating embeddings",
		Buckets:   prometheus.DefBuckets,
	})

	// ErrorsTotal total errors by type
	ErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "cortex",
		Name:      "errors_total",
		Help:      "Total errors by type",
	}, []string{"type"})
)
