# Cortex 推广提交追踪

> 更新时间：2026-05-06

---

## 提交清单

### GitHub 站内（仓库设置，需手动操作）

| 设置项 | 建议值 | 状态 |
|--------|--------|:----:|
| **Description** | `Cortex - AI Agent 的第二大脑。单二进制本地知识库，MCP 原生支持，混合搜索 + Agent 记忆系统。` | ⬜ |
| **Topics** | `go, ai-agent, mcp, model-context-protocol, knowledge-base, rag, vector-search, hnsw, bm25, sqlite, memory-system, agent-memory, local-ai, llm, embedding, hacktoberfest, golang, semantic-search` | ⬜ |
| **Website** | `https://github.com/lh123aa/cortex` | ⬜ |
| **Discussions** | Settings → General → 启用 Discussions | ⬜ |
| **GitHub Pages** | Settings → Pages → main 分支 → /docs 目录 | ⬜ |

### 目录收录

| 目标 | 状态 | 链接 | 提交方式 |
|------|:----:|------|----------|
| **awesome-go** | ⬜ | https://github.com/avelino/awesome-go | PR 提交到 README.md |
| **MCP Servers 目录** | ⬜ | https://github.com/modelcontextprotocol/servers | PR 提交 |
| **awesome-mcp-servers** | ✅ 已有 | 用户自己的列表 | 已自维护 |
| **Go Weekly** | ⬜ | https://golangweekly.com/ | 邮件投稿 |

### 中文社区

| 平台 | 状态 | 链接 | 优先级 |
|------|:----:|------|:------:|
| **掘金** | 📝 文稿就绪 | https://juejin.cn | ⭐⭐⭐ |
| **知乎** | 📝 文稿就绪 | https://zhihu.com | ⭐⭐⭐ |
| **V2EX** | 📝 文稿就绪 | https://v2ex.com/go/go | ⭐⭐⭐ |
| **开源中国** | 📝 文稿就绪 | https://oschina.net | ⭐⭐ |
| **SegmentFault 思否** | 📝 文稿就绪 | https://segmentfault.com | ⭐⭐ |
| **CSDN** | 📝 文稿就绪（可复用掘金版） | https://csdn.net | ⭐ |

### 英文社区

| 渠道 | 状态 | 链接 | 优先级 |
|------|:----:|------|:------:|
| **Hacker News** | 📝 文稿就绪 | https://news.ycombinator.com/ | ⭐⭐⭐ |
| **Reddit r/golang** | 📝 文稿就绪 | https://reddit.com/r/golang | ⭐⭐⭐ |
| **Reddit r/LocalLLaMA** | 📝 文稿就绪 | https://reddit.com/r/LocalLLaMA | ⭐⭐ |
| **Twitter/X** | 📝 文稿就绪 | https://twitter.com | ⭐⭐ |

---

## 提交文案汇总

### awesome-go PR 文案

```markdown
* [cortex](https://github.com/lh123aa/cortex) - AI Agent knowledge base with MCP native support, single binary deployment, hybrid search and agent memory system.
```

### MCP Servers 目录 PR 文案

分类：Knowledge Base / Memory

```markdown
## Memory & Knowledge

- [Cortex](https://github.com/lh123aa/cortex) - Local knowledge base with hybrid search and agent memory system. Single binary, MCP native, 100% local.
```

### Go Weekly 投稿邮件模板

**Subject:** Cortex v2.2 – AI Agent Knowledge Base in Pure Go

**Body:**
> Cortex is a local knowledge base engine for AI Agents, written entirely in Go. Key highlights:
> - Single binary, zero dependencies
> - MCP protocol native (5 built-in tools)
> - Pure Go SQLite (modernc.org/sqlite, no CGO)
> - Hybrid search (HNSW vector + BM25 + RRF)
> - Built-in agent memory system
> - 114 unit tests
>
> GitHub: https://github.com/lh123aa/cortex

---

## 发布节奏建议

| 时间 | 渠道 | 备注 |
|:----:|------|------|
| Day 1 上午 | **V2EX** | 发帖后关注评论区互动 |
| Day 1 下午 | **掘金** | 审核通常几小时 |
| Day 2 | **知乎** | 隔一天发布，避免时间冲突 |
| Day 3 | **开源中国** | 提交项目，非文章 |
| Day 3 | **SegmentFault** | 发布技术文章 |
| Day 4 | **Hacker News** | Show HN（注意美国时区，建议 UTC 14:00） |
| Day 4 | **Reddit r/golang** | 可与 HN 同天发 |
| Day 5 | **Twitter/X** | 发帖+标签 |
| Week 2 | **awesome-go / MCP PR** | 等 Star 数上升后再提交，成功率更高 |
| Week 2 | **Go Weekly** | 邮件投稿 |

---

## 效果追踪

### KPI 目标

| 指标 | 当前 | 2周目标 | 1月目标 |
|:-----|:----:|:-------:|:-------:|
| GitHub Stars | 1 | 50 | 200+ |
| GitHub Forks | 0 | 5 | 20+ |
| 掘金阅读/点赞 | — | 2000+/50+ | 10000+/100+ |
| 知乎阅读/赞同 | — | 1000+/30+ | 5000+/80+ |
| 总文章曝光 | — | 5000+ | 30000+ |
| awesome-go 收录 | ❌ | — | ✅ |

### 追踪方式

- GitHub Insights → Traffic 查看流量来源
- 各平台阅读/点赞/评论数据
- Issue / PR 来源渠道分析

---

## 已创建的文件

| 文件 | 用途 |
|:-----|:------|
| `docs/promotion/PROMOTION_PLAN.md` | 整体推广方案 |
| `docs/promotion/juejin-article.md` | 掘金深度技术文章 |
| `docs/promotion/zhihu-article.md` | 知乎文章 |
| `docs/promotion/v2ex-post.md` | V2EX 推广帖 |
| `docs/promotion/oschina-submission.md` | 开源中国项目提交 |
| `docs/promotion/segmentfault-article.md` | SegmentFault 技术文章 |
| `docs/promotion/hackernews-post.md` | Hacker News / Reddit 英文帖 |
| `docs/promotion/twitter-thread.md` | Twitter/X 线程（中英双语） |
| `docs/promotion/demo-script.md` | Demo 脚本 |
| `.github/FUNDING.yml` | GitHub 赞助通道 |
