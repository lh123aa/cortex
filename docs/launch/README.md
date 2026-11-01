# Cortex 实测与宣发材料包

> 版本：v4.0.0（commit `65100e5`）｜ 实测日期：2026-10-30 ｜ 环境：Windows 11 / x86_64 / 本地 SSD

这个目录是**可对外使用**的实测与宣发材料，所有数字都来自本机实跑，原始数据在 `raw/`。

## 文件索引

| 文件 | 内容 |
|---|---|
| [01-install-and-run.md](01-install-and-run.md) | 本地安装、首次运行、三组语料索引实录、发现的问题 |
| [02-benchmark.md](02-benchmark.md) | 索引/检索/写操作/内存/启动的完整实测数据与瓶颈分析 |
| [03-comparison.md](03-comparison.md) | 与 XERJ / AnythingLLM / Dify / Open WebUI / Mem0 的功能与定价对比 |
| [04-launch-copy.md](04-launch-copy.md) | README 首屏、HN、Twitter、掘金/知乎/V2EX 文案、FAQ、宣发红线 |
| [05-performance-optimization.md](05-performance-optimization.md) | **性能优化报告（v4.0.0 → v4.0.1：索引 3.1× / 检索 P95 8.7×）** |
| `raw/` | 原始 JSON 测量结果 + 控制台与审计接口截图 |

## 核心数字速查（v4.0.1，优化后）

| 指标 | 数值 | 条件 |
|---|---|---|
| 冷启动 | **0.36 s** | 27,157 文档 / 1.2 GB 库 |
| 检索 P50 | **0.43 ms** | 同上，HTTP 持久连接 |
| 检索 P95 | **2.35 ms** | 同上 |
| 检索吞吐 | **1,440 QPS** | 单实例串行 |
| 响应体（limit=10） | **5.8 KB** | 优化前 1,914 KB |
| 常驻内存 | **53.3 MB** | 27k 文档 / 84k 分块 |
| 索引吞吐 | **328–365 files/s** | 合成语料 1k/5k |
| 单二进制 | **37.4 MB** | 开源版与企业版一致 |
| 审计开销 | **+0.17 ms/次写** | 企业版 vs 开源版，100 次采样 |
| 召回率 | **Recall@10 = 1.000** | 合成语料，ground truth 可验证 |

> v4.0.0 基线数字（优化前：P50 1.01 ms / P95 20.5 ms / 118 files/s）见 [02-benchmark.md](02-benchmark.md)。
> 缓存命中率实测 100%（303/303），所以 HTTP 延迟主要反映序列化与传输成本。

## 原始数据

| 文件 | 说明 |
|---|---|
| `raw/perf-A-keepalive.json` | 语料 A（本仓库 207 文档）检索延迟 |
| `raw/perf-B-keepalive.json` | 语料 B（XERJ 文档 277 文档 / 3,060 分块） |
| `raw/perf-C-keepalive.json` | 语料 C（真实库 27,157 文档 / 84,562 分块） |
| `raw/perf-C-cold-keepalive.json` | 语料 C 冷查询（单轮，不命中缓存） |
| `raw/write-public.json` | 开源版写操作延迟 |
| `raw/write-ee.json` | 企业版（含审计中间件）写操作延迟 |
| `raw/screenshot-console.png` | 控制台实测截图（27k 文档 + Team 套餐） |
| `raw/screenshot-ee-audit.png` | 企业版 `/ee/audit` 接口截图 |

> 早期的 `perf-A.json` / `perf-B.json` / `perf-C.json` 是用 PowerShell/urllib 逐请求新建连接测的，
> 客户端开销严重污染结果（stats 端点都测出 38 ms）。**以 `*-keepalive.json` 为准。**

## 使用约定

1. 对外引用数字时，必须带条件（语料规模、是否缓存、是否单实例）。
2. 不引用合成基准的 recall 作为「语义检索质量」。
3. 与竞品对比时标注「对方为官方自测，环境不同，仅量级参考」。
4. 短板已在各文档「诚实的短板」章节列出，宣发时不要删掉。
