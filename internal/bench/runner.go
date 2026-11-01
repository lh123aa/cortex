package bench

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/lh123aa/cortex/internal/index"
	"github.com/lh123aa/cortex/internal/models"
	"github.com/lh123aa/cortex/internal/search"
	"github.com/lh123aa/cortex/internal/storage"
)

// Config 是一次基准运行的参数。
type Config struct {
	// CorpusDir 是语料目录（生成或复用）。
	CorpusDir string `json:"corpus_dir"`
	// DBPath 是基准专用数据库路径（每次运行前应删除以保证冷启动）。
	DBPath string `json:"db_path"`
	// Documents 是语料文档数。
	Documents int `json:"documents"`
	// Seed 是语料随机种子。
	Seed int64 `json:"seed"`
	// Workers 是索引并发数。
	Workers int `json:"workers"`
	// QueriesPerDoc 是每篇文档生成的查询数（1-3）。
	QueriesPerDoc int `json:"queries_per_doc"`
	// TopK 是检索返回条数（质量计算至少需要 10）。
	TopK int `json:"top_k"`
	// Repeat 是延迟测量时每条查询重复次数。
	Repeat int `json:"repeat"`
}

func (c Config) normalized() Config {
	if c.Documents <= 0 {
		c.Documents = 500
	}
	if c.Workers <= 0 {
		c.Workers = 8
	}
	if c.QueriesPerDoc <= 0 {
		c.QueriesPerDoc = 3
	}
	if c.TopK < 10 {
		c.TopK = 10
	}
	if c.Repeat <= 0 {
		c.Repeat = 3
	}
	if c.Seed == 0 {
		c.Seed = 20261023
	}
	return c
}

// Result 是一次完整基准的结果。
type Result struct {
	Config       Config           `json:"config"`
	Corpus       CorpusStats      `json:"corpus"`
	Index        IndexStats       `json:"index"`
	Search       SearchStats      `json:"search"`
	Quality      Metrics          `json:"quality"`
	Ambiguous    AmbiguousMetrics `json:"ambiguous"`
	ChunksPerDoc float64          `json:"chunks_per_doc"`
}

// CorpusStats 是语料规模。
type CorpusStats struct {
	Documents  int  `json:"documents"`
	TotalChars int  `json:"total_chars"`
	Generated  bool `json:"generated"`
}

// IndexStats 是索引阶段指标。
type IndexStats struct {
	Files     int           `json:"files"`
	Indexed   int           `json:"indexed"`
	Failed    int           `json:"failed"`
	Skipped   int           `json:"skipped"`
	Duration  time.Duration `json:"duration_ns"`
	FilesPerS float64       `json:"files_per_s"`
	Chunks    int           `json:"chunks"`
}

// SearchStats 是检索阶段指标。
type SearchStats struct {
	Queries     int          `json:"queries"`
	Latency     LatencyStats `json:"latency"`
	QueriesPerS float64      `json:"qps"`
}

// RunAll 执行「生成语料 → 索引 → 检索测量 → 质量评估」的完整流程。
func RunAll(ctx context.Context, log func(string, ...any), cfg Config) (*Result, error) {
	cfg = cfg.normalized()
	res := &Result{Config: cfg}

	// 1) 语料
	log("generating corpus: %d docs (seed=%d) -> %s", cfg.Documents, cfg.Seed, cfg.CorpusDir)
	corpus, err := GenerateCorpus(cfg.CorpusDir, CorpusOptions{Docs: cfg.Documents, Seed: cfg.Seed})
	if err != nil {
		return nil, err
	}
	res.Corpus = CorpusStats{
		Documents:  len(corpus.Docs),
		TotalChars: corpus.TotalChar,
		Generated:  true,
	}

	// 2) 打开基准库（调用方负责删除旧库以获得冷启动）
	st, err := storage.NewSQLiteStorage(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open benchmark db: %w", err)
	}
	defer st.Close()

	// 3) 索引
	idx, err := index.NewIndexer(st, nil, cfg.Workers)
	if err != nil {
		return nil, fmt.Errorf("create indexer: %w", err)
	}
	log("indexing %d documents with %d workers", len(corpus.Docs), cfg.Workers)
	start := time.Now()
	idxRes, err := idx.IndexDirectory(ctx, filepath.Clean(cfg.CorpusDir), "")
	elapsed := time.Since(start)
	if err != nil {
		return nil, fmt.Errorf("index failed: %w", err)
	}

	chunks, _ := st.GetChunksCount("")
	res.Index = IndexStats{
		Files:    idxRes.Total,
		Indexed:  idxRes.Indexed,
		Failed:   idxRes.Failed,
		Skipped:  idxRes.Skipped,
		Duration: elapsed,
		Chunks:   chunks,
	}
	if secs := elapsed.Seconds(); secs > 0 {
		res.Index.FilesPerS = float64(idxRes.Indexed) / secs
	}
	log("indexed %d/%d files in %s (%.1f files/s, %d chunks)",
		idxRes.Indexed, idxRes.Total, elapsed.Round(time.Millisecond), res.Index.FilesPerS, chunks)

	// 4) 检索 + 质量
	se := search.NewHybridSearchEngine(st, nil)
	se.SetRRFK(60)
	se.DisableCache() // 基准必须测真实检索，不能命中缓存

	queries := BuildQueries(corpus, cfg.QueriesPerDoc)
	log("running %d queries (top_k=%d, repeat=%d)", len(queries), cfg.TopK, cfg.Repeat)

	results := make([][]string, len(queries))
	samples := make([]time.Duration, 0, len(queries)*cfg.Repeat)

	for i, q := range queries {
		paths, dur, err := runQuery(ctx, se, st, q.Text, cfg.TopK)
		if err != nil {
			return nil, fmt.Errorf("query %q failed: %w", q.Text, err)
		}
		results[i] = paths
		samples = append(samples, dur)

		// 额外重复，只采延迟，不参与质量计算
		for r := 1; r < cfg.Repeat; r++ {
			_, d, err := runQuery(ctx, se, st, q.Text, cfg.TopK)
			if err != nil {
				return nil, fmt.Errorf("query %q (repeat) failed: %w", q.Text, err)
			}
			samples = append(samples, d)
		}
	}

	res.Search = SearchStats{
		Queries: len(queries),
		Latency: Latency(samples),
	}
	if d := res.Search.Latency.Mean; d > 0 {
		res.Search.QueriesPerS = 1e9 / float64(d.Nanoseconds())
	}

	res.Quality = Evaluate(queries, results)
	res.Ambiguous = EvaluateAmbiguous(queries, results, 5)
	log("quality: recall@1=%.3f recall@5=%.3f recall@10=%.3f mrr=%.3f misses=%d",
		res.Quality.RecallAt1, res.Quality.RecallAt5, res.Quality.RecallAt10,
		res.Quality.MRR, res.Quality.Misses)

	return res, nil
}

// runQuery 执行单条查询，返回命中的文档路径（按排名）与耗时。
func runQuery(ctx context.Context, se *search.HybridSearchEngine, st storage.Storage, q string, topK int) ([]string, time.Duration, error) {
	start := time.Now()
	rs, err := se.Search(ctx, q, models.SearchOptions{TopK: topK, Mode: "hybrid"})
	dur := time.Since(start)
	if err != nil {
		return nil, dur, err
	}

	paths := make([]string, 0, len(rs))
	seen := make(map[string]bool, len(rs))
	for _, r := range rs {
		if r == nil || r.Chunk == nil {
			continue
		}
		p := r.FilePath
		if p == "" && r.Chunk.DocumentID != "" {
			if doc, err := st.GetDocumentByID(r.Chunk.DocumentID, ""); err == nil && doc != nil {
				p = doc.Path
			}
		}
		if p == "" {
			continue
		}
		// 按文档去重：一篇文档有多个分块时只保留它最高的名次。
		// 否则 top-10 会被同一篇文档的多个分块占满，
		// recall / MRR 就退化成「分块排名」而不是「文档排名」。
		if seen[p] {
			continue
		}
		seen[p] = true
		paths = append(paths, p)
	}
	return paths, dur, nil
}

// TopDirectories 返回语料目录下的文件名样本（供报告展示）。
func TopDirectories(dir string, limit int) []string {
	if limit <= 0 {
		limit = 5
	}
	return []string{dir}
}

var _ = sort.Strings
