<p align="center">
  <img src="docs/assets/img/cortex-logo-static.svg" width="110" height="110" alt="Cortex Logo">
</p>

<h1 align="center">Cortex</h1>

<p align="center">
  <b>让 Claude Code / Cursor 记住你的所有笔记、文档和代码。</b><br>
  <sub>单文件安装，不用 Docker，不用数据库，5 分钟接上。</sub>
</p>

<p align="center">
  <a href="https://github.com/lh123aa/cortex/releases/latest"><img src="https://img.shields.io/badge/下载-v4.0.1-7B61FF?style=flat-square" alt="Version"></a>
  <img src="https://img.shields.io/badge/许可-Apache--2.0-blue?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/二进制-33--36MB-green?style=flat-square" alt="Size">
  <img src="https://img.shields.io/badge/测试-24_24_包通过-success?style=flat-square" alt="Tests">
  <a href="https://github.com/lh123aa/cortex/actions"><img src="https://img.shields.io/github/actions/workflow/status/lh123aa/cortex/build.yml?style=flat-square&logo=github" alt="Build"></a>
  <a href="https://github.com/lh123aa/cortex/stargazers"><img src="https://img.shields.io/github/stars/lh123aa/cortex?style=flat-square&logo=github" alt="Stars"></a>
</p>

<p align="center">
  <a href="#-30-秒看到效果"><b>30 秒看到效果</b></a> ·
  <a href="#-why-cortex">为什么是 Cortex</a> ·
  <a href="#-performance-measured-not-estimated">性能数据</a> ·
  <a href="#-mcp-tools">MCP 工具</a> ·
  <a href="#-api">API</a> ·
  <a href="README.en.md">English</a>
</p>

---

## 🎬 30 秒看到效果

```bash
# 1. 安装（一条命令，不需要 Go / Node / Docker）
curl -fsSL https://raw.githubusercontent.com/lh123aa/cortex/master/scripts/install.sh | sh

# 2. 索引你的知识库
cortex index ~/notes ~/Documents

# 3. 启动并自动接入你的 AI 工具
cortex start
```

`cortex start` 会做三件事：

- 启动 MCP 服务器（stdio）+ REST API
- **自动写入你的 AI 工具配置**（Claude Code / Cursor / OpenCode / Trae），带备份，不覆盖已有 MCP server
- 打印接入指引

然后回到 Claude Code，直接问：

> 「我之前写在笔记里关于 Redis 缓存穿透的那段怎么处理的？」

Claude 会通过 Cortex 的 MCP 工具检索你的笔记，而不是猜。

---

## 💡 Why Cortex

### 你可能正在遇到的问题

| 你的痛点 | 现有方案 | Cortex 的做法 |
|---|---|---|
| AI 聊完就忘，每次都要重新粘贴上下文 | 手动复制粘贴 | **持久化记忆层**，MCP 工具让 Agent 主动检索 |
| 想把 2 万个笔记接进 Cursor，结果要装 Docker、跑向量库、配 5 个服务 | Dify / RAGFlow / AnythingLLM | **单文件 33–36 MB，`cortex start` 一条命令** |
| 中文笔记搜不准（"机器学习"搜不到"机器学习模型"） | 通用分词器 | **CJK 专用分词**，标题与正文分别处理 |
| 数据必须留在本地（客户资料、代码、病历） | 云端 RAG 服务 | **100% 本地**，只有你配的 embedding API 会出网 |
| 云端按量计费，团队 10 个人每月几百刀 | 托管服务 | **Apache-2.0 免费**，自建零边际成本 |

### 和同类工具的区别

| | Cortex | AnythingLLM | Dify | Chroma / Qdrant |
|---|---|---|---|---|
| 部署 | **单二进制 + SQLite** | Docker / Desktop | Docker Compose 多服务 | 需自建检索服务 |
| 安装到可用 | **~1 分钟** | ~5 分钟 | ~15 分钟 | ~30 分钟 |
| 中文分词 | **✅ CJK 专用** | 通用 | 通用 | 无（只管向量） |
| 国内 embedding | **✅ 通义 / 智谱 / 文心** | ❌ | 部分 | ❌ |
| MCP 原生 | **✅ 14 个工具 + 自动注册** | ✅ | 插件式 | ❌ |
| Agent 记忆 | **✅ 独立记忆 API** | 会话级 | 会话级 | ❌ |
| 许可 | **Apache-2.0** | MIT | Apache-2.0 + 附加条款 | Apache-2.0 |

> 诚实说明：Cortex 支持 **25 种格式**，比 XERJ（40+）少；没有 Elasticsearch 兼容 DSL；
> 英文长尾查询比中文慢。这些是真实短板，不是谦虚。

---

## 📊 Performance: measured, not estimated

全部可由 [`cmd/cortex-bench`](benchmarks/README.md) 复现。环境：Windows 11 / x86_64 / 本地 SSD / `CGO_ENABLED=0` / FTS-only。

### 检索延迟（27,157 文档 / 84,562 分块的真实知识库）

| 指标 | 实测值 |
|---|---|
| **P50** | **0.43 ms** |
| P95 | **2.35 ms** |
| P99 | 3.91 ms |
| 吞吐（单实例） | **1,440 QPS** |
| 响应体（10 条结果） | **5.8 KB** |
| 冷启动 | **0.36 s** |
| 常驻内存 | **53 MB** |

### 索引吞吐（合成语料，可复现）

| 语料规模 | 吞吐 | 分块数 |
|---|---|---|
| 1,000 文档 | **328 files/s** | 2,510 |
| 5,000 文档 | **365 files/s** | 12,668 |

### 检索质量

| 指标 | 1,000 文档 | 5,000 文档 |
|---|---|---|
| Recall@1 | **1.000** | **1.000** |
| Recall@5 | **1.000** | **1.000** |
| Recall@10 | **1.000** | **1.000** |
| MRR | **1.000** | **1.000** |

> **关于这些数字的边界**（请务必阅读）：
> - 合成语料的 recall 证明的是「检索管线正确、无系统性漏召」，**不代表语义检索质量**（真实质量需要 embedding + 重排）。
> - Windows 计时器分辨率是 507µs，亚毫秒延迟无法精确测量，P50 标注为实测下限。
> - 索引吞吐语料与 XERJ 不同（XERJ 含二进制解析、支持 40+ 格式），**不能直接横比**。

完整报告：[性能优化记录](docs/launch/05-performance-optimization.md) · [基准方法与边界](benchmarks/README.md)

---

## ✨ 核心能力

<table>
<tr><td width="50%" valign="top">

**🔍 混合检索**
- FTS5 BM25（SQLite 原生全量索引）
- 向量检索 + RRF 融合排序
- CJK 分词：标题与正文分别处理
- 两級缓存（内存 L1 + SQLite L2，异步写入）

**🧠 Agent 记忆**
- `/v1/memory` 写入与批量写入
- 记忆检索与 RAG 上下文构建
- 分块去重（MinHash）

</td><td width="50%" valign="top">

**🔌 MCP 原生**
- 14 个工具，stdio + Streamable HTTP (SSE)
- 自动注册 Claude Code / Cursor / OpenCode / Trae
- 写入前备份，保留你已有的 MCP server
- 无配置即用

**📦 单二进制**
- 33–36 MB，6 平台（linux/darwin/windows × amd64/arm64）
- 纯 Go，无 CGO，无外部数据库
- SQLite + 向量索引内嵌
- Windows / macOS / Linux 同等支持

</td></tr>
<tr><td valign="top">

**🔐 企业级基础设施**
- RBAC：`admin` / `editor` / `viewer` 三角色
- Ed25519 离线许可（feature 门控）
- 配额强制（文档数 / 存储）
- 审计日志（企业版）

</td><td valign="top">

**📈 可观测**
- 32 个 Prometheus 指标
- 内置 Grafana 面板
- `/health` `/health/ready` `/health/live`
- Web 控制台（`/admin`，默认仅本机）

</td></tr>
</table>

---

## 🔌 MCP Tools

Cortex 向 AI Agent 暴露 **14 个 MCP 工具**：

| 工具 | 作用 |
|---|---|
| `cortex_search` | 混合检索知识库，返回带路径与标题的结果 |
| `cortex_context` | 构建 RAG 上下文（适合直接塞进 prompt） |
| `cortex_suggest` | 查询补全建议 |
| `cortex_memory_write` | 写入一条记忆 |
| `cortex_memory_batch` | 批量写入记忆 |
| `cortex_memory_search` | 检索记忆 |
| `cortex_memory_delete` | 删除指定记忆 |
| `cortex_memory_delete_batch` | 批量删除记忆 |
| `cortex_code_search` | 按函数/类名精确检索代码符号（Go 用官方 AST） |
| `cortex_stats` | 知识库统计 |
| `cortex_health` | 健康检查 |
| `cortex_map` | 知识库主题地图 |
| `cortex_plan` | 基于知识库内容生成执行计划 |
| `cortex_project_context` | 项目级上下文聚合 |

手动接入（如果 `cortex start` 没覆盖你的工具）：

```json
{
  "mcpServers": {
    "cortex": {
      "command": "cortex",
      "args": ["mcp"]
    }
  }
}
```

---

## 🛠️ 安装

### 一行安装（推荐，无需 Go）

```bash
# Linux / macOS
curl -fsSL https://raw.githubusercontent.com/lh123aa/cortex/master/scripts/install.sh | sh

# Windows PowerShell
irm https://raw.githubusercontent.com/lh123aa/cortex/master/scripts/install.ps1 | iex
```

安装脚本会：下载对应平台预编译二进制 → 校验 SHA256 → 安装到 `PATH` → 运行 `cortex install`。

指定版本：

```bash
CORTEX_VERSION=v4.0.1 sh scripts/install.sh
```

### 从源码构建（需要 Go 1.25+）

```bash
git clone https://github.com/lh123aa/cortex.git
cd cortex
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o cortex ./cmd/cortex
./cortex install ~/notes
```

### Docker

```bash
docker build -t cortex .

# 索引（挂载你的文档目录与数据目录）
docker run --rm -v ~/notes:/data -v ~/.cortex:/root/.cortex \
  cortex index /data

# 启动 REST API（cortex serve 默认端口 2021，需显式映射）
docker run -d --name cortex \
  -p 2021:2021 -p 9090:9090 \
  -v ~/.cortex:/root/.cortex \
  cortex serve -p 2021
```

> ⚠️ 官方 `Dockerfile` 的 `EXPOSE` 写的是 8080/9090，而 `cortex serve` 默认监听 **2021**。
> 以上命令用 `-p 2021:2021` 并显式传 `-p 2021` 是一致的做法。

---

## ⚡ CLI

17 个命令，核心 5 个：

| 命令 | 用途 |
|---|---|
| `cortex index <path>` | 索引文档（实时进度、断点续传、并发 8–16 worker） |
| `cortex start` | 一键启动 MCP + REST + 自动注册 AI 工具 |
| `cortex search <query>` | 命令行检索 |
| `cortex watch <path>` | 监听目录，增量索引 |
| `cortex status` | 查看文档数、分块数、用量 |

其他：`install` `serve` `mcp` `setup` `context` `dedup` `usage` `user` `license` `version`。

---

## 📡 API

**19 个 REST 端点**。核心：

```bash
# 检索
curl 'http://localhost:2021/v1/search?q=Redis%20缓存穿透&limit=5'

# 写入记忆
curl -X POST http://localhost:2021/v1/memory \
  -H 'Content-Type: application/json' \
  -d '{"content":"今天解决了缓存穿透问题，用布隆过滤器"}'

# RAG 上下文
curl 'http://localhost:2021/v1/context?q=缓存方案&limit=5'
```

| 类别 | 端点 |
|---|---|
| 检索 | `GET /v1/search` `GET /v1/context` `GET /v1/suggest` |
| 文档 | `GET /v1/docs` `GET /v1/docs/:id` `GET /v1/stats` |
| 记忆 | `POST /v1/memory` `POST /v1/memory/batch` `GET /v1/memory/search` `GET /v1/memory/context` `DELETE /v1/memory/:id` |
| MCP | `/mcp`（Streamable HTTP + SSE） |
| 认证 | `POST /auth/register` `POST /auth/login` `POST /auth/logout` |
| 健康 | `GET /health` `GET /health/ready` `GET /health/live` |
| 监控 | `GET :9090/metrics`（独立端口，32 个 Prometheus 指标） |
| 管理 | `GET /admin`（Web 控制台，默认仅本机）<br>`GET /admin/keys`（仅 admin 角色） |

> 默认只返回 800 字符摘要。需要全文传 `?full=1`（响应体会从 5.8 KB 涨到 ~1.9 MB）。

### 角色权限（RBAC）

开启 `auth_enabled: true` 后：

| 角色 | 读 | 写 | 管理用户/密钥 |
|---|:---:|:---:|:---:|
| `viewer` | ✅ | ❌ | ❌ |
| `editor` | ✅ | ✅ | ❌ |
| `admin` | ✅ | ✅ | ✅ |

```bash
cortex user add alice --role admin --password '...'
cortex user set-role bob viewer
```

---

## 📦 支持格式（25 种）

| 类别 | 格式 |
|---|---|
| 文档 | `.md` `.pdf` `.docx` `.txt` |
| 代码 | `.go` `.py` `.js` `.ts` `.java` `.cpp` `.c` `.rb` `.sh` `.ps1` `.css` `.html` |
| 配置 | `.yaml` `.json` `.toml` `.ini` `.cfg` `.hcl` `.env` `.properties` `.xml` |

**代码符号**：Go 使用官方 `go/ast` 解析器（精确）；其他语言使用词法提取。

---

## 🔧 配置

```yaml
# ~/.cortex/config.yaml
cortex:
  db_path: ~/.cortex/cortex.db
  log_level: info
  auth_enabled: false

embedding:
  provider: none            # none | ollama | openai | cohere | voyage | dashscope | zhipu | baidu
  ollama:
    base_url: http://localhost:11434
    model: nomic-embed-text
  openai:
    api_key: ""

index:
  workers: 8                # 并发索引 worker
  max_tokens: 512           # 单块最大 token
  overlap_tokens: 64        # 块间重叠
  min_chars: 50             # 低于此字符数的文档会被跳过

search:
  cache_ttl: 5m
  default_top_k: 10
  prefetch: false           # 预联想（消耗额外内存）
```

### Embedding 提供商（7 家 + 零依赖模式）

`none`（纯 FTS，零依赖，默认）｜ `ollama`（本地，推荐）｜ `openai` ｜ `cohere` ｜ `voyage` ｜ `dashscope`（阿里通义）｜ `zhipu`（智谱）｜ `baidu`（百度文心）

内置常用模型预设：`nomic-embed-text`、`bge-m3`、`text-embedding-3-small/large`、`embed-multilingual-v3.0`、`voyage-3` 等。

```bash
cortex setup    # 交互式配置向导
```

---

## 🏗️ 架构

```
┌─────────────────────────────────────────────────┐
│  Claude Code · Cursor · OpenCode · Trae          │
└──────────────┬──────────────────────────────────┘
               │ MCP (stdio / Streamable HTTP)
┌──────────────▼──────────────────────────────────┐
│              Cortex (单二进制)                    │
│ ┌────────────┬────────────┬───────────────────┐ │
│ │ MCP Server │ REST API   │ Web Console       │ │
│ │ 14 tools   │ 19 routes  │ /admin            │ │
│ └─────┬──────┴─────┬──────┴────────┬──────────┘ │
│       │            │               │            │
│ ┌─────▼────────────▼───────────────▼──────────┐ │
│ │ Hybrid Search: FTS5 BM25 + Vector + RRF     │ │
│ └─────┬───────────────────────────────────────┘ │
│ ┌─────▼───────────────────────────────────────┐ │
│ │ SQLite (文档/分块/FTS/向量/符号/缓存/用户)   │ │
│ └─────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────┘
```

设计取舍：
- **SQLite 而非独立向量库** —— 目标是「下载即用」，免运维。规模上限约 3 万文档。
- **单二进制而非容器编排** —— 个人/小团队的真实使用场景是笔记本，不是 K8s。
- **FTS-only 默认** —— 零依赖启动；需要语义检索时装 Ollama 开 embedding。

---

## 🔐 开源与商业

**开源版（Apache-2.0）功能完整可用，无功能阉割。**

| 能力 | 开源版现状 |
|---|---|
| 索引 / 混合检索 / MCP 14 工具 / REST API / CLI | ✅ 可用 |
| 中文化检索（CJK 分词） | ✅ 可用 |
| Agent 记忆 API | ✅ 可用 |
| RBAC（admin / editor / viewer） | ✅ 可用 |
| 离线许可机制（Ed25519 签发与激活） | ✅ 可用 |
| 文档配额强制 | ✅ 可用（Free：5,000 篇 / 1 GB） |
| Web 控制台（`/admin`，默认仅本机） | ✅ 可用 |

> **关于付费版：目前没有可购买的付费版。**
>
> 说清楚以免误导：Ed25519 离线许可与配额门控的**机制已经完整可用**，但
> *审计日志*、*SSO / OIDC*、*S3 备份*、*多租户* 等企业能力**尚未实现**，
> `pkg/ext` 扩展接口目前是空的。这些能力列在路线图上，不是现有功能。

### 需要私有化部署支持？

如果你的场景要求**数据不出内网**、需要中文检索、又不想维护 Docker / 向量库编排，
Cortex 的单二进制形态正好合适。可以提供远程部署与定制支持（按次/按单计费的**服务**，
不是软件订阅——软件本身永远开源免费）。

**怎么发起：** 开一个 [部署咨询 Issue](https://github.com/lh123aa/cortex/issues/new?template=deployment_inquiry.md)，
表单会问你 5 个问题（文档量、格式、环境、使用者、期望查询）。一般 24 小时内回第一句。

**能做什么：** 远程部署跑通、全量索引、检索效果调优、MCP / HTTP API 接入联调、一次使用培训。
**不能做什么（如实说明）：** 审计日志、SSO / OIDC、S3 备份、多租户这些能力代码里**还没有**，
不会在报价单上出现；也不做客户数据托管或 7×24 SLA 承诺。

服务按次/按单计价（部署 ¥500 起，私有化部署 + 调优 ¥3,000 起），具体以 Issue 沟通后的报价单为准。

许可迁移记录见 [LICENSE_MIGRATION.md](docs/planning/LICENSE_MIGRATION.md)。
路线图见 [docs/planning/PRODUCT_ITERATION_AND_MONETIZATION_PLAN.md](docs/planning/PRODUCT_ITERATION_AND_MONETIZATION_PLAN.md)。

---

## 🧪 质量

| 项 | 状态 |
|---|---|
| `go vet ./...` | 0 警告 |
| `go test ./...` | **24/24 包通过** |
| 跨平台构建 | 6/6（linux/darwin/windows × amd64/arm64） |
| 安装验证 | Windows + Linux 实跑官方安装命令 |
| 基准可复现 | `go run ./cmd/cortex-bench all --docs 5000` |

回归测试覆盖真实事故：[索引统计丢失](internal/index/short_doc_test.go) · [符号表列名](internal/storage/bundle_test.go) · [登录 token 失效](pkg/license/license_test.go) · [RBAC 权限矩阵](internal/api/rbac_test.go)

---

## ❓ FAQ

**Q：和 XERJ 什么关系？**
同一需求、不同层级。XERJ 是引擎（ES 兼容、40+ 格式），Cortex 是产品（中文优先、单二进制、MCP 开箱即用）。我们索引吞吐已超过它的公开数据，但**格式支持仍是短板**。

**Q：能索引多少文档？**
舒适区约 3 万文档。5,000 文档实测 P95 6 ms，27,157 文档 P95 2.35 ms。

**Q：数据会上传吗？**
不会。默认全本地。只有你主动配置的 embedding API 会出网（可以用 Ollama 完全离线）。

**Q：为什么不用 PostgreSQL + pgvector？**
目标是「下载即用」。SQLite 免运维，单人/小团队够用。百万级是 v5 的话题。

**Q：中文搜索怎么做的？**
CJK 单字分词 + 标题/正文分别处理 + 查询时 2-gram 展开。解决了「机器学习」搜不到「机器学习模型」的问题。

**Q：支持 Windows 吗？**
支持，Windows 是主力测试平台之一，有 PowerShell 一键脚本。

---

## 🤝 贡献

欢迎 Issue、PR、文档修正。见 [CONTRIBUTING.md](CONTRIBUTING.md)。

Apache-2.0 下提交即按同许可授权（DCO）。企业版相关目录需要 CLA。

---

## 📄 许可证

[Apache License 2.0](LICENSE) —— 可自由使用、修改、分发、商用（含闭源产品），保留 [NOTICE](NOTICE) 即可。

Apache-2.0 **不授予商标权**：「Cortex」名称与 Logo 需单独授权用于商业产品名。

---

<p align="center">
  <sub>单二进制 · 零配置 · MCP 原生 · 完全本地 · Apache-2.0</sub><br><br>
  <b>如果这个项目帮你省下了折腾 RAG 栈的时间，欢迎 ⭐</b>
</p>
