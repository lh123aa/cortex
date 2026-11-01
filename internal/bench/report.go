package bench

import (
	"fmt"
	"strings"
	"time"
)

// Stage 表示本次运行覆盖的阶段。
type Stage string

const (
	StageIndex   Stage = "index"
	StageSearch  Stage = "search"
	StageQuality Stage = "quality"
	StageAll     Stage = "all"
)

// RenderMarkdown 把结果渲染成可直接贴进 README 的 Markdown。
func RenderMarkdown(r *Result, stage Stage, at time.Time) string {
	var b strings.Builder

	fmt.Fprintf(&b, "## Cortex Benchmark\n\n")
	fmt.Fprintf(&b, "> 生成时间（UTC）：%s\n", at.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "> 复现命令：`go run ./cmd/cortex-bench all --docs %d --workers %d --seed %d`\n",
		r.Config.Documents, r.Config.Workers, r.Config.Seed)
	if r.Config.TopK != 10 || r.Config.Repeat != 3 {
		fmt.Fprintf(&b, "> 参数：top_k=%d, repeat=%d, queries_per_doc=%d\n",
			r.Config.TopK, r.Config.Repeat, r.Config.QueriesPerDoc)
	}
	b.WriteString("\n")

	// 语料
	b.WriteString("### 语料\n\n")
	b.WriteString("| 指标 | 值 |\n|---|---|\n")
	fmt.Fprintf(&b, "| 文档数 | %d |\n", r.Corpus.Documents)
	fmt.Fprintf(&b, "| 总字符数 | %d |\n", r.Corpus.TotalChars)
	fmt.Fprintf(&b, "| 生成方式 | 确定性合成（seed=%d） |\n\n", r.Config.Seed)

	// 索引
	if stage == StageAll || stage == StageIndex {
		b.WriteString("### 索引\n\n")
		b.WriteString("| 指标 | 值 |\n|---|---|\n")
		fmt.Fprintf(&b, "| 文件数 | %d |\n", r.Index.Files)
		fmt.Fprintf(&b, "| 成功 / 失败 / 跳过 | %d / %d / %d |\n", r.Index.Indexed, r.Index.Failed, r.Index.Skipped)
		fmt.Fprintf(&b, "| 耗时 | %s |\n", r.Index.Duration.Round(time.Millisecond))
		fmt.Fprintf(&b, "| 吞吐 | %.1f files/s |\n", r.Index.FilesPerS)
		fmt.Fprintf(&b, "| 产出分块 | %d |\n\n", r.Index.Chunks)
	}

	// 检索
	if stage == StageAll || stage == StageSearch {
		b.WriteString("### 检索延迟（hybrid，缓存关闭）\n\n")
		b.WriteString("| 指标 | 值 |\n|---|---|\n")
		fmt.Fprintf(&b, "| 查询数 | %d |\n", r.Search.Queries)
		fmt.Fprintf(&b, "| 采样数 | %d |\n", r.Search.Latency.Count)
		fmt.Fprintf(&b, "| P50 | %s |\n", latencyCell(r.Search.Latency.P50, r.Search.Latency))
		fmt.Fprintf(&b, "| P95 | %s |\n", latencyCell(r.Search.Latency.P95, r.Search.Latency))
		fmt.Fprintf(&b, "| P99 | %s |\n", latencyCell(r.Search.Latency.P99, r.Search.Latency))
		fmt.Fprintf(&b, "| Mean | %s |\n", latencyCell(r.Search.Latency.Mean, r.Search.Latency))
		fmt.Fprintf(&b, "| 吞吐 | %.0f QPS |\n\n", r.Search.QueriesPerS)

		// 时钟分辨率不足时必须显式告知，否则「P50 0.00 ms」会被当成真实数据引用。
		if r.Search.Latency.BelowResolution {
			fmt.Fprintf(&b,
				"> ⚠️ **P50 低于本机计时器分辨率**（约 %s，%d/%d 个样本测为 0）。\n"+
					"> 这说明检索本身快于时钟粒度，真实 P50 只能用 `Mean` 与 P95 推断，**不要把 0.00 ms 当作实测值对外引用**。\n"+
					"> 需要精确亚毫秒数据时，请在 Linux/macOS 上复跑，或在进程内用 `time.Now()` 的高精度时钟累加多次取平均。\n\n",
				ms(r.Search.Latency.ClockResolution),
				r.Search.Latency.ZeroSamples, r.Search.Latency.Count)
		}
	}

	// 质量
	if stage == StageAll || stage == StageQuality {
		b.WriteString("### 检索质量（唯一答案查询）\n\n")
		b.WriteString("| 指标 | 值 |\n|---|---|\n")
		fmt.Fprintf(&b, "| 评估查询数 | %d |\n", r.Quality.Queries)
		fmt.Fprintf(&b, "| Recall@1 | %.3f |\n", r.Quality.RecallAt1)
		fmt.Fprintf(&b, "| Recall@5 | %.3f |\n", r.Quality.RecallAt5)
		fmt.Fprintf(&b, "| Recall@10 | %.3f |\n", r.Quality.RecallAt10)
		fmt.Fprintf(&b, "| MRR | %.3f |\n", r.Quality.MRR)
		fmt.Fprintf(&b, "| Top-10 未命中 | %d |\n\n", r.Quality.Misses)

		// 歧义查询：候选文档证据完全等价，排序优劣没有客观标准，
		// 因此这里只作为「参考指标」展示，不参与质量判定。
		if r.Ambiguous.Queries > 0 {
			b.WriteString("### 参考指标（合成歧义查询）\n\n")
			b.WriteString("| 指标 | 值 | 说明 |\n|---|---|---|\n")
			fmt.Fprintf(&b, "| 查询数 | %d | 每簇约 5 篇候选 |\n", r.Ambiguous.Queries)
			fmt.Fprintf(&b, "| 目标进入 Top-10 | %.3f | 目标文档是否出现在前 10 |\n", r.Ambiguous.ContainmentAt10)
			fmt.Fprintf(&b, "| MRR | %.3f | 目标文档的平均倒数排名 |\n", r.Ambiguous.MRR)
			fmt.Fprintf(&b, "| 随机基线 MRR | %.3f | 完全随机排序的期望值 |\n\n", r.Ambiguous.RandomBaselineMRR)

			b.WriteString("> **仅供参考，不参与质量判定。** 这组查询的候选文档证据完全等价，\n")
			b.WriteString("> 任何排序都无法客观优于另一种，MRR 接近随机基线是正常结果。\n")
			b.WriteString("> 另外多词查询要求这些词落在**同一个分块**里才命中，\n")
			b.WriteString("> 所以 containment 天然低于 1.0，不代表召回缺陷。\n\n")
		}
	}

	b.WriteString("---\n")
	b.WriteString("**口径说明**：语料为固定 seed 生成的合成文档，每个查询带唯一标识词，\n")
	b.WriteString("因此 ground truth 无歧义；「共享词」类查询没有唯一答案，不计入 recall。\n")
	b.WriteString("数字只能用于同一 harness 的纵向对比（版本间回归），不要与外部产品横比。\n")
	return b.String()
}

// JSONResult 是机器可读的结果结构。
type JSONResult struct {
	GeneratedAt string           `json:"generated_at"`
	Stage       string           `json:"stage"`
	Config      Config           `json:"config"`
	Corpus      CorpusStats      `json:"corpus"`
	Index       IndexStats       `json:"index"`
	Search      JSONSearch       `json:"search"`
	Quality     Metrics          `json:"quality"`
	Ambiguous   AmbiguousMetrics `json:"ambiguous"`
}

// JSONSearch 是检索指标的 JSON 形态（延迟转成毫秒数字，便于消费）。
type JSONSearch struct {
	Queries     int     `json:"queries"`
	Samples     int     `json:"samples"`
	P50ms       float64 `json:"p50_ms"`
	P95ms       float64 `json:"p95_ms"`
	P99ms       float64 `json:"p99_ms"`
	Meanms      float64 `json:"mean_ms"`
	QueriesPerS float64 `json:"qps"`
}

// ToJSON 转换为可序列化结构。
func ToJSON(r *Result, stage Stage, at time.Time) JSONResult {
	return JSONResult{
		GeneratedAt: at.Format(time.RFC3339),
		Stage:       string(stage),
		Config:      r.Config,
		Corpus:      r.Corpus,
		Index:       r.Index,
		Search: JSONSearch{
			Queries:     r.Search.Queries,
			Samples:     r.Search.Latency.Count,
			P50ms:       toMs(r.Search.Latency.P50),
			P95ms:       toMs(r.Search.Latency.P95),
			P99ms:       toMs(r.Search.Latency.P99),
			Meanms:      toMs(r.Search.Latency.Mean),
			QueriesPerS: r.Search.QueriesPerS,
		},
		Quality:   r.Quality,
		Ambiguous: r.Ambiguous,
	}
}

func ms(d time.Duration) string {
	return fmt.Sprintf("%.2f ms", toMs(d))
}

func toMs(d time.Duration) float64 {
	// 必须用浮点纳秒换算：d.Microseconds() 会把 <1ms 的值截断为 0，
	// 于是 P50 在 500µs 以下时永远显示「0.00 ms」—— 看起来像没测到，
	// 实际是取整把数据吃掉了（曾让一次 2ms 的基准被误读成 0ms）。
	return float64(d) / float64(time.Millisecond)
}

// latencyCell 渲染单个延迟单元格；测不出来的值显式标注，而不是显示 0.00 ms。
func latencyCell(d time.Duration, st LatencyStats) string {
	if d == 0 && st.ZeroSamples > 0 {
		res := st.ClockResolution
		if res <= 0 {
			res = time.Millisecond
		}
		return fmt.Sprintf("< %.2f ms（低于计时器分辨率）", toMs(res))
	}
	return ms(d)
}
