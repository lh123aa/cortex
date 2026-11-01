package bench

import (
	"sort"
	"time"
)

// Metrics 是一组检索质量指标。
type Metrics struct {
	// Queries 是参与计算的查询数（只统计有唯一答案的查询）。
	Queries int `json:"queries"`
	// RecallAt1/5/10 是 top-k 命中率。
	RecallAt1  float64 `json:"recall_at_1"`
	RecallAt5  float64 `json:"recall_at_5"`
	RecallAt10 float64 `json:"recall_at_10"`
	// MRR 是平均倒数排名（Mean Reciprocal Rank）。
	MRR float64 `json:"mrr"`
	// Misses 是 top-10 内完全没命中的查询数。
	Misses int `json:"misses"`
}

// LatencyStats 是延迟分布。
//
// 注意 ClockResolution：Windows 的计时器分辨率约 0.5ms（本机实测 507.5µs），
// 亚毫秒的检索耗时会直接被测成 0。当 P50 小于一个时钟刻度时，
// 分位数字段没有意义，必须看 BelowResolution 与 Mean。
type LatencyStats struct {
	Count int           `json:"count"`
	P50   time.Duration `json:"p50_ns"`
	P95   time.Duration `json:"p95_ns"`
	P99   time.Duration `json:"p99_ns"`
	Max   time.Duration `json:"max_ns"`
	Mean  time.Duration `json:"mean_ns"`

	// ZeroSamples 是测得耗时为 0 的样本数（被时钟精度吃掉）。
	ZeroSamples int `json:"zero_samples"`
	// ClockResolution 是本次测量观测到的计时器最小非零增量。
	ClockResolution time.Duration `json:"clock_resolution_ns"`
	// BelowResolution 为真表示 P50 落在时钟分辨率以下，P50 数值不可信。
	BelowResolution bool `json:"below_clock_resolution"`
}

// Evaluate 计算检索质量。
//
// 参数：
//   - queries：查询及其 ground truth（只使用 Kind != "shared" 的项，
//     因为共享词查询没有唯一答案）。
//   - results：与 queries 一一对应的结果列表，每项是按排名排好的文档路径。
func Evaluate(queries []Query, results [][]string) Metrics {
	n := len(queries)
	if n != len(results) {
		n = min(n, len(results))
	}

	var hits1, hits5, hits10, misses int
	var rr float64
	evaluated := 0

	for i := 0; i < n; i++ {
		if queries[i].Kind == "shared" {
			continue
		}
		evaluated++
		target := queries[i].TargetPath
		rank := rankOf(target, results[i])
		// 注意：rank==0 表示未命中，绝不能因为 "0 <= 10" 而被算成 recall@10 命中。
		if rank == 0 {
			misses++
			continue
		}
		if rank <= 1 {
			hits1++
		}
		if rank <= 5 {
			hits5++
		}
		if rank <= 10 {
			hits10++
		}
		rr += 1.0 / float64(rank)
	}

	m := Metrics{Queries: evaluated, Misses: misses}
	if evaluated == 0 {
		return m
	}
	m.RecallAt1 = float64(hits1) / float64(evaluated)
	m.RecallAt5 = float64(hits5) / float64(evaluated)
	m.RecallAt10 = float64(hits10) / float64(evaluated)
	m.MRR = rr / float64(evaluated)
	return m
}

// AmbiguousMetrics 衡量「没有任何词能唯一确定答案」时的排序质量。
//
// 这类查询的候选集是一整个文档簇（默认 5 篇），随机排序的期望 MRR 约为 0.46
// （1/1+1/2+…+1/5 除以 5）。观测值明显高于随机，说明排序确实在用相关性
// 而不只是随便返回。
type AmbiguousMetrics struct {
	Queries int `json:"queries"`
	// ContainmentAt10 是「目标文档出现在 top-10 里」的比例。
	ContainmentAt10 float64 `json:"containment_at_10"`
	// MRR 是目标文档排名的平均倒数排名。
	MRR float64 `json:"mrr"`
	// RandomBaselineMRR 是随机排序下的期望 MRR（用于对照）。
	RandomBaselineMRR float64 `json:"random_baseline_mrr"`
}

// EvaluateAmbiguous 只统计 Kind == "shared" 的查询。
// clusterSize 用于计算随机基线。
func EvaluateAmbiguous(queries []Query, results [][]string, clusterSize int) AmbiguousMetrics {
	var m AmbiguousMetrics
	var rr float64
	var contained int

	for i, q := range queries {
		if q.Kind != "shared" || i >= len(results) {
			continue
		}
		m.Queries++
		rank := rankOf(q.TargetPath, results[i])
		if rank > 0 {
			rr += 1.0 / float64(rank)
			if rank <= 10 {
				contained++
			}
		}
	}

	if m.Queries > 0 {
		m.MRR = rr / float64(m.Queries)
		m.ContainmentAt10 = float64(contained) / float64(m.Queries)
	}
	if clusterSize > 0 {
		var base float64
		for r := 1; r <= clusterSize; r++ {
			base += 1.0 / float64(r)
		}
		m.RandomBaselineMRR = base / float64(clusterSize)
	}
	return m
}

// rankOf 返回值在列表中的排名（1-based）；0 表示未命中。
func rankOf(target string, list []string) int {
	for i, v := range list {
		if v == target {
			return i + 1
		}
	}
	return 0
}

// Latency 计算延迟分位数。
func Latency(samples []time.Duration) LatencyStats {
	st := LatencyStats{Count: len(samples)}
	if len(samples) == 0 {
		return st
	}
	sorted := make([]time.Duration, len(samples))
	copy(sorted, samples)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	var sum time.Duration
	for _, s := range sorted {
		sum += s
		if s == 0 {
			st.ZeroSamples++
		}
	}
	st.Mean = sum / time.Duration(len(sorted))
	st.P50 = percentile(sorted, 0.50)
	st.P95 = percentile(sorted, 0.95)
	st.P99 = percentile(sorted, 0.99)
	st.Max = sorted[len(sorted)-1]

	// 时钟分辨率取「最小非零样本」：它就是本机计时器能分辨的最小粒度。
	for _, s := range sorted {
		if s > 0 {
			st.ClockResolution = s
			break
		}
	}
	// P50 为 0 表示过半样本落在分辨率以下 —— 真实值未知，只能说是亚毫秒级。
	st.BelowResolution = st.P50 == 0 && st.ZeroSamples > 0
	return st
}

// percentile 用 nearest-rank 法取分位数。
func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)) * p)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	if idx < 0 {
		idx = 0
	}
	return sorted[idx]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
