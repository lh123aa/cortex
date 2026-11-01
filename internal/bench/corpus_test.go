package bench

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestGenerateCorpusDeterministic(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	opts := CorpusOptions{Docs: 12, Seed: 42}
	c1, err := GenerateCorpus(dir1, opts)
	if err != nil {
		t.Fatal(err)
	}
	c2, err := GenerateCorpus(dir2, opts)
	if err != nil {
		t.Fatal(err)
	}

	if len(c1.Docs) != 12 || len(c2.Docs) != 12 {
		t.Fatalf("doc count mismatch: %d vs %d", len(c1.Docs), len(c2.Docs))
	}
	for i := range c1.Docs {
		if c1.Docs[i].Topic != c2.Docs[i].Topic {
			t.Errorf("doc %d topic differs across runs: %q vs %q", i, c1.Docs[i].Topic, c2.Docs[i].Topic)
		}
		if c1.Docs[i].UniqueToken != c2.Docs[i].UniqueToken {
			t.Errorf("doc %d unique token differs: %q vs %q", i, c1.Docs[i].UniqueToken, c2.Docs[i].UniqueToken)
		}
		if c1.Docs[i].BodyLen != c2.Docs[i].BodyLen {
			t.Errorf("doc %d body length differs: %d vs %d", i, c1.Docs[i].BodyLen, c2.Docs[i].BodyLen)
		}
	}
}

func TestGenerateCorpusWritesFiles(t *testing.T) {
	dir := t.TempDir()
	c, err := GenerateCorpus(dir, CorpusOptions{Docs: 5, Seed: 1})
	if err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 5 {
		t.Fatalf("expected 5 files, got %d", len(entries))
	}

	for _, d := range c.Docs {
		data, err := os.ReadFile(d.Path)
		if err != nil {
			t.Fatalf("read %s: %v", d.Path, err)
		}
		content := string(data)
		if !strings.Contains(content, d.UniqueToken) {
			t.Errorf("%s missing unique token %q", d.Path, d.UniqueToken)
		}
		if len(content) < 800 {
			t.Errorf("%s too short (%d chars); multi-chunk docs are required", d.Path, len(content))
		}
	}
}

func TestUniqueTokensAreDistinct(t *testing.T) {
	dir := t.TempDir()
	c, err := GenerateCorpus(dir, CorpusOptions{Docs: 30, Seed: 7})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, d := range c.Docs {
		if seen[d.UniqueToken] {
			t.Fatalf("duplicate unique token: %q", d.UniqueToken)
		}
		seen[d.UniqueToken] = true
	}
}

// TestSharedPairsDoNotRepeatWithinCorpus 保证同一次语料里不会有多个簇共用同一对共享词，
// 否则「共享词查询」的候选集会变大，歧义指标失去意义。
func TestSharedPairsDoNotRepeatWithinCorpus(t *testing.T) {
	dir := t.TempDir()
	// 40 簇需要 40 个不重复词对；C(12,2)=66，够用
	c, err := GenerateCorpus(dir, CorpusOptions{Docs: 1000, Seed: 3, ClusterSize: 5})
	if err != nil {
		t.Fatal(err)
	}

	seen := map[string]int{}
	for i := 0; i < len(c.Docs); i += 5 {
		pair := strings.Join(c.Docs[i].SharedTokens, "|")
		if prev, ok := seen[pair]; ok {
			t.Fatalf("shared pair %q repeated: cluster at doc %d and %d", pair, prev, i)
		}
		seen[pair] = i
	}
}

func TestBuildQueriesKinds(t *testing.T) {
	dir := t.TempDir()
	c, err := GenerateCorpus(dir, CorpusOptions{Docs: 4, Seed: 9})
	if err != nil {
		t.Fatal(err)
	}

	q1 := BuildQueries(c, 1)
	if len(q1) != 4 {
		t.Fatalf("perDoc=1 should yield 4 queries, got %d", len(q1))
	}
	for _, q := range q1 {
		if q.Kind != "unique" {
			t.Errorf("expected unique kind, got %q", q.Kind)
		}
	}

	q3 := BuildQueries(c, 3)
	if len(q3) != 12 {
		t.Fatalf("perDoc=3 should yield 12 queries, got %d", len(q3))
	}
	kinds := map[string]int{}
	for _, q := range q3 {
		kinds[q.Kind]++
	}
	for _, k := range []string{"unique", "topic", "shared"} {
		if kinds[k] != 4 {
			t.Errorf("kind %q count = %d, want 4", k, kinds[k])
		}
	}
}

func TestBuildQueriesGroundTruthIsExactDoc(t *testing.T) {
	dir := t.TempDir()
	c, err := GenerateCorpus(dir, CorpusOptions{Docs: 6, Seed: 11})
	if err != nil {
		t.Fatal(err)
	}
	paths := map[string]bool{}
	for _, d := range c.Docs {
		paths[d.Path] = true
	}
	for _, q := range BuildQueries(c, 2) {
		if !paths[q.TargetPath] {
			t.Fatalf("query %q has unknown target %q", q.Text, q.TargetPath)
		}
	}
}

func TestCorpusOptionsDefaults(t *testing.T) {
	o := CorpusOptions{}.normalized()
	if o.Docs != 100 || o.MinChars < 1000 || o.ClusterSize != 5 {
		t.Fatalf("unexpected defaults: %+v", o)
	}
	// 正文必须够长，否则切不出多个分块
	if o.MinChars < 1000 {
		t.Fatalf("default MinChars too small for multi-chunk docs: %d", o.MinChars)
	}
}

func TestRenderMarkdownContainsKeySections(t *testing.T) {
	r := &Result{
		Config:  Config{Documents: 100, Workers: 4, Seed: 1, TopK: 10, Repeat: 3, QueriesPerDoc: 3},
		Corpus:  CorpusStats{Documents: 100, TotalChars: 123456},
		Index:   IndexStats{Files: 100, Indexed: 100, Duration: 1_000_000_000, FilesPerS: 100, Chunks: 250},
		Search:  SearchStats{Queries: 300, Latency: LatencyStats{Count: 900, P50: 2_000_000}},
		Quality: Metrics{Queries: 200, RecallAt1: 1, RecallAt5: 1, RecallAt10: 1, MRR: 1},
		Ambiguous: AmbiguousMetrics{
			Queries: 100, ContainmentAt10: 0.86, MRR: 0.39, RandomBaselineMRR: 0.46,
		},
	}
	md := RenderMarkdown(r, StageAll, time.Now())

	for _, want := range []string{
		"Cortex Benchmark", "### 语料", "### 索引", "### 检索延迟",
		"### 检索质量", "### 参考指标", "Recall@1", "P95", "复现命令",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q", want)
		}
	}
	if !strings.Contains(md, "cortex-bench all") {
		t.Errorf("markdown should include reproduction command")
	}

	j := ToJSON(r, StageAll, time.Now())
	if j.Quality.RecallAt1 != 1 {
		t.Errorf("json quality not carried over")
	}
	if j.Search.P50ms != 2 {
		t.Errorf("json p50 should be 2ms, got %v", j.Search.P50ms)
	}
}

func TestStageFiltering(t *testing.T) {
	r := &Result{Config: Config{Documents: 1, Seed: 1, TopK: 10, Repeat: 1, QueriesPerDoc: 1}}

	idxOnly := RenderMarkdown(r, StageIndex, time.Now())
	if !strings.Contains(idxOnly, "### 索引") {
		t.Error("index stage should include index section")
	}
	if strings.Contains(idxOnly, "### 检索质量") {
		t.Error("index stage should not include quality section")
	}
}

func TestTopDirectories(t *testing.T) {
	got := TopDirectories("some/dir", 0)
	if len(got) != 1 || got[0] != "some/dir" {
		t.Fatalf("unexpected: %v", got)
	}
}
