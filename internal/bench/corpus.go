// Package bench 提供可复现的基准测试：确定性语料、查询集与指标计算。
//
// 设计原则：
//   - 确定性：同一 seed 生成的语料与查询完全一致，任何人能复现同一组数字。
//   - 可归因：每个查询都有唯一 ground truth 文档，recall 高低能直接说明检索质量。
//   - 零依赖：只用标准库 + 本项目内部包，不联网、不读外部数据集。
package bench

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

// Doc 是语料中的一篇文档。
type Doc struct {
	// Path 是写入磁盘的绝对路径，也是检索结果的 ground truth 标识。
	Path string
	// Topic 是文档主题（中文短语），用于生成查询。
	Topic string
	// UniqueToken 只出现在这一篇文档里，保证查询有唯一正确答案。
	UniqueToken string
	// SharedTokens 是同一簇文档共享的词，用于制造「难查询」。
	SharedTokens []string
	// BodyLen 是正文大致字符数。
	BodyLen int
}

// Corpus 是一次生成的完整语料。
type Corpus struct {
	Docs      []Doc
	Dir       string
	TotalChar int
}

// CorpusOptions 控制语料规模与形态。
type CorpusOptions struct {
	// Docs 是文档数量。
	Docs int
	// Seed 固定随机种子，保证可复现。
	Seed int64
	// MinChars/MaxChars 是正文长度范围（字符数）。
	MinChars int
	MaxChars int
	// ClusterSize 是共享词簇的文档数（>=1）。
	ClusterSize int
}

func (o CorpusOptions) normalized() CorpusOptions {
	if o.Docs <= 0 {
		o.Docs = 100
	}
	// 正文必须足够长才能切出多个分块。
	// 单块文档会让检索退化成「一次命中」，recall 永远 1.0，测不出任何回归。
	if o.MinChars <= 0 {
		o.MinChars = 1500
	}
	if o.MaxChars <= o.MinChars {
		o.MaxChars = o.MinChars + 2000
	}
	if o.ClusterSize <= 0 {
		o.ClusterSize = 5
	}
	return o
}

// topics 是一组中文主题词，用于生成语义可读的文档与查询。
var topics = []string{
	"检索增强生成", "向量数据库选型", "反向索引结构", "分块策略", "语义重排",
	"增量索引", "冷启动优化", "多租户隔离", "权限模型", "审计日志",
	"中文分词", "同义词扩展", "查询改写", "缓存穿透", "写放大",
	"容灾备份", "灰度发布", "可观测性", "成本控制", "数据脱敏",
	"知识图谱", "实体链接", "文档解析", "PDF 抽取", "表格识别",
	"嵌入模型", "量化压缩", "近似最近邻", "召回评估", "标注体系",
}

// sharedVocab 是共享词池。
//
// 规模决定了「最多能支持多少互不重复的词对」：C(n,2)。
// 24 个词 = 276 个唯一词对，按每簇 5 篇算，可覆盖 1380 篇文档。
// 若语料更大，词对会开始重复，歧义查询的候选集会变大 —— 此时应调大本列表，
// 或接受该指标的退化（TestSharedPairsDoNotRepeatWithinCorpus 会守住这个边界）。
var sharedVocab = []string{
	"延迟", "吞吐", "召回率", "准确率", "索引构建", "内存占用",
	"磁盘 IO", "并发控制", "事务隔离", "数据一致性", "超时重试", "限流降级",
	"缓存命中", "冷启动", "写放大", "读写分离", "分片策略", "副本同步",
	"快照恢复", "权限校验", "审计追踪", "灰度发布", "压测基线", "容量规划",
}

var fillers = []string{
	"本节说明设计取舍与实现细节，并给出可复现的验证方式。",
	"实际部署时需要关注资源上限与故障恢复路径。",
	"我们对比了若干方案，最终选择在当前约束下最稳妥的一种。",
	"该模块的行为由配置驱动，默认值面向单机小规模场景。",
	"后续计划补充更多真实数据下的观测结果。",
	"接口保持向后兼容，旧版本客户端无需改造即可接入。",
	"测试覆盖了正常路径、边界条件与异常输入。",
	"性能瓶颈主要集中在磁盘随机读与序列化开销。",
}

// sharedPairs 枚举 sharedVocab 的所有唯一两词组合（C(n,2) 个）。
//
// 为什么不能简单地用 (vocab[c%n], vocab[(c+k)%n])：
// 那样词对会随簇号周期性重复，不同簇共用同一组词，
// 导致「共享词查询」的候选集比预期大一倍以上，指标失去意义。
func sharedPairs() [][2]string {
	n := len(sharedVocab)
	pairs := make([][2]string, 0, n*(n-1)/2)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			pairs = append(pairs, [2]string{sharedVocab[i], sharedVocab[j]})
		}
	}
	return pairs
}

// GenerateCorpus 生成确定性语料并写入 dir。
// dir 会被创建；同名文件会被覆盖。
func GenerateCorpus(dir string, opts CorpusOptions) (*Corpus, error) {
	o := opts.normalized()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create corpus dir: %w", err)
	}

	rng := rand.New(rand.NewSource(o.Seed))
	c := &Corpus{Dir: dir, Docs: make([]Doc, 0, o.Docs)}

	for i := 0; i < o.Docs; i++ {
		topic := topics[rng.Intn(len(topics))]
		// 唯一词用纯字母编码，避免数字与文中其它数字（段号等）撞车。
		// 早期用「专有标识%04d」且标题带「第 N 篇」序号，导致 doc-1001 的标题
		// 出现「第 1002 篇」，与 doc-1002 的唯一词碰撞，两者互抢排名 ——
		// 那会让 recall@1 的下降看起来像检索退化，实际是语料自身污染。
		unique := "zk" + alphaCode(i)
		cluster := i / o.ClusterSize
		pairs := sharedPairs()
		pair := pairs[cluster%len(pairs)]
		shared := []string{pair[0], pair[1]}

		bodyLen := o.MinChars + rng.Intn(o.MaxChars-o.MinChars+1)
		body := buildBody(rng, topic, unique, shared, bodyLen)

		name := fmt.Sprintf("doc-%04d.md", i)
		path := filepath.Join(dir, name)
		// 标题不带序号数字：序号会污染全文检索。
		content := fmt.Sprintf("# %s\n\n%s\n", topic, body)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return nil, fmt.Errorf("write %s: %w", path, err)
		}

		c.Docs = append(c.Docs, Doc{
			Path:         path,
			Topic:        topic,
			UniqueToken:  unique,
			SharedTokens: shared,
			BodyLen:      bodyLen,
		})
		c.TotalChar += len(content)
	}
	return c, nil
}

// alphaCode 把整数编码成 4 位大写字母（A-Z），保证唯一且不含数字。
// 4 位可覆盖 26^4 = 456,976 篇文档，足够基准使用。
func alphaCode(n int) string {
	const base = 26
	b := make([]byte, 4)
	for i := 3; i >= 0; i-- {
		b[i] = byte('A' + n%base)
		n /= base
	}
	return string(b)
}

// buildBody 构造正文：主题句 + 唯一 token 句 + 共享词散落 + 填充段落。
// 唯一 token 会在正文中重复出现（不同段落），这样它至少能落在其中一个分块里。
func buildBody(rng *rand.Rand, topic, unique string, shared []string, targetLen int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "本文讨论%s的落地经验。%s 是这一篇的核心标识。\n\n", topic, unique)

	para := 0
	for b.Len() < targetLen {
		para++
		fmt.Fprintf(&b, "第 %d 段：", para)
		switch {
		case para%5 == 0:
			// 每 5 段复述一次唯一 token，保证跨分块可检索
			fmt.Fprintf(&b, "再次强调 %s 的上下文：它与%s的其它部分需要一起阅读。", unique, topic)
		case para%2 == 0:
			fmt.Fprintf(&b, "%s 相关的实现细节需要结合%s一起考虑。", topic, shared[para%len(shared)])
		default:
			fmt.Fprintf(&b, "围绕%s展开时，%s是常见的观测指标。", shared[para%len(shared)], shared[(para+1)%len(shared)])
		}
		for k := 0; k < 2; k++ {
			fmt.Fprintf(&b, "%s", fillers[rng.Intn(len(fillers))])
		}
		b.WriteString("\n\n")
	}
	return b.String()
}

// Query 是一条基准查询及其 ground truth。
type Query struct {
	Text string
	// TargetPath 是唯一正确答案（文档路径）。
	TargetPath string
	// Kind 区分查询难度：unique（唯一词）/ topic（主题词）/ shared（共享词，答案不唯一）。
	Kind string
}

// BuildQueries 基于语料生成查询集。
//
// 三种难度：
//   - unique：包含"只出现在该文档"的 token，答案唯一，用于衡量 recall@k。
//   - topic ：主题词 + 唯一 token，更接近真实自然语言查询。
//   - shared：只用簇内共享词，答案不唯一，用于观察排序是否稳定（不参与 recall 断言）。
func BuildQueries(c *Corpus, perDoc int) []Query {
	if perDoc <= 0 {
		perDoc = 1
	}
	out := make([]Query, 0, len(c.Docs)*perDoc)
	for _, d := range c.Docs {
		out = append(out, Query{Text: d.UniqueToken, TargetPath: d.Path, Kind: "unique"})
		if perDoc >= 2 {
			out = append(out, Query{
				Text:       strings.Join([]string{d.Topic, d.UniqueToken}, " "),
				TargetPath: d.Path,
				Kind:       "topic",
			})
		}
		if perDoc >= 3 && len(d.SharedTokens) > 0 {
			out = append(out, Query{
				Text:       strings.Join(d.SharedTokens, " "),
				TargetPath: d.Path,
				Kind:       "shared",
			})
		}
	}
	return out
}
