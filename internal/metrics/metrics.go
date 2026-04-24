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

	// IndexErrors total indexing errors
	IndexErrors = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "cortex",
		Name:      "index_errors_total",
		Help:      "Total indexing errors",
	})

	// IndexChunksTotal total number of chunks indexed
	IndexChunksTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "cortex",
		Name:      "index_chunks_total",
		Help:      "Total number of chunks indexed",
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

	// SearchCacheMisses total cache misses
	SearchCacheMisses = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "cortex",
		Name:      "search_cache_misses_total",
		Help:      "Total number of search cache misses",
	})

	// SearchResultsReturned number of search results returned
	SearchResultsReturned = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "cortex",
		Name:      "search_results_returned",
		Help:      "Number of search results returned per query",
		Buckets:   []float64{1, 5, 10, 20, 50, 100},
	})

	// SearchByMode search requests by mode (vector/fts/hybrid)
	SearchByMode = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "cortex",
		Name:      "search_total",
		Help:      "Total search requests by mode",
	}, []string{"mode"})

	// ChunkTotal total number of chunks
	ChunkTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "cortex",
		Name:      "chunks_total",
		Help:      "Total number of indexed chunks",
	})

	// DocumentTotal total number of documents
	DocumentTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "cortex",
		Name:      "documents_total",
		Help:      "Total number of indexed documents",
	})

	// VectorDimension current embedding vector dimension
	VectorDimension = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "cortex",
		Name:      "vector_dimension",
		Help:      "Embedding vector dimension",
	})

	// VectorTotal total number of vectors
	VectorTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "cortex",
		Name:      "vectors_total",
		Help:      "Total number of vectors in index",
	})

	// EmbeddingDuration seconds spent generating embeddings
	EmbeddingDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "cortex",
		Name:      "embedding_duration_seconds",
		Help:      "Time spent generating embeddings",
		Buckets:   prometheus.DefBuckets,
	})

	// EmbeddingErrors total embedding generation errors
	EmbeddingErrors = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "cortex",
		Name:      "embedding_errors_total",
		Help:      "Total embedding generation errors",
	})

	// EmbeddingBatchSize size of embedding batches
	EmbeddingBatchSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "cortex",
		Name:      "embedding_batch_size",
		Help:      "Size of embedding batches",
		Buckets:   []float64{1, 2, 4, 8, 16, 32, 64},
	})

	// HNSWIndexSize size of HNSW index in memory
	HNSWIndexSize = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "cortex",
		Name:      "hnsw_index_size_bytes",
		Help:      "Size of HNSW index in memory",
	})

	// SearchLatency vector search latency
	SearchLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "cortex",
		Name:      "search_latency_seconds",
		Help:      "Search operation latency",
		Buckets:   prometheus.DefBuckets,
	})

	// ErrorsTotal total errors by type
	ErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "cortex",
		Name:      "errors_total",
		Help:      "Total errors by type",
	}, []string{"type", "operation"})

	// APIRequestTotal total API requests
	APIRequestTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "cortex",
		Name:      "api_requests_total",
		Help:      "Total API requests by endpoint",
	}, []string{"endpoint", "method", "status"})

	// APIRequestDuration API request latency
	APIRequestDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "cortex",
		Name:      "api_request_duration_seconds",
		Help:      "API request latency",
		Buckets:   prometheus.DefBuckets,
	})
)