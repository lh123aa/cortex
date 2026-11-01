// cortex-bench 是 Cortex 的可复现基准工具。
//
// 设计目标是「任何人都能用一条命令复现同一组数字」：
//   - 语料由固定 seed 生成，不依赖外部数据集
//   - 每个查询有唯一 ground truth，召回率可解释
//   - 结果同时输出 JSON（机器读）与 Markdown（贴 README）
//
// 用法：
//
//	go run ./cmd/cortex-bench all --docs 500 --workers 8
//	go run ./cmd/cortex-bench all --docs 500 --out benchmarks/results/latest.md --json-out benchmarks/results/latest.json
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lh123aa/cortex/internal/bench"
	"github.com/lh123aa/cortex/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	cmd := os.Args[1]
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	docs := fs.Int("docs", 500, "语料文档数")
	seed := fs.Int64("seed", 20261023, "语料随机种子（同 seed 结果可复现）")
	workers := fs.Int("workers", 8, "索引并发 worker 数")
	queriesPerDoc := fs.Int("queries-per-doc", 3, "每篇文档生成的查询数（1-3）")
	topK := fs.Int("top-k", 10, "每次检索返回条数（>=10）")
	repeat := fs.Int("repeat", 3, "每条查询重复次数（用于延迟统计）")
	workDir := fs.String("dir", "", "工作目录（默认 _temp/bench）")
	outMD := fs.String("out", "", "Markdown 结果输出路径")
	outJSON := fs.String("json-out", "", "JSON 结果输出路径")

	_ = fs.Parse(os.Args[2:])

	root := *workDir
	if root == "" {
		root = filepath.Join("_temp", "bench")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		fatalf("create work dir: %v", err)
	}

	cfg := bench.Config{
		CorpusDir:     filepath.Join(root, "corpus"),
		DBPath:        filepath.Join(root, "bench.db"),
		Documents:     *docs,
		Seed:          *seed,
		Workers:       *workers,
		QueriesPerDoc: *queriesPerDoc,
		TopK:          *topK,
		Repeat:        *repeat,
	}

	// 每次运行前清理旧库与语料，保证冷启动、结果可比
	switch cmd {
	case "all", "index", "search", "quality":
		if err := os.RemoveAll(cfg.DBPath); err != nil && !os.IsNotExist(err) {
			fatalf("clean db: %v", err)
		}
		if err := os.RemoveAll(cfg.CorpusDir); err != nil && !os.IsNotExist(err) {
			fatalf("clean corpus: %v", err)
		}
	case "help", "-h", "--help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		usage()
		os.Exit(2)
	}

	logf := func(format string, args ...any) {
		fmt.Printf("  "+format+"\n", args...)
	}

	fmt.Printf("cortex-bench %s\n", version.Full())
	fmt.Println("──────────────────────────────────────────────")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	upTo := stageOf(cmd)
	res, err := bench.RunAll(ctx, logf, cfg)
	if err != nil {
		fatalf("benchmark failed: %v", err)
	}

	fmt.Println("──────────────────────────────────────────────")
	md := bench.RenderMarkdown(res, upTo, time.Now().UTC())
	fmt.Println(md)

	if *outMD != "" {
		if err := os.MkdirAll(filepath.Dir(*outMD), 0o755); err != nil {
			fatalf("create output dir: %v", err)
		}
		if err := os.WriteFile(*outMD, []byte(md), 0o644); err != nil {
			fatalf("write markdown: %v", err)
		}
		fmt.Printf("\nMarkdown 已写入: %s\n", *outMD)
	}
	if *outJSON != "" {
		if err := os.MkdirAll(filepath.Dir(*outJSON), 0o755); err != nil {
			fatalf("create output dir: %v", err)
		}
		b, err := json.MarshalIndent(bench.ToJSON(res, upTo, time.Now().UTC()), "", "  ")
		if err != nil {
			fatalf("marshal json: %v", err)
		}
		if err := os.WriteFile(*outJSON, append(b, '\n'), 0o644); err != nil {
			fatalf("write json: %v", err)
		}
		fmt.Printf("JSON 已写入: %s\n", *outJSON)
	}
}

func stageOf(cmd string) bench.Stage {
	switch cmd {
	case "index":
		return bench.StageIndex
	case "search":
		return bench.StageSearch
	case "quality":
		return bench.StageQuality
	default:
		return bench.StageAll
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `cortex-bench — Cortex 可复现基准

用法:
  cortex-bench <command> [flags]

命令:
  all       生成语料 → 索引 → 检索 → 质量评估（完整流程）
  index     只测索引吞吐与分块
  search    只测检索延迟
  quality   只测召回率（recall@k / MRR）
  help      显示本帮助

常用参数:
  --docs N            语料文档数（默认 500）
  --seed N            随机种子（同 seed 完全可复现，默认 20261023）
  --workers N         索引并发（默认 8）
  --queries-per-doc N 每篇生成的查询数（1-3，默认 2）
  --repeat N          每条查询重复次数（默认 3）
  --out FILE.md       同时输出 Markdown 报告
  --json-out FILE.json 同时输出 JSON 结果

示例:
  cortex-bench all --docs 500 --out benchmarks/results/latest.md
  cortex-bench all --docs 2000 --workers 16 --json-out benchmarks/results/2k.json
`)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "\n❌ "+format+"\n", args...)
	os.Exit(1)
}
