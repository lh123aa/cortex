# Cortex 系统性评估与开源 / 商业化方案

> 版本：v1.0 ｜ 日期：2026-10-29
> 数据来源：本仓库实测（benchmark harness）、竞品官网公开信息（2026-10 抓取）、Git 历史
> 结论先行：**Cortex 不应该和 XERJ 拼引擎，应该做「中文优先 + 零运维 + MCP 原生记忆」的产品层；开源核心 + 私有企业版是正确路径；当前最大的技术瓶颈是 SQLite 单写者与向量线性扫描，不是功能数量。**

---

## 0. 结论摘要（TL;DR）

| 问题 | 结论 |
|---|---|
| 竞品中最像谁 | **XERJ**（本地优先 + MCP + 混合检索 + 记忆），但它是引擎层，Cortex 是产品层 |
| 真实差异化 | ①中文/国内 embedding 生态 ②纯 Go 单二进制无 CGO ③MCP 工具面（14 个）与自动注册 ④内置商业化基础设施（License/配额/RBAC） |
| 最大短板 | 检索深度（无重排/无 DSL/无聚合）、采集格式窄、规模上限（SQLite 单写者 + 线性向量） |
| 性能瓶颈 | 索引写入串行（106 files/s @5k）、向量 O(n·dim) 全扫、CJK 单字分词索引膨胀、P99 随规模线性上升 |
| 开源切分 | 核心（索引/检索/MCP/REST/CLI/控制台/基础 RBAC）开源；审计/SSO/团队/加密/HA/ES 兼容留私有版 |
| 许可策略 | 当前仓库实际是 **AGPL-3.0**（README 曾误写 MIT）。推荐迁移到 **Apache-2.0 核心 + 专有 EE**；若无法取得贡献者同意，则 AGPL + 商业双许可 |
| 付费方案 | Free / Pro ¥199 年 / Team ¥899 年（3 席）/ Enterprise ¥3 万 年起；Ed25519 离线 License + 可选 License Server；国际走 MoR，国内走 面包多/爱发电 → 支付宝/微信 |
| 现实目标 | 第一年 ARR ¥10–40 万（Pro 60–120 + Team 5–15 + Enterprise 1–3），不是 $120K |

---

## 1. 竞品地图与对比

### 1.1 对比对象

| 类别 | 对象 | 为什么选它 |
|---|---|---|
| 直接竞品 | **XERJ** | 同一需求（本地优先 + MCP + 混合检索 + 记忆），同形态（单二进制） |
| 同需求不同形态 | **Mem0**、**Khoj** | 都做「Agent 记忆」，Mem0 是托管记忆层 |
| 上位竞品 | **AnythingLLM**、**Dify**、**Open WebUI** | 都是「私有知识库 / AI 工作台」，抢同一批用户 |
| 邻近组件 | **Chroma / Qdrant** | 用户可能用它们自建，Cortex 的替代对象 |
| 国内 | **FastGPT / MaxKB / RAGFlow / 语雀 / 飞书知识库** | 中文市场真实对手 |

### 1.2 对比矩阵

| 维度 | Cortex | XERJ | AnythingLLM | Dify | Open WebUI | Mem0 |
|---|---|---|---|---|---|---|
| 语言/栈 | Go（纯静态，无 CGO） | Rust | Node + Docker | Python 全家桶 | Python + Svelte | Python |
| 部署形态 | **单二进制**，无外部依赖 | 单静态二进制 | Docker/Desktop | Docker Compose 多服务 | Docker/K8s | 开源库 + 托管 |
| 检索内核 | SQLite FTS5（BM25）+ 内存向量 + RRF | **自研** WAL/postings/BM25/HNSW + ES 8 兼容 | 向量库 + BM25（外挂） | 外挂向量库 | 外挂向量库 | 向量 + 图记忆 |
| 查询能力 | 关键词/混合，无 DSL 无聚合 | 50 种 ES 查询类型、聚合、kNN | 基础 | 工作流编排 | 基础聊天检索 | 语义记忆 API |
| 采集格式 | MD/PDF/DOCX/TXT/代码/配置 | 代码(tree-sitter)/日志/CSV/JSONL/PDF/PPTX/XLSX/EPUB/mail/Unity 等 | 多格式 | 多格式 | 多格式 | 对话记忆 |
| 代码符号 | Go 用官方 AST，其余词法 | tree-sitter 36 语言 | 无 | 无 | 无 | 无 |
| MCP | **14 工具**，stdio + Streamable HTTP，自动注册 4 客户端 | 20 工具，26 客户端文档 | 有 MCP | 插件式 | 有 MCP（生态扩展） | 有 MCP |
| 记忆系统 | `/v1/memory` + 批量 + RAG 上下文 | `/_memory` + brain 图 | 会话记忆 | 会话变量 | 会话记忆 | **专注记忆** |
| 权限 | RBAC 三角色（刚完成） | RBAC/多租户（企业版） | SSO/RBAC（企业版） | 企业版 RBAC/SSO | 企业版 | 工作区 |
| 监控 | Prometheus 39 指标 + Grafana 模板 | 13 指标 | 基础 | 基础 | 基础 | 托管侧 |
| 许可 | **AGPL-3.0**（README 曾写 MIT） | Apache-2.0 | MIT | Apache-2.0 + 附加条款 | 自有许可（白标需企业授权） | Apache-2.0 |
| 定价 | 待定（方案见 §5） | 社区免费 / 专业按节点报价 / 企业定制 | Cloud **$50 / $99 每月**；企业私有化定制 | Cloud **$590 / $1,590 每工作区每年**；企业定制 | 基础免费；**白标/企业功能需企业许可** | 托管平台（未核实具体档位） |
| 成熟度 | 3 stars / 2 贡献者 / 停更 5 个月后重启 | **13k stars / 89 贡献者 / 3,541 commits** | YC 公司，融资产品 | 大厂级社区 | 数万企业部署 | 融资产品 |
| 中文能力 | **CJK 分词 + 9 家 embedding（含国内 3 家）** | 英文为主 | 一般 | 一般 | 一般 | 一般 |

> 未核实项：Mem0 具体价格、Khoj 现状（官网 404）。
> 价格来自官网公开页面，签约前需复核。

### 1.3 Cortex 的真实差异化（只有这四条站得住）

1. **中文优先**：CJK 单字分词、中文标题/正文分别分词、9 家 embedding（含 DashScope/智谱/文心）、中文文档与定价。XERJ、Dify、AnythingLLM 都不专门做这件事。
2. **零运维单二进制**：纯 Go、无 CGO、6 平台交叉编译 33–36MB、SQLite 即数据目录。对比 AnythingLLM/Dify 的 Docker 全家桶，安装成本低一个量级。
3. **MCP 工具面 + 自动注册**：14 个工具覆盖搜索/上下文/记忆/地图/计划/代码/项目上下文，`cortex start` 自动写入 Claude Code / Cursor / OpenCode / Trae 配置（含备份与合并保护）。
4. **内置商业化基础设施**：Ed25519 离线 License、配额强制、RBAC 三档、Prometheus。同类开源项目大多没有这些，商业化时要现补。

### 1.4 诚实的短板

| 短板 | 具体表现 | 影响 |
|---|---|---|
| 检索深度不足 | 无重排实现（接口在、实现空）、无查询 DSL、无聚合、无 ES 兼容 | 复杂检索需求会输给 XERJ |
| 采集面窄 | 无 CSV/JSONL/XLSX/PPTX/EPUB/日志/mail | 数据集类用户直接流失 |
| 规模上限 | SQLite 单写者；向量全扫；5k 文档 P99 已 10.5ms | >50 万 chunk 场景不可用 |
| 英文生态弱 | README/文档以中文为主，无英文 landing | 国际市场基本为零 |
| 社区为零 | 3 stars、无 issue 讨论、无外部贡献者 | 获客只能靠内容与渠道 |
| 许可混乱 | LICENSE=AGPL-3.0，README 曾宣称 MIT | 企业法务会直接否掉 |

### 1.5 竞争策略：三个「不做」

- **不与 XERJ 拼引擎**（54 万行 Rust vs 1.7 万行 Go，且它已 Apache-2.0 免费）。
- **不做通用 LLM 工作流平台**（Dify/RAGFlow/Coze 已占）。
- **不做重资产 SaaS**（验证付费意愿前不自建多租户托管）。

**要做的一件事**：把「装好就能用的中文 Agent 记忆/知识库」做到极致——安装 5 分钟、中文搜得准、控制台看得懂、MCP 接得上。

---

## 2. 性能现状与瓶颈

### 2.1 实测数据

环境：Windows / 本地磁盘 / `CGO_ENABLED=0` / 默认 FTS-only（无 embedding）/ 8 索引 worker / 缓存关闭。
复现：`go run ./cmd/cortex-bench all --docs N --workers 8`

| 指标 | 1,000 文档 | 5,000 文档 | 真实库（27,157 文档） |
|---|---|---|---|
| 索引吞吐 | **126.7 files/s** | **106.0 files/s** | — |
| 分块数 | 2,512 | 12,668 | 84,562 |
| 检索 P50 | 2.00 ms | 1.69 ms | 120 ms（含进程启动） |
| 检索 P95 | 3.54 ms | **6.01 ms** | — |
| 检索 P99 | — | **10.51 ms** | — |
| Recall@1/@5/@10 | 1.000 | 1.000 | — |
| 单次搜索（CLI 端到端） | — | — | **0.12 s** |

关键观察：
- **P50 基本不随规模变，P95/P99 明显上升**（1k→5k：P95 3.5→6.0ms）。说明瓶颈在「候选集变大后的排序与结果装配」，不在单条 SQL。
- 索引吞吐随规模下降（126→106 files/s），因为 SQLite 是单写者，且 FTS 触发器与索引维护成本随数据量增长。
- 真实库 27k 文档/84k 分块仍可用，但已接近 SQLite + FTS5 的舒适区上限。

> 说明：之前一次 5k 结果里 recall@1=0.682，经排查是**基准语料自身的 bug**（文档标题带「第 N 篇」序号，数字与相邻文档唯一词撞车），修正语料后恢复 1.000。这段经历本身说明：没有干净的 ground truth，性能数字毫无意义。

### 2.2 瓶颈定位与优化清单

| # | 瓶颈 | 证据 | 优化手段 | 预期收益 | 工作量 |
|---|---|---|---|---|---|
| 1 | **SQLite 单写者** | 索引吞吐 126→106 files/s（1k→5k） | 批量提交（100 文件/事务）、`synchronous=NORMAL`、FTS 延迟重建、读写分离 | 索引吞吐 **2–4×** | M（3–5 天） |
| 2 | **向量全量线性扫描** | `vectorSearchInMemory` 对 23k 向量逐条算余弦（O(n·dim)） | 恢复 HNSW（修稳定性）或 IVF-PQ；多线程 SIMD | 10 万向量 P95 从 ~100ms → **<10ms** | L（1–2 周） |
| 3 | **CJK 单字分词膨胀** | 每个汉字一个 token，索引体积与 postings 双高 | 改用 bigram / 专用 CJK tokenizer；A/B 验证召回 | 索引体积 **−30~50%**，检索 **−20%** | M（3–5 天） |
| 4 | **长尾延迟** | P99 10.5ms @5k，随候选集增长 | 限制候选（FTS top-k 提前截断）、RRF 阶段少分配、批量补文档（已修 N+1） | P99 **−30~50%** | S（1–2 天） |
| 5 | **L2 缓存写竞争** | 每次搜索都写 SQLite 缓存，与索引写抢锁 | L2 只读命中、异步刷新、命中率低时禁用 | 写竞争下降 | S（1 天） |
| 6 | **启动加载向量** | 启动时读 80MB 扁平索引文件 | mmap 或惰性分片加载 | 启动 **−50%** | M（2–3 天） |
| 7 | **JSON 序列化** | gin + encoding/json，大结果集 CPU 高 | 换 bytedance/sonic（依赖已在 go.mod） | RPS **+10~20%** | S（1 天） |
| 8 | **索引 worker 空转** | 多 worker 争抢单写者，锁等待 | 采集并行 + 单写者提交队列 | 吞吐再 **+30%** | M（3 天） |

**已在本轮修复**：检索结果补文档路径的 N+1（原来每条结果查一次 `documents`，topK=20 就是 20 次额外查询）。修完后 FTS 与向量两条腿都在 SQL 里一次带出 `d.path`。

### 2.3 优化路线（分三批）

**第一批（低垂果实，1 周内）**
1. 索引批量提交 + `synchronous=NORMAL`
2. L2 缓存异步化
3. 换 sonic 序列化
4. 长尾截断调参

**第二批（4 周内）**
5. CJK 分词方案 A/B（bigram vs 单字）
6. 索引器读写分离
7. 向量索引选型（HNSW 修复 / IVF-PQ）

**第三批（3 个月）**
8. 百万级 chunk 的分层存储（热数据 SQLite + 冷数据文件）
9. 可选外挂向量库适配器（Qdrant/Milvus 兼容）

### 2.4 性能验证方法

```bash
# 回归对比（同 seed、同参数）
go run ./cmd/cortex-bench all --docs 5000 --workers 8 --json-out benchmarks/results/after.json

# 只看检索延迟
go run ./cmd/cortex-bench search --docs 5000
```

结果存 `benchmarks/results/`，每次发版前跑一次，对比 P95/P99 与 recall，避免「感觉变快了」。

---

## 3. 上升空间

### 3.1 能力缺口（按性价比排序）

| 缺口 | 为什么重要 | 难度 |
|---|---|---|
| 采集格式扩展（CSV/JSONL/XLSX/PPTX/EPUB/日志） | 数据集类用户的第一需求，XERJ 已做到 | S–M |
| 重排（rerank） | 直接提升 top-1 准确率，是「搜得准」的关键 | M |
| ES 兼容读接口（`/_search`、`/_cat/indices`） | 直接接入现有 ES 客户端生态，XERJ 已验证有效 | L |
| 代码符号精度（Python/JS/TS） | 词法级会漏判；Go 已用官方解析器 | M（WASM tree-sitter）|
| 团队协作（共享库/审计/SSO） | Team/Enterprise 的付费理由 | L |
| 托管版 | 降低使用门槛，但运维成本高 | XL |

### 3.2 规模上限路径

```
当前舒适区：≤10 万 chunk（约 3 万文档）
第一档（优化后）：50 万 chunk —— 批量写入 + CJK 分词优化 + ANN
第二档（架构改造）：100 万+ chunk —— 读写分离 + 分层存储
第三档（换引擎）：不再用 SQLite 做检索主存
```

> 现实建议：把产品定位在「个人/小团队 ≤3 万文档」，不要在销售材料里承诺百万级。

### 3.3 生态位与获客杠杆

- GitHub 开源 + README 首屏（英文版缺失，需补）
- MCP 目录：mcp.so / Smithery / Glama / PulseMCP / awesome-mcp-servers
- 编辑器插件：Obsidian、VSCode、Raycast（每个都是现成流量池）
- 中文内容：掘金/知乎/V2EX/B 站，选题围绕「给 Claude Code 装记忆」
- 企业：私有化 AI 知识库关键词 + 集成商渠道 + 云市场上架

### 3.4 商业化能力缺口

| 能力 | 现状 | 需要的动作 |
|---|---|---|
| License 校验 | ✅ Ed25519 离线验签 + feature gating | 补试用、机器绑定、吊销列表 |
| 配额 | ✅ 文档/存储上限 + 402 提示 | 补按席位计数 |
| RBAC | ✅ admin/editor/viewer | 补自定义角色、资源级权限 |
| 审计日志 | ❌ | EE 首个功能 |
| SSO | ❌ | EE 第二个功能 |
| 支付 | ❌ | 接 MoR + 国内渠道 |
| License Server | ❌ | v1.1 可选组件 |

---

## 4. 开源 / 闭源切分

### 4.1 三条铁律

1. **开源版必须能独立完成核心工作流**（装、索引、搜、接 MCP），否则没人愿意试用。
2. **付费卖的是「规模、协作、合规、服务」，不是把核心功能阉割**。
3. **边界用一个明确的接口隔开**（已实现 `pkg/ext`），闭源代码不进公库，公库代码不依赖闭源。

### 4.2 模块归属表

| 模块 | 归属 | 理由 |
|---|---|---|
| `internal/storage`（SQLite/FTS/向量/缓存） | **开源** | 核心引擎，必须可审计 |
| `internal/search`（混合检索/RRF/缓存） | **开源** | 核心能力 |
| `internal/chunker` + `internal/index` | **开源** | 采集与索引 |
| `internal/embedding`（9 家 provider） | **开源** | 生态入口 |
| `internal/rag` + `internal/snippet` | **开源** | 检索质量基础 |
| `internal/symbols`（代码符号） | **开源** | 基础能力（Go AST + 词法） |
| `internal/api`（REST/MCP/控制台） | **开源** | 接入层 |
| `internal/cli` + `cmd/cortex` | **开源** | 用户入口 |
| `internal/auth`（基础 RBAC） | **开源** | 单机/小团队够用 |
| `pkg/license` + `pkg/ext` | **开源** | 授权与扩展接缝（EE 依赖它们） |
| `internal/metrics` | **开源** | 可观测性 |
| `benchmarks/` + `cmd/cortex-bench` | **开源** | 可信度资产（XERJ 的做法证明有效） |
| **审计日志**（操作留痕/导出） | **EE** | 合规需求，付费理由明确 |
| **SSO**（OIDC/SAML/SCIM） | **EE** | 企业采购硬门槛 |
| **团队/多租户**（共享库、席位管理） | **EE** | Team 档核心 |
| **高级 RBAC**（自定义角色、资源级） | **EE** | 企业版差异化 |
| **加密 at-rest / BYOK** | **EE** | 金融/医疗刚需 |
| **S3/GCS 备份与快照** | **EE** | 运维能力 |
| **HA/集群** | **EE** | 大客户 |
| **ES 兼容读接口** | **EE**（可后续下沉开源） | 迁移桥，先做付费卖点 |
| **企业级 rerank 服务集成** | **EE** | 增值能力 |
| **License Server** | **私有** | 商业化基础设施 |
| **定价页/支付流程** | **私有** | 商业信息 |

### 4.3 仓库结构与扩展接缝（已落地）

**实际存放位置（本机）**

| 版本 | 路径 | 许可 | 是否公开 |
|---|---|---|---|
| 开源免费版 | `E:\程序\github\my-projects\Cortex` | Apache-2.0 | ✅ GitHub 公开 |
| 企业付费版 | `E:\程序\github\my-projects\Cortex-ee` | 商业许可（专有） | ❌ 私有仓库 |

> **关键约束**：Go 的 `internal/` 包不能被外部模块导入。所以扩展接缝必须放在公开包：
> - `github.com/lh123aa/cortex/pkg/ext` —— 扩展注册表 + `Extension` 接口
> - `github.com/lh123aa/cortex/pkg/license` —— 许可解析与 feature 判定
> - `github.com/lh123aa/cortex/pkg/app` —— 公开启动入口 `app.Run()`
>
> 核心内部仍使用 `internal/...`，企业版只依赖这三个 `pkg/` 包。

企业版实现扩展（示例为已落地的审计日志）：

```go
// Cortex-ee/ee/audit/audit.go
type Extension struct{ store *Store }

func (e *Extension) Name() string               { return "audit-log" }
func (e *Extension) RequiredFeatures() []string { return []string{"audit-log"} }
func (e *Extension) RegisterMCP(s *mcp.Server) error { /* 注册 cortex_audit_query */ return nil }
func (e *Extension) RegisterHTTP(r *gin.Engine) error {
    r.GET("/ee/audit", ...)   // 企业版路由
    r.Use(auditMiddleware)    // 全局旁路审计（必须早于业务路由注册）
    return nil
}

func init() { ext.Register(NewExtension(...)) }
```

企业版入口只有两行：

```go
import (
    _ "github.com/lh123aa/cortex-ee/ee/audit"
    "github.com/lh123aa/cortex/pkg/app"
)

func main() { app.Run() }
```

**私有仓库实际结构**

```
Cortex-ee/                      ← E:\程序\github\my-projects\Cortex-ee
├── go.mod                      module github.com/lh123aa/cortex-ee
│                               require github.com/lh123aa/cortex + replace ../Cortex
├── ee/audit/                   ✅ 审计日志（feature: audit-log，已实现）
├── cmd/cortex-ee/main.go       import 扩展 + app.Run()
├── bin/cortex-ee.exe           构建产物（gitignored）
├── README.md                   构建、门控、签发流程
├── LICENSE                     专有许可声明
└── .gitignore                  排除 bin/、licenses/、*.key、*.jsonl
```

规划中的扩展目录：`ee/sso/`、`ee/team/`、`ee/rbacx/`、`ee/crypto/`、`ee/backup/`、`ee/escompat/`。

**接缝契约（重要）**

`pkg/ext.RegisterHTTP` 在核心注册任何业务路由**之前**被调用。这样企业版可以：
- 用 `r.Use(...)` 安装全局中间件（审计、限流、脱敏）；
- 用 `r.GET/POST(...)` 注册自己的路由（如 `/ee/audit`）。

若放在路由注册之后调用，`r.Use` 会静默失效（gin 只对之后注册的路由生效）。

构建：

```bash
# 公开版
cd cortex && go build -o cortex ./cmd/cortex
# 企业版（在私有仓库）
cd cortex-ee && go build -o cortex-ee ./cmd/cortex-ee
```

> 关键设计：**同一份核心代码，两种二进制**。企业版不 fork 核心，升级核心时企业版只需重新编译。

### 4.4 许可策略决策（已执行：方案 A）

> **2026-10-30 更新：已按方案 A 完成迁移。**
> - `LICENSE` 已替换为 **Apache License 2.0**
> - 新增 `NOTICE`（第三方依赖与商标声明）
> - README / README.en / 落地页 / 宣传稿 / 商业计划中的 33 处许可声明已统一为 Apache-2.0
> - 迁移记录：见 [LICENSE_MIGRATION.md](./LICENSE_MIGRATION.md)
> - 企业版仍为商业许可、不入本仓库

下表保留为决策依据（历史记录）：

| 方案 | 优点 | 缺点 | 适用 |
|---|---|---|---|
| **A. 核心 Apache-2.0 + EE 专有（已采用）** | 企业友好、有利于采用；EE 闭源法律上干净（通过接口调用，非链接） | 无法阻止大厂白嫖核心 | 想要增长与商业化的现实选择 |
| B. 核心 AGPL-3.0 + 商业双许可 | 对「拿去做闭源 SaaS」有强约束；你拥有版权即可双许可 | 国内企业普遍排斥 AGPL，采用率显著降低 | 你拥有全部版权（需 CLA/转让）且目标是大客户 |
| C. 核心 MIT | 采用最快 | 无专利条款、保护最弱 | 不推荐（商业版价值支撑不足） |

**执行前已完成**：核对历史贡献者（`git shortlog` 仅两个提交身份，其中 `Cortex <cortex@example.com>` 为本地占位身份），并由仓库所有者确认同意迁移。

### 4.5 公开仓库上线前检查清单（迁移后状态）

- [x] `LICENSE` 与 README 一致（**已统一为 Apache-2.0**）
- [ ] 移除 `docs/private/`（当前为空目录，但仍应从版本库删除）
- [x] 确认 `_temp/` 全部被忽略（含开发私钥 `license-private.key`、测试 license）
- [x] `.env.example` 中无真实 key（当前仅有占位）
- [x] 已扫描跟踪文件中的密钥模式（`sk-*`、`AKIA*`、`BEGIN PRIVATE KEY`）—— 无命中
- [ ] `git log` 历史全文检索一次密钥模式（仓库历史尚未扫描）
- [ ] 移除或重写 `COMMERCIAL_PLAN.md` 中与最终定价冲突的数字
- [x] `coverage` 等构建产物不入库（当前根目录有未跟踪文件，未进版本库，建议清理并确认 `.gitignore` 覆盖）
- [ ] 补英文 README（国际采用的前提）
- [ ] 确认 `scripts/dev/` 之类开发用私钥目录不存在

---

## 5. 付费管理方案

### 5.1 分层与权益

| 能力 | Free | Pro | Team | Enterprise |
|---|---|---|---|---|
| 单机部署、CLI、REST、MCP 全部工具 | ✅ | ✅ | ✅ | ✅ |
| 文档上限 | 5,000 | 500,000 | 1,000,000 | 不限 |
| 存储上限 | 1 GiB | 100 GiB | 500 GiB | 不限 |
| Web 控制台 | 只读状态 | ✅ | ✅ | ✅ |
| 自动索引守护 | ❌ | ✅ | ✅ | ✅ |
| 重排（rerank） | ❌ | ✅ | ✅ | ✅ |
| 多用户 / RBAC | ❌（单用户） | ❌ | ✅ 3 席起 | ✅ 不限 |
| 审计日志 | ❌ | ❌ | ✅ 基础 | ✅ 完整 + 导出 |
| SSO（OIDC/SAML/SCIM） | ❌ | ❌ | OIDC | ✅ |
| 加密 at-rest / BYOK | ❌ | ❌ | ❌ | ✅ |
| S3/GCS 备份、快照 | ❌ | ❌ | ✅ | ✅ + 跨区域 |
| HA / 集群 | ❌ | ❌ | ❌ | ✅ |
| ES 兼容读接口 | ❌ | ❌ | ❌ | ✅ |
| 支持 | 社区 | 邮件 48h | 邮件 24h | SLA 4h + 培训 |
| 许可 | AGPL / Apache | 商业许可 | 商业许可 | 商业许可 |

> 免费版原则：**功能不阉割、无时间限制**，只限规模；付费买「规模、协作、合规、服务」。

### 5.2 定价

竞品锚点：
- AnythingLLM Cloud：**$50 / $99 每月**
- Dify Cloud：**$590 / $1,590 每工作区每年**
- XERJ：社区免费 / 专业按节点报价 / 企业定制
- Open WebUI：基础免费，白标与企业功能需企业许可

Cortex 定价建议：

| 档位 | 国内 | 国际 | 说明 |
|---|---|---|---|
| Free | ¥0 | $0 | ≤5,000 文档 |
| Pro | **¥199/年**（¥25/月） | **$29/年** | 个人/小团队，低于 AnythingLLM 一个数量级 |
| Team | **¥899/年**（3 席，+¥199/席） | **$129/年** | 小团队协作 |
| Enterprise | **¥3 万/年起** | **$4,500/年起** | 私有化 + 合规 + SLA |

促销与政策：
- 早鸟：前 100 名 Pro **¥99 / 2 年**（明确非永久）
- 教育/开源项目 5 折
- 14 天无理由退款
- 试用：14 天全功能（见 §5.3）

### 5.3 授权技术方案

**License 格式（已实现，Ed25519）**

```
license = base64url(payload) + "." + base64url(signature)

payload = {
  v: 1,
  plan: "pro" | "trial" | "team" | "enterprise",
  features: ["console", "audit-log"],   // 可选，叠加在套餐默认 feature 上
  email: "buyer@example.com",
  order: "ord_xxx",
  seats: 1,
  iat: 1790000000,
  exp: 1820000000                          // 0 = 永久
}
```

校验：客户端内置公钥离线验签；`exp` 到期即失效；`features` 结合 `PlanFeatures(plan)` 决定 EE 扩展是否注册（已实现于 `pkg/license` + `pkg/ext`）。

**Feature 映射**

| feature | 含义 | 默认包含 |
|---|---|---|
| `console` | Web 控制台完整功能 | Pro / Team / Enterprise / Trial |
| `rerank` | 重排 | Pro+ |
| `auto-index` | 自动索引守护 | Pro+ |
| `rbac-team` | 多用户与角色 | Team+ |
| `audit-log` | 审计日志 | Team+ |
| `sso-oidc` | SSO | Team（OIDC）/ Enterprise（SAML/SCIM） |
| `backup-s3` | S3 备份 | Team+ |
| `*` | 全部 | Enterprise |

**试用**

- 14 天，`plan: "trial"`，按 Pro 配额。
- 签发方式二选一：
  - 在线：License Server 收到邮箱后签发（推荐，可防重复试用）。
  - 离线：`cortex license trial` 本地生成（绑定机器码 + 首次运行时间）。**接受可被重置**，因为免费档本来就不设防。
- 试用到期后自动降级 Free（文档仍可读，只读不写）。

**席位（Seats）**

- Team 版按 `seats` 限制本地用户数（`cortex user list` 计数）。
- 离线容忍：允许超出 ≤10% 的短暂越界（便于离职/换人），超限时提示但不禁用。
- 联网时由 License Server 校准（可选）。

**机器绑定（可选）**

- payload 可含 `machine` 指纹（主机名 + CPU + 磁盘序列号的哈希）。
- 绑定后换机需重新激活（或联系客服解绑），默认**不强制**，避免伤害个人用户体验。

**吊销**

- License Server 维护吊销列表；客户端每次联网校验时拉取，缓存 30 天。
- 离线环境无法吊销 → 接受，靠授权协议与法律约束。

### 5.4 License Server（v1.1，可选组件）

最小可用设计（Cloudflare Worker + D1/KV，成本 ≈ $5/月）：

| 接口 | 用途 |
|---|---|
| `POST /v1/trial` | 邮箱 + 机器码 → 签发 14 天 trial license |
| `POST /v1/activate` | license → 绑定机器，返回激活记录 |
| `POST /v1/refresh` | 续期检查 + 吊销列表 |
| `POST /v1/deactivate` | 解绑（换机） |
| `GET /v1/revocations` | 吊销列表（签名） |

原则：
- 响应全部用同一把 Ed25519 私钥签名，客户端离线也能验。
- 离线宽限 30 天：断网不影响已激活的付费用户。
- 只存最小数据（邮箱哈希、机器码哈希、订单号），不存文档内容。

### 5.5 支付与开票

| 市场 | 渠道 | 说明 |
|---|---|---|
| 国际 | **Paddle / Lemon Squeezy / Polar**（Merchant of Record） | 自动处理 VAT/销售税、支持 License Key 交付；费率约 4–6% + 固定费 |
| 国内（启动期） | **面包多 / 爱发电** | 个人即可收款，手动发 key，验证付费意愿 |
| 国内（正式） | **支付宝当面付 / 微信支付** | 需个体户或公司主体；可开发票 |
| 企业 | 对公转账 + 合同 + 发票 | Enterprise 必须 |

运营流程：

```
浏览定价页 → 支付（MoR/国内） → 自动或人工发 license
   → cortex license activate <key>
   → 到期前 7 天邮件提醒 → 续费（发新 key 或延长 exp）
   → 14 天内可退款（作废 key + 退款）
```

退款政策：14 天无理由；已激活且使用超 30 天的按比例退款需人工审核。

### 5.6 反盗版与风险

| 风险 | 应对 |
|---|---|
| License 被 patch | 接受弱 DRM；价值在持续更新、支持、企业合规 |
| 试用被重置 | 在线签发 + 邮箱/机器码；离线接受滥用 |
| 团队席位超用 | 超 10% 提示、超 50% 拒绝写入 |
| 代理/转售 | 合同限制 + 订单绑定邮箱 |
| 支付合规 | MoR 处理国际税务；国内注册主体后规范开票 |

### 5.7 商业化指标

| 指标 | 目标（12 个月） |
|---|---|
| 试用→付费转化 | ≥ 5% |
| Pro 付费数 | 60–120 |
| Team 客户 | 5–15 |
| Enterprise 客户 | 1–3 |
| 月度 churn | < 5% |
| 退款率 | < 3% |
| ARR | ¥10–40 万 |

---

## 6. 90 天执行计划

| 周 | 动作 | 产出 |
|---|---|---|
| W1–2 | 解决许可（Apache-2.0 或双许可）+ 公库清理（§4.5 清单） | 可合法公开发布的仓库 |
| W2–3 | 发 v4.0.0 开源版（含 bench 报告、README 英文版） | GitHub Release |
| W3–4 | 建私有仓库 `cortex-ee` + 移植 1 个 EE 功能（审计日志） | 企业版可编译 |
| W5–6 | License Server MVP + 试用签发 | 可自助试用 |
| W6–8 | 定价页 + 支付接入（MoR + 面包多） | 可收款 |
| W9–10 | 性能第一批优化（批量写入/L2 缓存/sonic） | 索引吞吐 2× |
| W11–12 | 中文内容 5 篇 + MCP 目录提交 + 种子用户 20 人 | 首批付费线索 |

---

## 7. 本轮已落地的代码改动

| 改动 | 文件 | 说明 |
|---|---|---|
| 扩展接缝 | `pkg/ext/` | 开源核心与 EE 的唯一边界；按 license feature 门控；失败隔离；7 个测试 |
| 许可 feature 模型 | `pkg/license/license.go` | `PlanTrial`、`PlanFeatures`、`HasFeature`、`DaysLeft`、试用按 Pro 配额 |
| 许可注入 | `internal/cli/license_init.go` | 启动时把许可注入扩展注册表 |
| 接缝接线 | `internal/api/mcp.go`、`internal/api/rest.go` | MCP 工具与 HTTP 路由均支持 EE 扩展 |
| N+1 修复 | `internal/storage/search.go`、`internal/search/engine.go` | 检索结果一次带出 `d.path`，去掉每条结果一次文档查询 |
| 基准修正 | `internal/bench/corpus.go` | 唯一词改纯字母编码，标题去序号，消除语料自污染 |

验收：`go vet` 0 警告；22 包测试全绿（含新增 ext / license feature 测试）；6 平台交叉编译通过。

---

## 8. 一句话总结

**Cortex 的价值不在「又一个搜索引擎」，而在「中文用户装 5 分钟就能给 Agent 装上记忆」这件事的完成度。** 开源核心负责增长与信任，私有 EE 负责协作与合规变现；先修性能与许可，再谈规模。
