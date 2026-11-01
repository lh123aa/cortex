# Cortex 宣发文案包（v4.0.0）

> 所有数字来自 `docs/launch/02-benchmark.md` 实测。
> 原则：不夸大、不虚构 benchmark、不和 XERJ 比索引速度。
> 中文为主，英文附一版。

---

## 1. README 首屏（GitHub）

```markdown
# Cortex 🧠

**给 AI Agent 装一个 5 分钟能用、中文搜得准、完全本地的记忆库。**

单二进制 37 MB · 无 Docker · 无外部数据库 · MCP 原生 · Apache-2.0

[![Release](https://img.shields.io/badge/release-v4.0.0-blue)]()
[![License](https://img.shields.io/badge/License-Apache--2.0-blue)]()

```bash
# 1 分钟安装
curl -fsSL https://raw.githubusercontent.com/lh123aa/cortex/main/scripts/install.sh | sh
cortex start          # MCP + REST 一起起来，自动接上 Claude Code / Cursor
cortex index ~/notes  # 索引你的知识库
```

**实测数字**（Windows，27,157 文档 / 84,562 分块）
- 冷启动 **0.36 s**，常驻内存 **53 MB**
- 检索延迟 **P50 1 ms / P95 20 ms**
- 单实例 **400+ QPS**，零外部依赖
```

## 2. Hacker News（英文，Show HN）

```text
Show HN: Cortex – a single-binary, local-first knowledge base for AI agents

I got tired of "just run docker-compose" for a RAG stack, so I built a
knowledge base that ships as one static Go binary (37 MB, no CGO) and speaks
MCP natively.

What it does:
- Hybrid search: SQLite FTS5 (BM25) + vectors + RRF fusion
- Stores notes/docs/source code with heading-aware chunking
- 14 MCP tools, auto-registers into Claude Code / Cursor / OpenCode
- REST API + a tiny local console + Prometheus metrics
- Chinese-first segmentation (CJK bigrams), 9 embedding providers

Measured on my machine (Windows, 27k docs / 84k chunks):
- cold start 0.36 s, RSS 53 MB
- search P50 1 ms, P95 20 ms
- index throughput ~106–127 files/s

Honest limitations: no ES-compatible DSL, no tree-sitter yet (Go uses the
official parser; other languages are lexical), and English long-tail queries
are slower than Chinese.

Apache-2.0. Enterprise features (audit log, SSO, teams) live in a separate
proprietary binary that plugs in through a public extension interface.

https://github.com/lh123aa/cortex
```

## 3. Twitter / X（中英各一）

**中文**

```text
给 Claude Code / Cursor 装一个本地记忆库，要几步？
1. curl 安装脚本
2. cortex start
3. cortex index ~/notes

单二进制 37MB，无 Docker，无外部数据库。
27k 文档实测：冷启动 0.36s，检索 P50 1ms，内存 53MB。
中文分词 + 9 家 embedding + 14 个 MCP 工具。
Apache-2.0 开源 🧠
github.com/lh123aa/cortex
```

**English**

```text
Your AI agent forgets everything between sessions.

Cortex fixes that with one static binary (37 MB, no Docker):
→ cortex start && cortex index ~/notes
→ 14 MCP tools, auto-wired into Claude Code / Cursor
→ 27k docs: cold start 0.36s, search P50 1ms, RSS 53MB
→ Apache-2.0

github.com/lh123aa/cortex
```

## 4. 掘金 / 知乎 / V2EX（中文长文标题 + 摘要）

**标题备选**
1. 《我给 Claude Code 写了个本地记忆库：单二进制 37MB，检索 1ms》
2. 《为什么我把 RAG 栈塞进了一个 37MB 的 Go 二进制》
3. 《实测 2.7 万文档：Cortex 的索引、检索与内存占用》

**开头钩子**

> 你有没有遇到过这种情况：跟 AI 聊了三个小时，关掉窗口，第二天它什么都不记得。
> 或者：想把 2 万篇笔记接进 Claude Code，结果发现有 4 种方式可以装 Docker……
>
> 我做了一个东西：单个二进制、无外部依赖、MCP 原生。
> `cortex start` 之后，Claude Code / Cursor 就能搜你的知识库。

**正文结构**

1. 问题：Agent 没有长期记忆，本地 RAG 栈太重
2. 方案：单二进制 + SQLite(FTS5) + 内存向量 + RRF
3. 实测：三组语料（本仓库 / 851 文件文档库 / 2.7 万文档真实库）
4. 踩坑：中文分词、SQLite 并发、基准语料污染（真实故事）
5. 边界：不支持 ES DSL、格式支持有限、英文长尾慢
6. 开源与商业：Apache-2.0 核心 + 企业版

**结尾**

> 如果你也在用 Claude Code / Cursor，欢迎试一下，star 就是最大的支持。
> 遇到问题直接开 issue，我会回。

## 5. V2EX 短帖

```text
标题：[分享] 单二进制、MCP 原生的本地知识库，给 Claude Code 用

正文：
做了个 Go 写的本地知识库/记忆库，无 Docker、无外部数据库，一个 37MB 的二进制。
- MCP 原生，cortex start 后自动写入 Claude Code / Cursor 配置
- SQLite FTS5 + 向量 + RRF 混合检索，中文分词做了专门处理
- 27k 文档实测：冷启动 0.36s，检索 P50 1ms，内存 53MB
- Apache-2.0

不吹的地方：格式支持不如 XERJ 全，英文查询长尾比中文慢，没有 ES 兼容 DSL。
https://github.com/lh123aa/cortex
```

## 6. 邮件 / 私信模板（找种子用户）

```text
Hi <name>,

看到你在用 Claude Code 做 <场景>，想请你试一个我做的工具：
Cortex —— 单二进制本地知识库，MCP 原生，装完 cortex start 就能被 Claude Code 检索。

它解决的是「Agent 没有长期记忆」：把笔记/文档/代码索引进去，
之后 Claude Code 通过 MCP 直接搜，不用每次都贴上下文。

不需要 Docker，37MB 一个文件，Windows/Mac/Linux 都有。
如果不合适直接告诉我，不会打扰第二次。

https://github.com/lh123aa/cortex
```

## 7. FAQ（评论区高频问题准备）

| 问题 | 回答 |
|---|---|
| 和 XERJ 什么关系？ | 同需求不同层级。XERJ 是引擎（ES 兼容、格式最全），Cortex 是产品（中文、单二进制、MCP 开箱即用）。 |
| 支持 embedding 吗？ | 支持 9 家；默认 FTS-only 是为了零依赖启动，装了 Ollama 可开语义检索。 |
| 能索引多少文档？ | 舒适区 ≤3 万；5k 文档实测 P95 6 ms，27k 文档 P95 20 ms。 |
| 数据会上传吗？ | 不会。默认全本地，只有你配置的 embedding API 会出网。 |
| 为什么不用 Postgres/pgvector？ | 目标是「下载即用」，SQLite 免运维；规模需求是 v5 的事。 |
| 商业版卖什么？ | 审计日志、SSO、团队多租户、加密、HA、ES 兼容。核心永久 Apache-2.0。 |
| 有没有 Windows 版？ | 有，6 平台交叉编译，Windows 是主力测试平台之一。 |

## 8. 宣发红线（团队内部约定）

1. 不写「支持所有格式」「兼容 ES」「比 XERJ 快」这类无法核实的表述。
2. 所有性能数字必须来自 `docs/launch/raw/` 下的原始 JSON，且注明环境。
3. 合成基准的 recall 不能当「语义检索质量」宣传。
4. 免费版功能不阉割，只限规模 —— 文案里要明确，避免被骂「假开源」。
5. 企业版能力用「独立二进制 + 按 License 启用」描述，不说「锁功能」。
