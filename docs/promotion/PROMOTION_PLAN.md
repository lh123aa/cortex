# Cortex 推广方案

> 目标：提升 GitHub 仓库曝光度，吸引中文开发者社区关注
> 日期：2026-05-06

---

## 目录

- [一、GitHub 站内优化](#一github-站内优化)
- [二、中文技术社区推广](#二中文技术社区推广)
- [三、英文渠道推广](#三英文渠道推广)
- [四、执行排期](#四执行排期)
- [五、效果追踪](#五效果追踪)

---

## 一、GitHub 站内优化

### 1.1 仓库基础设置（需手动操作）

**Repo Description 建议值：**
> Cortex - AI Agent 的第二大脑。单二进制本地知识库，MCP 原生支持，混合搜索 + Agent 记忆系统。

**建议 Topics（至多 20 个）：**
```
go, ai-agent, mcp, model-context-protocol, knowledge-base, rag, vector-search, hnsw, bm25, sqlite, memory-system, agent-memory, local-ai, llm, embedding, hacktoberfest, golang, prompt-engineering, semantic-search, information-retrieval
```

**建议 Website URL：**
```
https://github.com/lh123aa/cortex
（或即将上线的 GitHub Pages: https://lh123aa.github.io/cortex/）
```

### 1.2 需要使用独立文件创建的内容

| 文件 | 状态 | 说明 |
|------|------|------|
| `.github/FUNDING.yml` | ⬜ 待创建 | 赞助通道（GitHub Sponsors / 微信 / 支付宝） |
| `.github/workflows/go-report.yml` | ⬜ 待创建 | Go Report Card CI 自动触发 |
| `docs/promotion/WEBSITE_TEMPLATE.md` | ⬜ 待创建 | GitHub Pages 网站模板 |

### 1.3 README 优化清单

- [x] 产品对比表格（已有，效果很好）
- [x] 徽章（已有）
- [ ] 添加 CI/CD 构建状态徽章
- [ ] 添加 Go Report Card 徽章
- [ ] 添加 Go Reference 文档徽章
- [ ] 添加聊天/讨论群二维码（如果有）
- [ ] 优化标题 SEO 关键词密度
- [ ] 添加使用案例/截图/GIF Demo

### 1.4 GitHub Pages 项目网站

用 GitHub Pages 搭建一个简单的产品展示页面，包含：
- 首页 Hero + 核心卖点
- 安装指南
- 特性展示
- 快速开始
- 链接回 GitHub

---

## 二、中文技术社区推广

### 2.1 文章发布矩阵

| 平台 | 适合内容 | 优先级 | 状态 |
|------|---------|--------|------|
| **掘金 (juejin.cn)** | 深度技术文章，带代码示例和架构图 | ⭐⭐⭐ | ⬜ |
| **知乎 (zhihu.com)** | 长文评测，技术选型思路分享 | ⭐⭐⭐ | ⬜ |
| **V2EX** | 简洁分享帖，带核心亮点和对比 | ⭐⭐⭐ | ⬜ |
| **开源中国 (oschina.net)** | 项目推荐，新闻向 | ⭐⭐ | ⬜ |
| **SegmentFault 思否** | 技术教程向 | ⭐⭐ | ⬜ |
| **CSDN** | 技术教程（备选，流量大但质量参差） | ⭐ | ⬜ |

### 2.2 文章标题策略

预设 3 个方向的标题，A/B 测试：

**方向 A — 痛点驱动型：**
> 🧠 我写了个 AI Agent 的"第二大脑"：单二进制 + MCP 原生 + 零依赖，给 Claude/OpenCode 装上永久记忆

**方向 B — 技术实战型：**
> 从零搭建 AI Agent 记忆系统：只用 1 个二进制文件搞定 RAG + 混合搜索 + MCP 协议

**方向 C — 对比评测型：**
> 对比 6 款 AI Agent 知识库工具后，我用 Go 自己写了一个：Cortex v2.2

### 2.3 文章结构模板

```
1. 痛点引入（Agent 没有记忆的问题）
2. 现有方案对比（Mem0 / ChromaDB / Dify 等）
3. Cortex 介绍 + 核心架构
4. 快速上手指南（带命令行演示）
5. 技术选型分析（为什么用 Go / SQLite / HNSW）
6. MCP 协议集成（与 Claude Code 等 Agent 配合）
7. 性能数据
8. 开源 & 未来规划
9. 结语 + Call to Action
```

### 2.4 推广节奏

```
Day 1: V2EX + 掘金 同时发布
Day 2: 知乎 发布文章
Day 3: OSCHINA 提交项目
Day 5: SegmentFault 发布
Day 7: 汇总数据，复盘优化
```

---

## 三、英文渠道推广

### 3.1 目标渠道

| 渠道 | 内容形式 | 优先级 |
|------|---------|--------|
| Hacker News | Show HN 帖子 | ⭐⭐⭐ |
| Reddit r/golang | 技术分享帖 | ⭐⭐⭐ |
| Reddit r/LocalLLaMA | AI 工具分享 | ⭐⭐ |
| awesome-go | PR 提交 | ⭐⭐⭐ |
| MCP Servers 目录 | PR 提交 | ⭐⭐⭐ |
| Go Weekly | 投稿 | ⭐⭐ |

### 3.2 现有资源

- `hackernews-post.md` — 已有 Hacker News 投稿文案 ✅
- `twitter-thread.md` — 已有 Twitter 线程 ✅
- `submission-tracker.md` — 已有提交追踪清单 ✅

### 3.3 需要完善

- [ ] awesome-go PR 文案优化
- [ ] Go Weekly 投稿邮件模板
- [ ] 英文 Demo GIF/截图

---

## 四、执行排期

### Week 1 — 基础优化 + 内容准备

| 日期 | 任务 |
|------|------|
| Day 1 | GitHub 仓库设置（description / topics / website） |
| Day 1 | 创建 FUNDING.yml |
| Day 1 | 优化 README 徽章和 SEO |
| Day 2 | 撰写并发布掘金文章 |
| Day 2 | 撰写并发布 V2EX 帖子 |
| Day 3 | 撰写并发布知乎文章 |
| Day 3 | OSCHINA 提交项目 |
| Day 4 | 制作 Demo GIF/截图 |
| Day 5 | SegmentFault 发布 |
| Day 5 | 英文渠道：Reddit + HN 准备 |

### Week 2 — 社区互动 + 渠道收录

| 日期 | 任务 |
|------|------|
| Day 8-10 | 回复各平台评论 |
| Day 8-10 | 提交 awesome-go / MCP Servers PR |
| Day 11-12 | 数据复盘，调整策略 |
| Day 13-14 | 迭代发布 v2.3，带出二次传播 |

---

## 五、效果追踪

### 关键指标 (KPI)

| 指标 | 当前 | 2周目标 | 1月目标 |
|------|------|---------|---------|
| GitHub Stars | 1 | 50 | 200+ |
| GitHub Forks | 0 | 5 | 20+ |
| 掘金阅读/点赞 | — | 2000+/50+ | 10000+/100+ |
| 知乎阅读/赞同 | — | 1000+/30+ | 5000+/80+ |
| 总文章曝光 | — | 5000+ | 30000+ |
| awesome-go 收录 | ❌ | ✅ | ✅ |

### 追踪方式

- GitHub Insights → Traffic 查看 Referral 来源
- 各平台阅读/点赞/评论数据
- Issue / PR 来自哪些渠道

---

## 六、需要你手动操作的部分

以下操作需要你在 GitHub 网页上完成：

1. **设置 Repo Description** → 仓库页面顶部 "About" 区域点击编辑
2. **添加 Topics** → 仓库页面顶部输入 Topics
3. **设置 Website URL** → 同上 About 区域
4. **开启 GitHub Pages** → Settings → Pages → 选择 main 分支 /docs 文件夹
5. **创建 Release** → 如果还没有，创建一个正式的初始 Release
6. **启用 Discussions** → Settings → General → Discussions
7. **关注 Issue/PR 回复** → 及时回复社区反馈

---

*本文件由 AI 自动生成，建议根据实际情况调整。*
