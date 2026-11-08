# DSH 插件生态竞品调研

> 调研日期：2026-11-08
> 数据来源：本地 DSH 插件市场索引缓存（`~/.dsh/profiles/desktop/.dsh-market/`）+ 各插件 GitHub README
> 目的：判断 Cortex 作为 DSH 插件是否有差异化空间

---

## 0. 结论摘要

| 问题 | 结论 |
|---|---|
| 市场里有竞品吗 | **有，而且很多**：97 个知识库/记忆类仓库 |
| 有赢家吗 | **没有**：抽样的头部竞品 star 数 **0–6** |
| 竞品有向量检索吗 | **没有**：最强的 Knowledge Center 自己写明「无向量嵌入，适合数百篇」 |
| 竞品有中文优化吗 | **基本没有**：仅个别提到中文，无 CJK 分词方案 |
| 竞品有独立数据库吗 | **没有**：全部基于文件（Markdown/JSON）或依赖外部服务 |
| Cortex 的差异化 | **大规模检索 + 中文 + 零外部依赖**，但这三点在 DSH 场景里**价值有限** |

**最终判断：差异化真实存在，但变现价值低。建议只做低成本 MCP 接入，不投入原生插件开发。**

---

## 1. 市场规模

| 指标 | 数值 |
|---|---|
| 插件市场总条目 | 4,871 |
| GitHub 来源插件 | 4,235 |
| 独立作者/组织 | 3,002 |
| **知识库/记忆类仓库** | **97** |

从命名看主题分布（按插件名匹配）：

| 主题 | 数量 |
|---|---|
| 效率工具 | 4,235（占绝大多数） |
| 知识库/RAG/记忆 | 169（插件条目）/ **97（独立仓库）** |
| 桌宠 | 142 |
| 机器人/IM | 121 |
| 模型/价格显示 | 118 |
| 皮肤/主题 | 109 |
| MCP 相关 | 51 |
| 游戏 | 19 |

---

## 2. 核心竞品抽样（实查数据）

| 插件 | Stars | 体积 | 技术路线 | 关键短板 |
|---|---:|---:|---|---|
| `lightporta/dsh-knowledge-center` | 1 | 406 KB | 调用 LLM 提炼工作区文档 → Markdown 知识库 | **自述「无向量嵌入，适合数百篇」**；依赖 LLM + LibreOffice |
| `andyfan1094/dsh-codebase-memory` | 6 | 87 KB | **桥接 MCP**（Codebase Memory 知识图谱） | 依赖外部 MCP server，非自研 |
| `Luke-Yong/dsh-plugin-knowledge-graph` | 4 | 42 KB | 知识图谱 | 体量小，功能未知 |
| `fangqiing/dsh-knowledge-lite` | 1 | 150 KB | 每篇 Markdown，全文搜索 + 标签 + 目录 | 文件级方案，无索引引擎 |
| `sd0426301x-dotcom/dsh-knowledge-base` | 1 | 33 KB | 本地可编辑知识库，侧边栏展示 | 体积最小，功能最薄 |
| `AllenWES365/dsh-local-memory` | 0 | 999 KB | 项目隔离长期记忆，**依赖 Hindsight daemon** | 必须装外部服务；实测大库 1.0–1.4 s |
| `ICCuse/dsh-knowledge` | 0 | 62 KB | 桥接用户全局 Markdown 知识库 | 依赖 Python CLI |
| `135642-163/dsh-knowledge-base` | 0 | 2.8 MB | 未知 | 无描述 |

**观察：没有一个是「自带检索引擎」的实现。**

---

## 3. 关键证据：竞品自述的能力边界

### 3.1 Knowledge Center（功能最完整的竞品）

原文（中文版）：

> 检索为本地关键词 + 摘要匹配，**没有向量嵌入**；适合数百篇规模，不适合大规模语义检索。

原文（英文版）：

> Retrieval is local keyword/summary matching with **no vector embeddings**; suitable for hundreds of notes, not for large-scale semantic search.

其他自述限制：

> 单篇超长文档只提炼前约 192k 字符，会在 front matter 中标记 `truncated: true`

> Office 需要 LibreOffice 服务；复杂版式（分栏、表格合并、图表）只保留文本顺序，不保留视觉布局

### 3.2 Local Memory

依赖外部 Hindsight daemon，实测性能：

| 规模 | 延迟 |
|---|---|
| 小库（几条事实） | ~30 ms |
| 大库（数百条事实、数万条关系边） | **~1.0–1.4 s** |

作者自己的结论：

> **客户端没有任何参数能改善单库成本**——实测：关重排 1035ms vs 基线 1030ms；`budget: low` 反而更慢（1255ms）

### 3.3 外部依赖汇总

| 插件 | 外部依赖 |
|---|---|
| Knowledge Center | LLM 调用 + LibreOffice + Node |
| Local Memory | Hindsight daemon |
| Knowledge Lite | Node |
| Codebase Memory | 外部 MCP server + Node |
| ICCuse Knowledge | Python |

**没有一个能做到「零外部依赖」。**

---

## 4. Cortex 的真实差异点

| 差异点 | Cortex | 竞品最好水平 | 差异价值 |
|---|---|---|---|
| **检索规模** | 27,157 文档实测 P95 **2.35 ms** | 「适合数百篇」 | ⭐⭐⭐ 真实但场景不匹配 |
| **向量检索** | ✅ 支持（7 家 provider） | ❌ 全部无 | ⭐⭐⭐ 真实差异 |
| **中文分词** | ✅ CJK 专用 + 标题/正文分离 | 无专门方案 | ⭐⭐ 对中文用户有意义 |
| **零外部依赖** | ✅ 单二进制 + SQLite | 全部需要 Node/LibreOffice/daemon | ⭐⭐⭐ 真实差异 |
| **代码符号检索** | ✅ Go AST 精确 | 部分有，多依赖外部 MCP | ⭐⭐ |
| **MCP 原生** | ✅ 14 工具 | 部分通过 mcp-client 桥接 | ⭐ 无差异（大家都支持） |
| **Agent 记忆** | ✅ 独立记忆 API | 多数有 | ⭐ 无差异 |

### 4.1 但差异的价值被打折

| 打折原因 | 说明 |
|---|---|
| DSH 用户的实际语料规模 | 插件生态主要服务**单个工作区**，几百个文件，用不上 2.7 万文档的检索能力 |
| 「提炼」比「检索」更被需要 | 头部竞品做的是**LLM 提炼成知识**，而不是原始文档检索。用户要的是「看懂」不是「搜到」 |
| 单二进制是劣势 | 竞品 `pnpm add` 即用；Cortex 要额外装 36 MB 二进制 + 索引步骤 |
| 装插件的人 ≠ 会配索引的人 | DSH 插件用户预期「装上就能用」，Cortex 需要 `cortex index` 前置步骤 |

---

## 5. 竞品的共同产品逻辑（值得学习的部分）

最成功的竞品 **不是**在做「检索」，而是在做**「把工作区文档变成可复用的知识」**：

```
散落的 MD/TXT/PDF  →  LLM 提炼  →  结构化知识条目  →  侧边栏可读可编辑  →  Agent 可调用
```

这个链条里，价值最高的环节是**「提炼」**（省 token、省理解成本），而不是「检索」（Cortex 强项）。

同时它们都做了 Cortex 没做的事：

| 能力 | 竞品 | Cortex |
|---|---|---|
| 侧边栏 UI 面板 | ✅ Knowledge Center 有完整面板 | ❌ 只有 `/admin` Web 控制台 |
| 知识条目可编辑 | ✅ | ❌ 只读 |
| 提炼结果预览（类 VS Code） | ✅ | ❌ |
| 按工作区授权 | ✅ | ⚠️ 有 RBAC 但非工作区粒度 |
| 后台自动提炼 | ✅ | ❌ |

---

## 6. 最终判断

### 6.1 是否有差异化空间

**技术上有，产品上没有。**

- 技术上：Cortex 的检索能力确实远超所有竞品（它们连向量检索都没有）
- 产品上：DSH 插件用户的核心需求是「轻量、即装即用、帮我提炼」，不是「大规模检索」

### 6.2 如果非要切入，正确的形态是什么

不是「把 Cortex 做成插件」，而是**「把 Cortex 变成插件的后端」**：

```
DSH 插件（薄，200 行 JS）
  ├─ 侧边栏 UI（对标 Knowledge Center）
  ├─ 调用 LLM 提炼文档 → 写入 Cortex
  └─ 检索时调用 Cortex REST API
                ↓
        Cortex 二进制（重，负责索引/检索/存储）
```

这样：
- 插件解决了「即装即用」（`pnpm add`）
- Cortex 提供了竞品没有的**大规模检索 + 中文 + 向量**
- 而且**插件可以只做免费引流，检索能力绑定付费版**（Pro 以上才支持 >5000 文档）

### 6.3 但也必须承认

**这个方案的工作量是 3–4 周，而市场头部竞品只有 6 个 star。**

投入产出比不成立。合理顺序仍是：

1. 先验证 Cortex 本体有用户
2. 如果 DSH 生态真的起来（某个竞品 star 过百），再跟进
3. 现在最多做 **MCP 接入（0.5 天）**，占个位置，不投入

---

## 7. 值得记录的三条洞察

1. **「没有向量检索」是整条赛道的空白**，但这不是技术问题，是需求问题——用户还没大到需要它。
2. **竞品的核心竞争力是「LLM 提炼」而不是「检索」**，这是 Cortex 的产品盲区。
3. **97 个竞品、头部 6 star** 说明这是一个「供给过剩、需求未验证」的市场。此刻投入是赌博。

---

## 附：数据复现方式

```bash
# 市场索引缓存位置
~/.dsh/profiles/desktop/.dsh-market/discovery-github-compatibility-v1.json
~/.dsh/profiles/desktop/.dsh-market/discovery-compatibility-v1.json

# 统计知识库类插件
cat <索引文件> | jq -r '.entries | keys[]' | grep -Ei 'knowledge|rag-|memory|recall|vector' | wc -l

# 查竞品 README
curl -s https://raw.githubusercontent.com/<owner>/<repo>/main/README.md
```
