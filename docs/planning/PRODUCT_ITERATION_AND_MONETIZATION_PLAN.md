# Cortex 产品迭代与变现方案

> 版本：v1.0 ｜ 日期：2026-10-22 ｜ 适用对象：Cortex 作者/维护者
> 依据：仓库代码实测（`go test ./...` 16 包全绿）、Git 历史（116 commits / 2 contributors / 2026-05-26 后停更）、GitHub 数据（3 stars）、竞品公开定价（Dify、XERJ，见文末来源）
> 结论先行：Cortex 现在**能跑、不能用、不可卖**。先用 4 周修信任 + 补控制台 + 打通授权与发布，再用 8 周补检索质量与分发，然后以 **Open Core + 订阅 License + 企业私有化授权** 变现。12 个月现实目标 ARR **¥10–40 万**，不是旧计划里的 $120K+。

---

## 0. 执行摘要

| 问题 | 现状 | 方案 |
|---|---|---|
| 能不能卖 | ❌ 不能：License 只是本地 SQLite 表、可自行生成；`usage` 永远显示 Free/1GB；Free 限额没有任何强制点；无支付、无 ToS、无隐私政策 | 4 周内补齐「可授权、可计价、可交付、可追责」四件套 |
| 有没有产品 | ⚠️ 半成品：CLI/REST/MCP 都通，但 Web 控制台空白、版本三套并存、检索质量未量化 | 先修控制台与版本，再补检索质量 + 基准 |
| 有没有用户 | ❌ 3 stars / 停更 5 个月 | 重新发布 v4.0.0，用中文内容 + MCP 生态 + Obsidian/VSCode 插件做冷启动 |
| 靠什么赚钱 | 旧计划：企业版 $199/月 | 新方案：Pro ¥199/年（个人/小团队）+ Team ¥899/年（3 席起）+ Enterprise ¥3 万/年起（私有化/合规/SLA） |
| 第一年目标 | 旧计划 $120K ARR / 50 企业客户 | 现实目标：Pro 60–120 个、Team 5–15 个、Enterprise 1–3 个 → ARR ¥10–40 万 |
| 最大风险 | LICENSE 是 AGPL-3.0 而 README 写 MIT；AGPL 会劝退企业客户 | 立刻二选一：推荐核心 Apache-2.0 + 商业版闭源（见 §4.2） |

**三个不做**：
1. 不做通用 RAG/工作流平台（Dify/RAGFlow 已占位）；
2. 不与 XERJ 正面拼搜索引擎内核（54 万行 Rust vs 1.3 万行 Go，没有胜算）；
3. 不在验证付费前做重资产 SaaS（先卖 License，后卖托管）。

---

## 1. 现状诊断

### 1.1 产品能力盘点

| 能力 | 状态 | 证据 |
|---|---|---|
| CLI（index/search/start/serve/mcp/watch/dedup/usage…） | ✅ 可用 | `internal/cli/`，16 个子命令 |
| REST API（15+ 端点） | ✅ 可用 | `/v1/search`、`/v1/context`、`/v1/memory/*`、`/health`、`/metrics` |
| MCP（8 工具，stdio + Streamable HTTP） | ✅ 可用 | `internal/api/mcp.go:132-176` |
| 混合检索（FTS5 BM25 + 内存向量 + RRF） | ⚠️ 能用但未量化 | `internal/search/engine.go`；向量默认线性扫描（`internal/storage/search.go:98`） |
| 嵌入 Provider（9 家） | ✅ 可用 | `internal/config/config.go:33-44` |
| 文件格式（MD/PDF/DOCX/代码/配置） | ✅ 基础可用 | `internal/chunker/`（markdown/pdf/docx/go/text/chinese） |
| Web 控制台 | ❌ 空白 | `internal/api/admin.go:14` 模板名 bug，响应 0 字节 |
| 监控（Prometheus 39 指标） | ✅ 可用 | `/metrics:9090` |
| 认证（bcrypt/token/API Key/限流） | ✅ 可用 | `internal/auth/service.go` |
| 测试 | ✅ 16 包全绿 | 2026-10-22 实跑 `go test ./...` |

### 1.2 变现就绪度（关键结论：现在收不了钱）

| 检查项 | 结论 | 证据 |
|---|---|---|
| License 生成/校验 | ❌ 只有本地表 + 随机 key；`CreateLicense` 是普通方法，任何调用方都能发证；没有 CLI 命令、没有验签、没有服务端校验 | `internal/storage/license.go:1-110`；`grep -r "license" internal/cli cmd` 无结果 |
| 套餐/配额 | ❌ `cortex usage` 永远输出 `Tier: free`、1GB 上限；没有任何地方强制 | `internal/cli/init.go:661-690`；`internal/config/config.go` 无 plan/quota 字段 |
| 用量计量 | ❌ 只有 `CalculateStorageUsed`，无按文档数/请求数计量 | `internal/cli/init.go:674` |
| 多用户/团队/RBAC | ❌ 只有 users/api_keys/auth_tokens；无角色、无租户隔离（`user_id` 仅为数据标记） | `internal/storage/schema.sql:8-54` |
| 审计日志/SSO | ❌ 不存在 | — |
| 发布流水线 | ⚠️ tag 触发三平台构建，但**未注入版本 ldflags**、无校验和、无自动 release notes、无自动更新 | `.github/workflows/build.yml`；`Makefile` 有 LDFLAGS 但 CI 未用 |
| 安装分发 | ⚠️ 有 `scripts/install.sh/.ps1`，无 Homebrew/Scoop/Docker Hub 自动化 | `scripts/` |
| 遥测/崩溃上报 | ❌ 没有 | `grep` 无结果 |
| 使用条款/隐私政策 | ❌ 没有 | 仓库内无 TERMS/PRIVACY |
| 许可证一致性 | ❌ **LICENSE 文件是 AGPL-3.0，README 写 MIT** | `LICENSE:1`；`README.md:694-696` |
| 版本一致性 | ❌ tag v1.0.4 / README v3.5.3 / 二进制 `dev` / `/admin` v2.3 四套并存 | `git tag`；`internal/api/admin.go:42` |

### 1.3 外部竞争坐标（2026-10 可查实价）

| 产品 | 形态 | 收费模型 | 价格锚点 |
|---|---|---|---|
| **XERJ** | Rust 自研检索引擎，ES 兼容，MCP 20 工具，54 万行，13k stars | 三档：Community 免费 / Professional 按节点报价 / Enterprise 定制；同一二进制，卖 HA、加密、RBAC、SLA、合规 | 免费社区版全功能单节点；Pro/Ent 需询价（来源：xerj.org/pricing） |
| **Dify** | 开源 LLM 应用/工作流平台 | Cloud 订阅 + 企业私有化 | Cloud Professional **$590/工作区/年**（3 人/500 文档/5GB），Team **$1,590/年**（来源：dify.ai/pricing） |
| FastGPT / MaxKB / RAGFlow | 国内开源知识库/RAG | 开源 + 商业版/企业版 | 商业版通常按年/按实例报价（未逐一核实） |
| Mem0 | 托管记忆层 + 开源 | 托管平台订阅 | 企业治理/审计默认内置（具体价格未核实） |
| 语雀/飞书知识库 | SaaS 知识库 | 按人/年订阅 | 国内 ¥99–299/人/年常见档位（未核实） |
| **Cortex** | 本地单二进制 + MCP + 记忆 | 无 | 3 stars，未上线任何付费 |

**判断**：Cortex 在两个市场之间：向上打不过 XERJ（引擎深度），横向打不过 Dify（平台广度）。唯一能赢的位置是 **「中文优先、零运维、私有化、Agent 记忆」的个人与小团队市场**——大厂看不上，XERJ 不做中文产品化，Dify 太重。

---

## 2. 战略

### 2.1 一句话定位

> **Cortex：给中文 AI 编程用户和小团队的「私有化第二大脑」——一条命令装好，Claude Code / Cursor 立刻拥有跨项目、跨会话的长期记忆。**

三个支点：**中文 + 零运维单二进制 + MCP 原生记忆**。不卖「搜索引擎」，卖「装好就能用的记忆」。

### 2.2 目标用户（ICP）

| 画像 | 规模 | 痛点 | 付费理由 | 对应版本 |
|---|---|---|---|---|
| A. 独立开发者 / 1–5 人小团队 | 大 | Claude Code/Cursor 每次新会话都失忆；代码库太大 grep 读不完；笔记/文档散落 | 省时间、跨项目检索、自动索引 | Pro ¥199/年 |
| B. 隐私敏感小团队（法务/医疗/财税/研发 5–50 人） | 中 | 不能把文档传给 SaaS；需要权限与审计 | 私有化 + 审计 + 团队共享 | Team ¥899/年 / Enterprise |
| C. 内容/教育/知识付费作者 | 中 | 大量 Markdown/PDF 需要可问答 | 一键建库 + 控制台管理 | Pro / 轻托管 |

### 2.3 竞争策略

- **借生态打**：MCP 目录（mcp.so / Smithery / Glama / PulseMCP / awesome-mcp-servers）、Obsidian 插件商店、VSCode Marketplace、Homebrew/Scoop。这些渠道的获客成本≈0。
- **用中文内容打**：掘金/知乎/V2EX/少数派/B站，选题围绕「给 Claude Code 装记忆」「本地知识库选型」，不写泛泛的 AI 新闻。
- **用诚实对比打**：公开承认引擎不如 XERJ、平台不如 Dify，但强调「中文 + 安装 5 分钟 + 私有化 + 记忆」，反而建立信任。
- **不打价格战**：Pro 定价刻意低于 Dify（$49/月起）一个数量级，用低价换装机量，用 Team/Enterprise 收 ACV。

---

## 3. 产品迭代路线

### 3.0 质量目标（先立标尺，再谈功能）

| 指标 | 现状 | 6 个月目标 |
|---|---|---|
| 首次可用时间（下载→索引→MCP 返回结果） | 未测 | **< 5 分钟** |
| 索引吞吐 | README 声称 ~460 files/s（未复现） | 公开基准：**≥ 1,000 文件/分钟**（1 万文件 < 10 分钟） |
| 检索 P95 | README 声称 < 200ms @ 20k chunks | **< 200ms @ 5 万 chunk**，< 500ms @ 10 万 |
| 检索质量 recall@10 | 无评测 | 自建 300 条中文查询集，**≥ 0.85** |
| 索引失败率 | 无统计 | **< 1%**（二进制/多媒体自动跳过并计数） |
| 崩溃/数据损坏 | 无统计 | 30 天内 0 个 P0；WAL + 备份演练通过 |
| 安装成功率 | 无统计 | Win/mac/Linux 各 ≥ 95% |

**动作**：把上面每一项做成 `scripts/bench/` 可复现脚本 + CI job + 结果写进 README。没有这张表，所有「快」「准」都是自说自话，也没法和 XERJ 的公开 scorecard 对话。

### 3.1 P0：信任修复 + 可卖性地基（0–4 周，约 20–28 人天）

| # | 任务 | 验收标准 | 估时 |
|---|---|---|---|
| P0-1 | 修 `/admin`（`template.ParseFS` 直接返回 `admin.html`）+ 控制台 v1 | 打开有内容；状态/搜索/文档/记忆/索引任务/设置/API Key 六个面板可用；数据来自真实 `/v1/*` | 5–8d |
| P0-2 | 版本与许可证统一 | 一个 `internal/version` 常量：tag/README/二进制/`/admin`/`--version` 全部一致；LICENSE 决策落地（见 §4.2） | 1–2d |
| P0-3 | 发布流水线 | tag 触发：6 平台（linux/darwin/windows × amd64/arm64）+ `-ldflags` 注入版本 + SHA256 + release notes + Docker 镜像 + 安装脚本一键可用 | 3–4d |
| P0-4 | License v1（Ed25519） | `cortex license activate <key>` / `cortex license status`；离线验签；payload 含 tier/features/exp/seats；伪造 key 验签失败；文档说明 | 4–6d |
| P0-5 | 配额与用量 | `storage_used` 实时可查；Free 超限返回 402 + 升级链接；`cortex usage` 显示真实 tier；控制台展示进度条 | 2–3d |
| P0-6 | 文档与 Agent 友好 | 5 分钟 Quickstart（Win/mac/Linux）、对比页、FAQ、`llms.txt`、`AGENTS.md`、MCP 工具 schema 自动生成 + 漂移门禁 | 2–3d |
| P0-7 | 基准 harness | `scripts/bench/` 跑通：索引吞吐、检索 P50/P95、recall@10；CI 每次发布跑一次并归档 | 3–5d |

**P0 退出标准**：一个陌生人从官网下载 → 5 分钟内让 Claude Code 搜到自己的笔记；`cortex license status` 显示 Pro；管理面板不再空白；GitHub Release 有 6 个平台包 + 校验和。

### 3.2 P1：能力提升，让用户留下来（4–10 周，约 40–55 人天）

| # | 任务 | 为什么 | 估时 |
|---|---|---|---|
| P1-1 | 检索质量包：中文同义词/分词优化、查询改写（可选 LLM）、snippet 高亮、混合权重可配、可选 cross-encoder rerank（本地 ONNX 或 API） | 直接决定「搜得准不准」，是留存第一因素 | 8–12d |
| P1-2 | 代码符号索引：tree-sitter 支持 Go/Python/JS/TS（4 语言起步），输出 file:line + 签名 | 对标 XERJ 的核心体验，代码场景必需 | 8–10d |
| P1-3 | 格式扩展：CSV/JSONL/XLSX/PPTX/EPUB/日志 | 覆盖「文档/资料」场景，扩大使用面 | 8–12d |
| P1-4 | MCP 工具 8 → 14：`cortex_map`、`cortex_code_search`、`cortex_plan`、`cortex_memory_batch`、`cortex_project_context`、`cortex_stats` | Agent 侧能力面 = 产品价值 | 4–6d |
| P1-5 | 记忆增强：项目命名空间、时间衰减、去重合并、导出/导入（聊天记录） | 强化「第二大脑」差异化 | 5–8d |
| P1-6 | 控制台 v2：多库/多项目、索引进度、备份管理、审计日志雏形 | 从「能用」到「好管」 | 6–10d |
| P1-7 | 规模：HNSW 修复或 IVF-PQ；增量索引；100 万 chunk 压测报告 | 决定能卖多大的客户 | 8–15d |
| P1-8 | 集成分发：Obsidian 插件、VSCode 扩展（二选一起步）、`cortex init` 一键接入 4 家客户端 | 冷启动获客杠杆 | 6–10d |

**P1 退出标准**：有可公开的 benchmark 报告；代码搜索能达到「给一个函数名，返回 file:line + 签名」；控制台可以完成日常管理；至少一个第三方集成能带来新增用户。

### 3.3 P2：商业化功能（10–24 周，按客户付费意愿排期）

| 模块 | 内容 | 触发条件 | 估时 |
|---|---|---|---|
| 团队/RBAC | admin/editor/viewer、成员管理、共享库 | ≥ 5 个 Team 付费意向 | 8–12d |
| 审计日志 | 操作审计、导出 | 首个 Enterprise 客户要求 | 4–6d |
| SSO | OIDC（先），SAML/SCIM（后） | 首个 Enterprise 客户要求 | 6–10d |
| 加密与合规 | at-rest 加密、S3 备份、Helm chart、air-gapped 安装包 | Enterprise 合同 | 8–15d |
| 托管版 beta | 一键开实例、按库订阅 | Pro ≥ 100 付费且支持工单可控 | 15–25d |

> 原则：**付费功能按合同排期**，不预先自嗨开发。P2 里只有 RBAC 值得提前做（Team 版必需）。

### 3.4 与旧计划的关系

旧计划（`docs/planning/ITERATION_PLAN_v3.0.md`、`COMMERCIAL_PLAN.md`）的问题是：
1. 假设企业客户愿意为 $199/月买单——但项目 3 stars、停更 5 个月，没有信任基础；
2. 把「License Key + 微信支付 + Stripe」列为 5 天 Sprint——实际支付/税务/退款/发票远不止 5 天；
3. 版本与许可不一致（MIT vs AGPL）会让客户法务直接否掉。
**建议：保留其功能分层思路，废弃其收入预测与定价，以本方案为准。**

---

## 4. 变现方案

### 4.1 商业模式：Open Core + 订阅 + 企业授权 + 服务

```
GitHub 开源核心（免费获客）
      │
      ├── Cortex Pro ¥199/年 ──────── 个人/小团队，自助购买
      │
      ├── Cortex Team ¥899/年(3席) ── 小团队，共享库+RBAC+审计
      │
      ├── Cortex Enterprise ¥3万/年起 ─ 私有化+合规+SLA+培训
      │
      └── 服务：迁移/定制/培训（按人天 ¥2,000–3,000）
```

**为什么不做纯 SaaS（至少第一年）**：Cortex 的差异化是「数据不出本地」，SaaS 与服务人群矛盾；托管版是给「不想运维但信得过你」的客户，等 Pro 留存验证后再做。

### 4.2 许可策略（必须本周决定）

| 方案 | 优点 | 缺点 | 适用 |
|---|---|---|---|
| **A. 核心 Apache-2.0 + Pro/EE 闭源私有仓（推荐）** | 企业友好、专利授权、生态易传播；Open Core 是主流变现路径 | 无法阻止大厂白嫖；靠商标+服务+持续开发建立壁垒 | 当前阶段最合适 |
| B. 核心 MIT（按 README 现状） | 传播最快 | 同上，且无专利条款；与现有 LICENSE 冲突最小改法 | 次选 |
| C. 保持 AGPL-3.0 + 双许可 | 对「改代码做 SaaS」的公司有强约束；卖商业授权是正统路径 | 国内企业普遍排斥 AGPL；个人开发者也会绕道；需要律师与 CLA；显著降低采用率 | 只想做企业生意时 |
| D. BSL（4 年后转 Apache） | 防云厂商托管抄袭 | 不是 OSI 开源，社区观感差，MCP 生态不友好 | 不建议 |

**推荐 A**：仓库根 `LICENSE` 换 Apache-2.0（保留历史贡献者署名），Pro/Enterprise 代码放私有仓编译；商标 `Cortex` 注册/声明。**若你更看重「防止别人拿去做闭源 SaaS」**，选 C，但必须接受获客变难，并在定价页明确「企业客户可购买商业许可豁免 AGPL」。
无论选哪个：**README、LICENSE、定价页、二进制 `--version` 必须同一套说法**。本文不构成法律意见，最终请律师过一遍。

### 4.3 版本分层与定价

| 能力 | Free | Pro ¥199/年 | Team ¥899/年 | Enterprise ¥3万/年起 |
|---|---|---|---|---|
| 本地单二进制、CLI、REST、MCP 全部工具 | ✅ | ✅ | ✅ | ✅ |
| 文档量上限 | 5,000 | 500,000 | 1,000,000 | 不限 |
| 混合检索（FTS+向量+RRF） | ✅ | ✅ | ✅ | ✅ |
| Web 控制台 | 只读状态页 | ✅ 完整 | ✅ 完整 | ✅ 完整 |
| 自动索引守护 / 增量 / 备份 | 手动 | ✅ | ✅ | ✅ |
| rerank / 查询改写 | — | ✅ | ✅ | ✅ |
| 多库/项目 | 1 | 10 | 50 | 不限 |
| 团队共享 / RBAC | — | — | ✅ | ✅ |
| 审计日志 | — | — | 基础 | 完整+导出 |
| SSO (OIDC/SAML) | — | — | OIDC | ✅ |
| 加密 at-rest / air-gapped | — | — | — | ✅ |
| 支持 | 社区 | 邮件 48h | 邮件 24h | SLA 4h + 培训 |
| 商业使用许可 | 个人/非商业 | ✅ 商业 | ✅ 商业 | ✅ |

> 免费版原则：**功能不阉割、无时间限制**，只限规模。付费买的是「规模、协作、合规、服务、省时间」。
> 早鸟：前 100 名 Pro **¥99 / 2 年**（明确非永久，避免未来收入被锁死）；教育/开源项目 5 折。

**定价锚点依据**：
- Dify Cloud Professional $590/工作区/年（≈¥4,200，3 人/500 文档）→ Cortex Pro ¥199/年 约 1/20 价格，属「闭眼买」区间；
- XERJ Pro 按节点询价、Enterprise 定制 → Cortex Enterprise ¥3 万/年是国内小团队私有化可接受的低位；
- 国内工具类订阅常见 ¥99–499/年 → Pro 落在中位，Team 落在团队预算内。

### 4.4 授权与支付架构

**License 设计（v1，无需服务端即可工作）**
```
payload = { v:1, plan:"pro", features:["console","rerank"], seats:1,
            email:"buyer@x", issued:"2026-10-22", exp:"2027-10-22", order:"ord_xxx" }
license = base64(payload) + "." + base64(ed25519_sign(payload))
```
- 客户端内置 **公钥** 验签；私钥只在你手里（离线签发）。
- `cortex license activate <key>` 写入 `~/.cortex/license`；`cortex license status` 显示套餐与到期。
- 可选联网校验（`cortex license check`，失败不阻断，仅提示），服务端只做「吊销列表 + 续费」。
- 设备绑定：可选 `--bind` 记录机器指纹，允许 2 台设备（防共享，不强制）。
- 防盗版预期：Go 二进制可被 patch，**接受这一点**；商业保护靠持续价值、更新、支持与商标，而不是 DRM。

**支付与开票**

| 市场 | 渠道 | 适用性 | 备注 |
|---|---|---|---|
| 国际 | Polar / Lemon Squeezy / Paddle（Merchant of Record） | 个人可开通；自动处理 VAT/税率；支持 License Key 自动发放 | 费率与可用性以官网为准（需核对），一般 4–6% + 固定费 |
| 国内（启动期） | 面包多 / 爱发电 | 个人即可收款，适合早鸟预售 | 手动发 key，先验证付费意愿 |
| 国内（正式） | 支付宝当面付 / 微信支付（需个体户或公司主体） | 可开发票、可持续经营 | 建议 3 个月内注册主体 |
| 企业 | 对公转账 + 合同 + 发票 | Enterprise 必须 | 需要合同模板与 SLA 附件 |

**不自建账单系统**：早期用托管 checkout（Polar/Lemon Squeezy/面包多）+ 邮件发 key；等月流水 > ¥2 万再考虑自建。

### 4.5 收入模型（12 个月，三档情景）

假设：GitHub star → 下载 10%，下载 → 试用 30%，试用 → 付费 3–8%。

| 情景 | Pro 付费 | Team | Enterprise | 年收入估算 |
|---|---|---|---|---|
| 保守 | 30 × ¥199 | 2 × ¥899 | 1 × ¥30,000 | ≈ ¥37,768 |
| 基准 | 100 × ¥199 | 8 × ¥899 | 3 × ¥30,000 | ≈ ¥116,892 |
| 乐观 | 300 × ¥199 | 20 × ¥899 | 6 × ¥30,000 | ≈ ¥257,680 |

成本（年）：支付手续费 ~5%、设计/内容外包 ¥2–4 万、服务器/域名 ¥3 千、法务/主体 ¥5 千。**基准情景净利率约 60–70%**。

> 这些是**目标与假设**，不是承诺。第一年真正的 KPI 是：**有没有 10 个愿意付钱的真实用户**，以及他们的留存。

### 4.6 GTM：90 天启动计划

| 周 | 动作 | 产出 |
|---|---|---|
| W1–2 | 修 LICENSE/版本/admin；发 v4.0.0；定价页上线；早鸟预售（面包多/爱发电） | 一个可信的 Release + 第一批 ¥99 订单 |
| W3–4 | 控制台 v1 + 5 分钟上手视频；掘金/知乎/V2EX 各 1 篇；提交 5 个 MCP 目录 + awesome-mcp-servers PR | 首个 100 下载 |
| W5–6 | benchmark 报告；诚实对比页（Cortex vs XERJ vs Dify）；Obsidian 插件 beta | 可引用的数字与第三方渠道 |
| W7–8 | 英文 Show HN / Reddit r/LocalLLaMA；种子用户 20 人访谈（每周 3 个） | 反馈清单 + 首批 Pro 转正 |
| W9–10 | 检索质量包上线（rerank/高亮）；MCP 工具扩到 12+ | 留存提升，NPS 调研 |
| W11–12 | Pro 正式开卖；Team 版内测（3 个团队）；复盘 KPI | 付费漏斗数据 + 下季度计划 |

**渠道优先级**：GitHub/README（最高）→ 中文技术社区 → MCP 目录/Obsidian 商店 → 英文 HN/Reddit → 集成商/云市场（企业单）。
**内容选题（10 个）**：给 Claude Code 装永久记忆；本地知识库选型；Cortex vs XERJ 诚实对比；私有化 AI 知识库合规；Obsidian 接入；10 万文档压测；中文 embedding 选型；MCP 工具开发；把聊天记录变成记忆；一个人维护开源项目的商业化复盘。

### 4.7 KPI 与漏斗

| 阶段 | 指标 | 目标（6 个月） |
|---|---|---|
| 获客 | GitHub stars、Release 下载数 | 300 stars / 3,000 下载 |
| 激活 | 完成 index 并成功调用一次 MCP 的比例 | ≥ 50% |
| 留存 | 周活（7 日内有检索/写入） | ≥ 30% |
| 付费 | 试用→付费转化 | ≥ 5% |
| 收入 | ARR、ARPU、churn | ARR ≥ ¥10 万；churn < 5%/月 |
| 口碑 | NPS、issue 响应时长 | NPS ≥ 40；< 48h |

---

## 5. 组织与执行

- **人力**：1 人全职产品/研发 + 0.5 人内容/设计（外包）；企业单出现后再加 0.5 支持。
- **节奏**：每周一个 patch release；每月一个 minor；每季度一次公开复盘（写进博客，积累信任）。
- **时间分配（前 8 周）**：60% 产品（P0/P1）、20% 内容、20% 用户访谈与销售。
- **决策原则**：先验证付费，再扩功能；先 10 个真实付费用户，再谈 1,000 stars；每个 P2 功能必须有客户/合同驱动。

---

## 6. 风险与合规

| 风险 | 概率 | 影响 | 对策 |
|---|---|---|---|
| 许可证混乱劝退客户/贡献者 | 高 | 高 | 本周统一；README/网站/二进制同源 |
| AGPL 导致企业客户法务否决 | 中 | 高 | 选 A（Apache-2.0 核心）或提供商业许可 |
| XERJ 继续扩张，挤压生存空间 | 高 | 中 | 不打引擎战；聚焦中文/记忆/控制台/私有化 |
| 大厂（飞书 ima/腾讯乐享/语雀）下探 | 中 | 中 | 做「本地 + 可编程 + MCP」它们不做的形态 |
| 支撑成本失控（企业客户） | 中 | 中 | 明确 SLA 分级；Enterprise 才给 4h 响应 |
| 支付/发票合规（个人主体） | 中 | 中 | 早鸟用个人渠道；3 个月内注册主体 |
| 盗版/共享 key | 高 | 低 | 接受弱 DRM；靠价值/更新/支持 |
| 数据安全责任 | 低 | 高 | 明确「本地优先、默认无遥测」；隐私政策与免责条款 |
| 单一维护者风险 | 高 | 高 | 公开 roadmap；关键模块文档化；逐步引入 1–2 个协作者 |

**合规清单**：LICENSE 与 README 一致；ToS/隐私政策（可先用模板）；商标声明；企业合同模板 + SLA；退款政策（建议 14 天无理由）；软件著作权登记（国内，可选）。

---

## 7. 立即执行清单（本周就能开始）

1. **今天**：决定许可证走向（推荐 Apache-2.0 核心），改 `LICENSE` 与 README；建 `internal/version` 单一版本源。
2. **Day 2–3**：修 `/admin` 模板 bug，先让页面有内容（数据接 `/v1/stats`）。
3. **Day 4–5**：tag v4.0.0，CI 注入版本号，产出 6 平台包 + SHA256 + release notes。
4. **Day 6–7**：写 `cortex license activate/status`（Ed25519），把 `cortex usage` 接真实 tier。
5. **第 2 周**：定价页 + 早鸟预售链接（面包多/爱发电），手动发前 20 个 key。
6. **第 2 周**：`scripts/bench/` 最小可用（索引吞吐 + P95 + recall），把结果写进 README。
7. **第 3 周**：5 分钟 Quickstart 视频 + llms.txt + AGENTS.md + 对比页；提交 5 个 MCP 目录。
8. **第 4 周**：控制台 v1 完整（搜索/文档/记忆/任务/API Key），发 v4.1.0，开始收集 20 个种子用户。

---

## 8. 参考来源

- Dify Pricing（Cloud Professional $590/工作区/年，Team $1,590/年）：https://dify.ai/pricing
- XERJ Pricing（Community 免费 / Professional 按节点 / Enterprise 定制；同一二进制）：https://xerj.org/pricing/
- XERJ 仓库与 README（Apache-2.0、20 MCP 工具、ES 兼容、基准）：https://github.com/xerj-org/xerj
- Mem0 平台文档（托管记忆层、企业治理/审计）：https://docs.mem0.ai/platform/overview
- Cortex 仓库内部证据：`LICENSE`、`README.md`、`internal/storage/license.go`、`internal/cli/init.go:661-690`、`internal/api/admin.go:14`、`internal/api/admin.go:40-48`、`.github/workflows/build.yml`、`internal/config/config.go`
- 数据时点：2026-10-22；GitHub star 数来自 shields.io；Dify/XERJ 价格来自官网当前页面，签约前请复核。

## 附录 A：定价页文案（可直接改）

> **把 AI Agent 的记忆，留在你自己的电脑里。**
> Cortex 是一个单文件安装的本地知识库：索引你的代码、文档和笔记，让 Claude Code / Cursor / OpenCode 通过 MCP 随时检索。数据不上云，不需要 Docker，不需要 Python。

| | Free | Pro | Team | Enterprise |
|---|---|---|---|---|
| 价格 | ¥0 | **¥199/年** | **¥899/年**（3 席） | 联系我们 |
| 适合 | 试用/个人 | 独立开发者 | 5–20 人团队 | 合规/私有化 |
| 文档量 | 5,000 | 500,000 | 1,000,000 | 不限 |
| Web 控制台 | 状态页 | ✅ | ✅ | ✅ |
| 自动索引/备份 | 手动 | ✅ | ✅ | ✅ |
| 团队共享/RBAC | — | — | ✅ | ✅ |
| 审计/SSO | — | — | OIDC | SAML/SCIM |
| 支持 | 社区 | 邮件 48h | 邮件 24h | SLA 4h |

> 早鸟：前 100 名 Pro **¥99 / 2 年**。所有付费版本均含商业使用许可；14 天无理由退款。

## 附录 B：销售 FAQ / 异议处理

| 客户说 | 回应 |
|---|---|
| 「免费版够用了」 | 免费版限 5,000 文档、单库手动索引；Pro 卖的是自动索引、控制台、rerank 和大规模，不是把免费版做残。 |
| 「为什么不直接用 XERJ？」 | XERJ 是引擎，适合工程团队自建；Cortex 是装好即用的中文产品，带控制台和记忆管理。两者可以并存（都用 MCP）。 |
| 「AGPL 我们法务不接受」 | 我们提供商业许可（Enterprise 含豁免）；同时核心将切换/已切换为 Apache-2.0（以仓库 LICENSE 为准）。 |
| 「数据安全怎么保证？」 | 默认本地、无遥测；可 air-gapped；license 校验可离线。企业版提供加密 at-rest 与审计导出。 |
| 「能不能便宜点/按人买？」 | Team 按 3 席起，额外席位 ¥199/席/年；教育/开源 5 折。 |
| 「有没有案例？」 | 早期客户可签「案例授权」，用 3 个月免费换公开案例与 logo。 |

