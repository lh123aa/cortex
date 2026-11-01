package bench

import (
	"testing"
	"time"
)

func TestEvaluatePerfectRanking(t *testing.T) {
	queries := []Query{
		{Text: "q1", TargetPath: "/a.md", Kind: "unique"},
		{Text: "q2", TargetPath: "/b.md", Kind: "unique"},
	}
	results := [][]string{
		{"/a.md", "/x.md"},
		{"/b.md", "/y.md"},
	}
	m := Evaluate(queries, results)
	if m.Queries != 2 {
		t.Fatalf("Queries = %d, want 2", m.Queries)
	}
	if m.RecallAt1 != 1.0 || m.MRR != 1.0 {
		t.Fatalf("perfect ranking should score 1.0, got recall@1=%.3f mrr=%.3f", m.RecallAt1, m.MRR)
	}
}

func TestEvaluateRanking(t *testing.T) {
	queries := []Query{
		{TargetPath: "target", Kind: "unique"},
	}
	// target 排第 3
	results := [][]string{{"a", "b", "target"}}
	m := Evaluate(queries, results)

	if m.RecallAt1 != 0 {
		t.Errorf("Recall@1 = %.3f, want 0", m.RecallAt1)
	}
	if m.RecallAt5 != 1 {
		t.Errorf("Recall@5 = %.3f, want 1", m.RecallAt5)
	}
	if want := 1.0 / 3.0; m.MRR != want {
		t.Errorf("MRR = %.4f, want %.4f", m.MRR, want)
	}
}

func TestEvaluateMiss(t *testing.T) {
	queries := []Query{{TargetPath: "target", Kind: "unique"}}
	results := [][]string{{"a", "b"}}
	m := Evaluate(queries, results)

	if m.Misses != 1 {
		t.Errorf("Misses = %d, want 1", m.Misses)
	}
	if m.RecallAt10 != 0 || m.MRR != 0 {
		t.Errorf("miss should score 0, got %.3f / %.3f", m.RecallAt10, m.MRR)
	}
}

func TestEvaluateSkipsSharedQueries(t *testing.T) {
	queries := []Query{
		{TargetPath: "/a.md", Kind: "unique"},
		{TargetPath: "/b.md", Kind: "shared"}, // 不该计入
	}
	results := [][]string{{"/a.md"}, {"/b.md"}}
	m := Evaluate(queries, results)

	if m.Queries != 1 {
		t.Fatalf("shared queries must be excluded, Queries = %d", m.Queries)
	}
}

func TestEvaluateEmpty(t *testing.T) {
	m := Evaluate(nil, nil)
	if m.Queries != 0 {
		t.Fatalf("empty input should give zero metrics, got %+v", m)
	}
}

func TestEvaluateMismatchedLengths(t *testing.T) {
	queries := []Query{
		{TargetPath: "/a.md", Kind: "unique"},
		{TargetPath: "/b.md", Kind: "unique"},
	}
	results := [][]string{{"/a.md"}} // 少一条
	m := Evaluate(queries, results)
	if m.Queries != 1 {
		t.Fatalf("should only evaluate the overlap, got %d", m.Queries)
	}
}

func TestAmbiguousMetrics(t *testing.T) {
	queries := []Query{
		{TargetPath: "a", Kind: "shared"},
		{TargetPath: "b", Kind: "shared"},
		{TargetPath: "c", Kind: "unique"}, // 忽略
	}
	results := [][]string{
		{"a", "x"}, // 排名 1
		{"x", "b"}, // 排名 2
		{"c"},
	}
	m := EvaluateAmbiguous(queries, results, 5)

	if m.Queries != 2 {
		t.Fatalf("Queries = %d, want 2", m.Queries)
	}
	if m.ContainmentAt10 != 1.0 {
		t.Errorf("ContainmentAt10 = %.3f, want 1.0", m.ContainmentAt10)
	}
	want := (1.0 + 0.5) / 2.0
	if diff := m.MRR - want; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("MRR = %.4f, want %.4f", m.MRR, want)
	}
	if m.RandomBaselineMRR <= 0 {
		t.Errorf("random baseline should be positive, got %.3f", m.RandomBaselineMRR)
	}
}

func TestRankOf(t *testing.T) {
	list := []string{"a", "b", "c"}
	if got := rankOf("b", list); got != 2 {
		t.Errorf("rankOf(b) = %d, want 2", got)
	}
	if got := rankOf("z", list); got != 0 {
		t.Errorf("rankOf(z) = %d, want 0", got)
	}
}

func TestLatencyPercentiles(t *testing.T) {
	samples := make([]time.Duration, 100)
	for i := range samples {
		samples[i] = time.Duration(i+1) * time.Millisecond
	}
	st := Latency(samples)

	if st.Count != 100 {
		t.Fatalf("Count = %d, want 100", st.Count)
	}
	if st.P50 < 49*time.Millisecond || st.P50 > 52*time.Millisecond {
		t.Errorf("P50 = %v, want ~50ms", st.P50)
	}
	if st.P95 < 94*time.Millisecond || st.P95 > 97*time.Millisecond {
		t.Errorf("P95 = %v, want ~95ms", st.P95)
	}
	if st.Max != 100*time.Millisecond {
		t.Errorf("Max = %v, want 100ms", st.Max)
	}
}

func TestLatencyEmpty(t *testing.T) {
	st := Latency(nil)
	if st.Count != 0 || st.P50 != 0 || st.Max != 0 {
		t.Fatalf("empty latency should be zero, got %+v", st)
	}
}

func TestLatencySingleSample(t *testing.T) {
	st := Latency([]time.Duration{5 * time.Millisecond})
	if st.P50 != 5*time.Millisecond || st.P95 != 5*time.Millisecond || st.Max != 5*time.Millisecond {
		t.Fatalf("single sample percentiles should equal the sample, got %+v", st)
	}
}
