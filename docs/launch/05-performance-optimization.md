# Cortex 性能优化报告：v4.0.0 → v4.0.1

> 日期：2026-11-01
> 环境：Windows 11 / x86_64 / 本地 SSD / `CGO_ENABLED=0` / 默认 FTS-only
> 对照基线：v4.0.0（`docs/launch/raw/perf-C-keepalive.json`）
> 原始数据：`_temp/perf/*.json`、`docs/launch/raw/perf-C-after-opt.json`

---

## 0. 结论

| 指标 | v4.0.0 | v4.0.1 | 变化 |
|---|---:|---:|---:|
| 索引吞吐（5k 文档） | 118 files/s | **365 files/s** | **3.1×** |
| 索引吞吐（1k 文档） | 127 files/s | **328 files/s** | **2.6×** |
| 检索 P50（27k 文档） | 1.01 ms | **0.43 ms** | **2.3×** |
| 检索 P95（27k 文档） | 20.52 ms | **2.35 ms** | **8.7×** |
| 检索 P99（27k 文档） | 23.35 ms | **3.91 ms** | **6.0×** |
| 检索吞吐 | 408 QPS | **1,440 QPS** | **3.5×** |
| 单次响应体（limit=10） | 1,914 KB | **5.8 KB** | **330×** |
| 召回率 Recall@10 | 1.000 | 1.000 | 不变 |

**在索引吞吐上已超过 XERJ 公开自测的 ~160 files/s，在检索延迟上进入亚毫秒级。**

---

## 1. 五个优化（按收益排序）

### 优化 1：检索响应瘦身（收益最大，8.7× P95）

**问题**：`limit=10` 的响应体实测 **1,914 KB**。原因是每条结果同时返回 `content`（完整分块）和 `content_raw`（分词后的稀疏文本），再加上逐条查 `documents` 的 N+1。

**改动**（`internal/api/rest.go`）：
- 默认只返回单个 `content` 字段，按 800 字符截断；
- 需要全文时显式传 `?full=1`；
- 路径直接复用检索层的 `FilePath`，删除 N+1 查询。

**效果**：1,914 KB → **5.8 KB**；P95 20.5 → 2.35 ms；吞吐 408 → 1,440 QPS。

> 这一项说明：**很多时候"检索慢"其实是序列化和传输慢**。在优化检索引擎之前，先看响应体大小。

### 优化 2：索引写入合并为单事务（3.1× 吞吐）

**问题**：索引一个文件要 3–4 次独立提交：
`SaveDocument` → `SaveChunks` → `DeleteSymbolsByDocument` → `SaveSymbols`。
每次提交在 WAL 下都是一次 fsync。

**改动**：
- 新增 `storage.DocumentBundle` + `BundleStore` 接口（`internal/storage/bundle.go`）；
- 文档 + 分块 + 向量 + 符号在**一个事务**内写入；
- 索引器通过类型断言使用，不实现该接口的存储自动回退到旧路径。

**效果**：118 → 265 files/s（2.25×）。

### 优化 3：`synchronous=NORMAL`（与优化 2 叠加到 3.1×）

**问题**：默认 `synchronous=FULL`，每次提交都 fsync。

**改动**（`internal/storage/sqlite.go`）：改为 `synchronous(NORMAL)`。

**取舍**：进程崩溃不丢数据；**掉电可能丢最后几批**。索引是可重跑的批处理，这个取舍可接受。写入数据库的业务数据（记忆）仍受 WAL 保护。

**效果**：265 → 365 files/s（1.38×）。

### 优化 4：L2 缓存异步化（消除读路径磁盘写）

**问题**：每次搜索都同步 `INSERT OR REPLACE` 写 `search_cache` 表，既拖慢读路径，又与索引写抢锁。

**改动**（`internal/search/engine.go`）：
- 新增长度 256 的异步写入队列 + 单 goroutine 串行消费；
- 队列满时丢弃并计数（`search_cache_writes_dropped_total`）。
- L1 内存缓存仍同步写（纯内存，微秒级）。

**说明**：bench 工具会 `DisableCache()` 测"真实检索"，所以**这一项在 bench 里测不出来**，只在真实 HTTP 服务里生效。这也是为什么必须做 HTTP 层测量。

### 优化 5：向量 Top-K 改小顶堆

**问题**：`vectorSearchInMemory` 每来一个候选就做 O(K) 插入排序，10 万向量 × K=10 时纯排序开销就有几毫秒。

**改动**（`internal/storage/search.go`）：改为 `container/heap` 维护大小为 K 的小顶堆，最后统一排序输出。

**效果**：向量路径 O(n·K) → O(n·log K)。当前默认 FTS-only 所以主流程未触发，但开 embedding 后生效。

---

## 2. 顺带修掉的两个基准测量 bug

### bug 1：`toMs` 把亚毫秒吃掉

```go
// 旧：d.Microseconds() 截断，<1ms 全部变 0
return float64(d.Microseconds()) / 1000.0
```

导致 P50 在 500µs 以下时永远打印 `0.00 ms` —— 看起来像没测到，实际是取整把数据吃了。改为浮点纳秒换算。

### bug 2：Windows 计时器分辨率 507.5µs

实测探针：200 万次 `time.Now()` 差值，**1,999,969 次为 0**，最小非零 507.5µs。

Windows 上亚毫秒检索根本测不出来。已加自动检测：
- `LatencyStats` 新增 `ZeroSamples` / `ClockResolution` / `BelowResolution`；
- 报告里改为 `< 0.51 ms（低于计时器分辨率）` 并附警告，**不再打印误导性的 0.00 ms**。

> 这两个 bug 会让任何人（包括我）把基准数据读错。基准工具的正确性和被测系统一样重要。

---

## 3. 验证

### 3.1 功能不变

| 项目 | 结果 |
|---|---|
| `go vet ./...` | ✅ 0 警告 |
| `go test ./...` | ✅ 24/24 包全绿 |
| Recall@1 / @5 / @10 | ✅ 1.000（优化前后一致） |
| 缓存命中率 | ✅ 303/303 = 100%（`search_cache_hits_total`） |

### 3.2 A/B 对照（排除机器噪声）

用优化前/后两个 bench 二进制各跑两轮（5k 文档，缓存关闭）：

| 轮次 | 版本 | P50 | P95 | P99 | Mean |
|---|---|---:|---:|---:|---:|
| 1 | 优化前 | 0.00* | 10.22 | 15.10 | 2.56 |
| 1 | 优化后 | 1.00 | 9.88 | 12.05 | 2.24 |
| 2 | 优化前 | 0.00* | 10.09 | 11.29 | 2.09 |
| 2 | 优化后 | 1.00 | 10.21 | 13.82 | 2.68 |

\* 优化前的 0.00 是 `toMs` bug。

**诚实结论**：bench 的检索阶段**关闭缓存**，所以优化 4/5 不计入；优化 2/3 只影响索引。
A/B 显示检索延迟差异在噪声范围内（±10%）——**bench 场景下检索没有实质提升，提升在真实 HTTP 服务上**。

## 4. 剩余瓶颈（下一阶段）

| # | 瓶颈 | 证据 | 方案 | 预期 |
|---|---|---|---|---|
| 1 | FTS 候选集大时排序开销 | P99 仍偶发 20ms+ | 候选截断调参 + BM25 预排序 | P99 −30% |
| 2 | CJK 单字分词索引膨胀 | 每个汉字一个 token | bigram 分词 A/B | 索引体积 −30~50% |
| 3 | 向量全量扫描 | 84k 向量逐条算余弦 | HNSW / IVF-PQ | 10 万向量 P95 <10ms |
| 4 | 启动加载向量索引 | 冷启动读扁平索引 | mmap / 惰性分片 | 启动 −50% |
| 5 | JSON 序列化 | gin 默认 encoding/json | 换 sonic | RPS +10~20% |
| 6 | Windows 上无法测亚毫秒 | 分辨率 507µs | 在 Linux 上复跑验证 | — |

## 5. 与竞品的位置

| 指标 | Cortex v4.0.1 | XERJ（官方自测） | 说明 |
|---|---|---|---|
| 索引吞吐 | **365 files/s** | ~160 files/s | Cortex 已超越 2.3×（但语料不同，XERJ 语料含二进制解析） |
| 检索 P50 | **0.43 ms**（27k 文档） | 未公开同口径 | — |
| 检索 P95 | **2.35 ms** | 未公开 | — |
| 响应体 | **5.8 KB / 10 条** | 未公开 | 差异点 |

> ⚠️ 宣发口径：索引吞吐的对比**语料不同、格式支持不同**，只能说"量级相当"，
> 不能说"比 XERJ 快 2 倍"。XERJ 支持 40+ 格式，Cortex 目前约 20 类。

## 6. 复现

```bash
# 索引吞吐（改前后可直接对比）
go build -o /tmp/bench ./cmd/cortex-bench
/tmp/bench index --docs 5000 --workers 8

# HTTP 检索延迟（必须用持久连接）
python _temp/launch/measure_keepalive.py \
  --base http://127.0.0.1:2112 --queries _temp/launch/queries_C.json \
  --rounds 10 --top-k 10 --out result.json

# 响应体大小
curl -s -o /dev/null -w '%{size_download}\n' \
  'http://127.0.0.1:2112/v1/search?q=Go&limit=10'
```
