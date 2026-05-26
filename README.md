<p align="center">
  <img src="docs/assets/img/cortex-logo-static.svg" width="120" height="120" alt="Cortex Logo">
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=for-the-badge&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/License-MIT-green?style=for-the-badge" alt="MIT">
  <img src="https://img.shields.io/badge/Version-3.5-blue?style=for-the-badge" alt="v3.5">
  <img src="https://goreportcard.com/badge/github.com/lh123aa/cortex?style=for-the-badge" alt="Go Report Card">
  <img src="https://img.shields.io/badge/Tests-11_11_packages-green?style=for-the-badge" alt="Tests">
  <img src="https://img.shields.io/badge/Iteration-Complete-7B61FF?style=for-the-badge" alt="Iteration Complete">
  <img src="https://img.shields.io/badge/Bug-0-success?style=for-the-badge" alt="Bug 0">
  <img src="https://img.shields.io/badge/MCP-8_Tools-7B61FF?style=for-the-badge" alt="MCP 8 Tools">
  <img src="https://img.shields.io/badge/Streamable_HTTP-SSE-7B61FF?style=for-the-badge" alt="SSE">
  <img src="https://img.shields.io/badge/Auto_Register-Trae/Claude/Cursor-7B61FF?style=for-the-badge" alt="Auto Register">
  <img src="https://img.shields.io/github/actions/workflow/status/lh123aa/cortex/build.yml?style=for-the-badge&logo=github" alt="Build">
  <img src="https://img.shields.io/github/stars/lh123aa/cortex?style=for-the-badge&logo=github" alt="Stars">
</p>

<h1 align="center">🧠 Cortex</h1>
<h3 align="center">AI Agent 的第二大脑 — 本地知识库 · 单二进制 · MCP 原生</h3>

<p align="center">
  <b>Cortex</b> 是一个为 AI Agent 设计的本地知识库引擎。单二进制文件部署，原生支持 <b>MCP 协议</b>，内置<b>混合搜索</b>（向量+BM25）和<b>Agent 记忆系统</b>。100% 本地运行，零外部依赖。<br>
  <code>cortex start</code> — 一条命令同时启动 MCP + REST API，自动适配所有环境 🚀
</p>
<p align="center">
  给 Claude Code、OpenCode 等 AI Agent 装上永久记忆 🧠
</p>
<p align="center">
  <a href="https://github.com/lh123aa/cortex"><b>GitHub 仓库</b></a> ·
  <a href="https://lh123aa.github.io/cortex/"><b>🎨 营销落地页</b></a> ·
  <a href="docs/index.html"><b>本地预览</b></a> ·
  <a href="docs/promotion/juejin-article.md"><b>📖 深度技术文章</b></a>
</p>

<p align="center">
  <a href="docs/index.html">
    <img src="docs/assets/img/terminal-demo.svg" width="100%" alt="Terminal Demo" style="max-width:720px;border-radius:16px;border:1px solid rgba(123,97,255,0.15);">
  </a>
  <br>
  <sub>⬆️ 点击查看完整营销落地页（小米/Mimo 风格）</sub>
</p>

<p align="center">
  <a href="#-产品对比">📊 产品对比</a> ·
  <a href="#-快速开始">⚡ 快速开始</a> ·
  <a href="#-核心特性">✨ 核心特性</a> ·
  <a href="#-系统架构">🏗️ 系统架构</a> ·
  <a href="#-rest-api">📡 API</a> ·
  <a href="#-配置">🔧 配置</a>
  <br>
  <a href="./README.en.md">🌐 English Version</a> ·
  <a href="./docs/TUTORIAL.md">📖 实战教程</a> ·
  <a href="#-开发">🛠️ 开发</a> ·
  <a href="./TEST_REPORT.md">📋 测试报告</a> ·
  <a href="./ITERATION_PLAN.md">📋 迭代计划</a> ·
  <a href="./SYSTEM_EVALUATION.md">📋 评估报告</a> ·
  <a href="./docs/grafana-dashboard.json">📊 Grafana</a>
</p>

---

## 📊 产品对比

<table>
<thead>
<tr>
  <th>功能</th>
  <th align="center">🚀 Cortex</th>
  <th align="center">Mem0</th>
  <th align="center">AnythingLLM</th>
  <th align="center">ChromaDB</th>
  <th align="center">Qdrant</th>
  <th align="center">Dify</th>
</tr>
</thead>
<tbody>
<tr>
  <td colspan="7"><strong>📦 部署</strong></td>
</tr>
<tr>
  <td>部署方式</td>
  <td align="center">✅ <strong>单二进制</strong><br><sub>下载即用</sub></td>
  <td align="center">⚠️ pip/Docker<br><sub>需 Python 环境</sub></td>
  <td align="center">⚠️ Docker/Desktop<br><sub>需 Node.js</sub></td>
  <td align="center">⚠️ pip/Docker<br><sub>需 Python</sub></td>
  <td align="center">✅ <strong>单二进制</strong><br><sub>下载即用</sub></td>
  <td align="center">⚠️ Docker Compose<br><sub>多服务部署</sub></td>
</tr>
<tr>
  <td>外部依赖</td>
  <td align="center">✅ <strong>零依赖</strong><br><sub>可选 Ollama</sub></td>
  <td align="center">❌ 需 LLM API</td>
  <td align="center">❌ 需 LLM API</td>
  <td align="center">✅ 无</td>
  <td align="center">✅ 无</td>
  <td align="center">❌ 多服务</td>
</tr>
<tr>
  <td colspan="7"><strong>🤖 AI Agent 集成</strong></td>
</tr>
<tr>
  <td>MCP 协议原生</td>
  <td align="center">✅ <strong>原生支持</strong><br><sub>cortex mcp</sub></td>
  <td align="center">⚠️ 插件集成</td>
  <td align="center">✅ 支持</td>
  <td align="center">❌ 不支持</td>
  <td align="center">❌ 不支持</td>
  <td align="center">✅ 支持</td>
</tr>
<tr>
  <td>MCP 工具数</td>
  <td align="center">🔧 <strong>8 个</strong><br><sub>搜索/上下文/记忆/健康/建议</sub></td>
  <td align="center">🔧 1-2 个</td>
  <td align="center">🔧 1 个</td>
  <td align="center">—</td>
  <td align="center">—</td>
  <td align="center">🔧 1-2 个</td>
</tr>
<tr>
  <td>Agent 记忆系统</td>
  <td align="center">✅ <strong>内置</strong><br><sub>长期记忆+RAG</sub></td>
  <td align="center">✅ <strong>专注</strong><br><sub>多层记忆</sub></td>
  <td align="center">❌ 仅对话</td>
  <td align="center">❌ 向量库</td>
  <td align="center">❌ 向量库</td>
  <td align="center">⚠️ 基础记忆</td>
</tr>
<tr>
  <td colspan="7"><strong>🔍 搜索</strong></td>
</tr>
<tr>
  <td>搜索类型</td>
  <td align="center">✅ <strong>混合搜索</strong><br><sub>向量+BM25+RRF</sub></td>
  <td align="center">✅ 混合搜索</td>
  <td align="center">✅ 向量搜索</td>
  <td align="center">✅ 向量/混合</td>
  <td align="center">✅ 混合搜索</td>
  <td align="center">⚠️ 依赖后端</td>
</tr>
<tr>
  <td>文件格式</td>
  <td align="center">📄 <strong>MD/PDF/DOCX</strong><br><sub>+代码文件</sub></td>
  <td align="center">— <sub>纯记忆</sub></td>
  <td align="center">✅ <strong>多格式</strong></td>
  <td align="center">— <sub>纯向量</sub></td>
  <td align="center">— <sub>纯向量</sub></td>
  <td align="center">✅ 多格式</td>
</tr>
<tr>
  <td colspan="7"><strong>📊 运维</strong></td>
</tr>
<tr>
  <td>内置监控</td>
  <td align="center">✅ <strong>Prometheus</strong><br><sub>39 指标</sub></td>
  <td align="center">⚠️ Dashboard</td>
  <td align="center">⚠️ 基础</td>
  <td align="center">❌ 无</td>
  <td align="center">❌ 无</td>
  <td align="center">✅ Grafana</td>
</tr>
<tr>
  <td>缓存加速</td>
  <td align="center">✅ <strong>L1+L2 两级</strong><br><sub>内存+SQLite</sub></td>
  <td align="center">⚠️ 基础缓存</td>
  <td align="center">⚠️ 基础缓存</td>
  <td align="center">❌ 无</td>
  <td align="center">✅ 内存</td>
  <td align="center">⚠️ 基础</td>
</tr>
<tr>
  <td>隐私保护</td>
  <td align="center">✅ <strong>完全本地</strong><br><sub>100% 离线</sub></td>
  <td align="center">⚠️ 本地/云</td>
  <td align="center">✅ 完全本地</td>
  <td align="center">✅ 完全本地</td>
  <td align="center">✅ 完全本地</td>
  <td align="center">⚠️ 本地/云</td>
</tr>
<tr>
  <td>开源协议</td>
  <td align="center">✅ <strong>MIT</strong><br><sub>商用自由</sub></td>
  <td align="center">✅ Apache 2.0</td>
  <td align="center">✅ MIT</td>
  <td align="center">✅ Apache 2.0</td>
  <td align="center">✅ Apache 2.0</td>
  <td align="center">⚠️ 附加条款</td>
</tr>
</tbody>
</table>

> **Cortex 的核心差异化**：它是唯一一个同时具备「单二进制部署 + MCP 原生 + 内置记忆系统 + 混合搜索 + Prometheus 监控」的工具，专为 AI Agent 场景设计。

---

## 🎯 适用场景

| 场景 | 说明 |
|------|------|
| 🤖 **AI Agent 记忆** | 让 Claude Code / OpenCode / Cursor 等 Agent 跨会话记住用户偏好和历史 |
| 📚 **团队知识库** | 将团队 Wiki、技术文档、项目规范索引为可搜索的 RAG 知识库 |
| 🔍 **代码库语义搜索** | 索引 Go/Python/JS 等代码，用自然语言搜索函数、类和实现 |
| 🏢 **企业内部文档** | 员工手册、产品文档、培训材料的本地私密检索，数据不出内网 |
| 🧪 **RAG 应用后端** | 作为 RAG pipeline 的检索层，提供 REST API 和 MCP 双协议接入 |
| 🔐 **隐私敏感场景** | 金融、医疗、法律等需要 100% 本地部署的知识管理需求 |

---

## ✨ 更新日志

### 🚀 v3.5.3 开箱即用改造 (2026-05-26)

- ✅ **`cortex start` 命令** — 一条命令同时启动 MCP + REST API，自动适配所有环境
- ✅ **端口自动检测** — 端口冲突时自动递增查找可用端口（2021→2022→...）
- ✅ **AI 工具自动注册** — 检测并注册到 Claude Code / Cursor / OpenCode / Trae，原配置自动 `.bak` 备份
- ✅ **启动面板** — 启动后打印 REST URL、已注册工具、文档统计、FTS 提示
- ✅ **MCP over SSE** — 新增 `POST /mcp` 端点，任何 HTTP 客户端均可调用 MCP Tool
- ✅ **搜索 snippet 截断** — CLI + REST API 输出统一 300 字符限制
- ✅ **文件类型权重** — 搜索结果评分加入文件类型权重（`.md` > 代码 > `.txt` > `.html/.css`）
- ✅ **默认 top_k 20** — 从 10 增加到 20，减少漏召回
- ✅ **FTS 搜索提示** — CLI 和 REST API 响应中标注 FTS-only 模式提示

### 🧪 v3.5.2 回归测试 + 内存优化 (2026-05-19)

- ✅ **5 个修复点全覆盖回归测试** — 11 个新测试 + 2 个已有，CI 22 包全过
- ✅ **内存效率⭐⭐→⭐⭐⭐⭐⭐** — 无 embedding 时 446MB→**19.6MB**（-95.6%）
- ✅ **索引持久化二进制化** — JSON 260MB→**binary 80MB**（-69%），加载 9.6s→~1s
- ✅ **扁平向量存储** — `[][]float32`→`[]float32`，去掉 23K 个 slice header
- ✅ **条件索引加载** — 无 embedding provider 时完全跳过（`initStorage` 新增 `buildIndex` 参数）
- ✅ **REST API htptest 集成测试** — /health、/v1/search 端点回归覆盖
- ✅ **Health tool 状态显示回归测试** — 有/无 embedding 两种场景

### 🐛 v3.5.1 REST API 修复 + HNSW 替换 + 向量索引持久化 (2026-05-19)

- ✅ **`cortex serve` 修复** — REST API 现在正确监听 `:8080`，/health、/v1/search 等端点可用
- ✅ **HNSW 替换** — 解决 2 万+ 向量规模下的构建崩溃，替换为轻量级内存向量索引
- ✅ **向量索引持久化** — 构建完成后写入 `cortex.db_flat_idx.json`，下次启动秒加载（3 秒读 23K 向量）
- ✅ **`cortex_health` MCP 工具显示修复** — 正确显示 embedding 状态而非硬编码 `none (FTS5)`
- ✅ **HNSW 构建性能优化** — EfConstruction 200→80，构建速度提升 2.5x
- ✅ **StorageBridge 进度回调** — 后台索引构建可追踪进度
- ✅ **回归测试** — 11/11 包通过，`go vet` 零警告

### 🚀 v3.5 测试加固 + 代码重构 (2026-05-14)

- ✅ **Phase C 测试加固** — models/storage/index/search/embedding 五模块测试强化
- ✅ **Phase A main.go 拆分** — 1000 行 → 7 行，拆分到 internal/cli/ (11 文件)
- ✅ **`cortex_suggest` MCP 工具** — 第 8 个 MCP 工具，预联想搜索建议
- ✅ **`GET /v1/suggest` REST 端点** — 基于 prefetch 缓存的即时搜索建议
- ✅ **Prefetch 文档同步** — cortex watch 自动启用 prefetch
- ✅ **CI 修复** — `go vet` 零警告，CRUD 测试重复声明修复
- ✅ **回归测试** — build/vet/test 全部通过

### 🚀 v3.4 Prefetch 预联想搜索引擎 (2026-05-14)

- ✅ **`PrefetchEngine` 预联想** — 文件变更时自动提取关键词并异步预搜索
- ✅ **智能关键词提取** — Markdown 标题 + 高频词自动识别
- ✅ **独立预取缓存** — 搜索时优先命中缓存，输入一半即出结果
- ✅ **`cortex watch` 集成** — 搭配 `search.prefetch: true` 启用
- ✅ **配置项** — `SearchConfig.Prefetch`（默认 false）
- ✅ **单元测试** — 关键词提取/缓存 key/引擎初始化 3 个测试用例

### 🐛 v3.3.1 Bug 修复 + 安全加固 (2026-05-13)

- ✅ **`cortex version` 命令** — 新增版本号查询
- ✅ **HTTP 服务器安全加固** — `ReadTimeout`/`WriteTimeout`/`ReadHeaderTimeout`/`MaxHeaderBytes`
- ✅ **channel 双 close 防护** — `StopAutoBackup()`/`watcher.Stop()` 使用 `sync.Once` 防 panic
- ✅ **文件监听内存泄漏修复** — `debounceMap` 定时清理过期条目
- ✅ **中文搜索健壮性** — `expandChineseQuery` 改用 `strings.HasSuffix` 代替字节索引
- ✅ **回归测试** — 10/10 包通过，`go vet` 零警告

### 🚀 v3.3 中文搜索修复 + 文件监听 + 性能优化 (2026-05-13)

- ✅ **中文搜索修复** — 从 2-gram 改为单字分词，FTS5 中文搜索正常匹配
- ✅ **文件排除规则** — 自动跳过 50+ 种二进制/多媒体扩展名，索引失败降 **95%**，提速 **3.5x**
- ✅ **`cortex watch` 命令** — 基于 fsnotify 的文件监听，变更自动增量索引
- ✅ **`cortex_health` MCP 工具** — 第 7 个 MCP 工具，健康检测 + 文档统计
- ✅ **搜索结果展示优化** — `Content` 存原文（无空格），标题用文件名代替"section"
- ✅ **PDF/DOCX 中文搜索** — PDF 和 Word 文件添加中文分词支持
- ✅ **配置热加载** — `cortex serve` 修改 config.yaml 自动生效，无需重启
- ✅ **配置向导默认认证** — `cortex setup` 首次配置自动开启 `auth_enabled: true`
- ✅ **自动定时备份** — 24h 间隔自动备份，保留最近 10 份
- ✅ **全量重建** — `--force` 标志跳过内容哈希，新分词/新规则即时生效

- ✅ **断点续传修复** — 中断后状态保持 `running`，下次运行自动恢复（修复：之前中断会错误标记为 completed）
- ✅ **实时进度条** — `cortex index` 显示动态进度条 `[████░░] 45%`，含速度、ETA、当前文件
- ✅ **Ctrl+C 安全退出** — 信号处理保存进度后优雅退出，不丢失已索引数据
- ✅ **全局超时** — `cortex index --timeout 30m` 超时自动保存进度退出
- ✅ **单文件超时** — Embedding 调用 5 分钟超时保护，防止卡死
- ✅ **`--force` 重新索引** — `cortex index --force path` 从头开始（忽略 checkpoint）
- ✅ **`--workers` 控制并发** — `cortex index --workers 32` 覆盖默认 16 个 worker
- ✅ **默认 16 workers** — I/O 密集型索引任务，吞吐量提升

### 🔥 v3.0.1 启动性能优化 (2026-05-07)

- ✅ **HNSW 按需加载** — `initStorageLight()` 跳过向量索引构建，所有 CLI 命令启动 <0.5s
- ✅ **默认切到 FTS5-only 模式** — `embedding.provider: none`，索引速度提升 28x，零外部依赖
- ✅ **根目录清理** — 从 39 项精简至 27 项，规划文档移入 `docs/planning/`，部署配置移入 `deploy/`，编译产物移入 `bin/`
- ✅ **文件组织规则** — 全局 AGENTS.md 新增 6 条强制规则，涵盖根目录守则、构建产出、临时文件、AI 文件生成

### 🚀 v3.1 多 Provider + 配置向导 (2026-05-07)

- ✅ **`cortex setup` 交互式向导** — 网络检测 → 选择 Provider → 输入 API Key → 选择模型 → 测试连接 → 写入配置
- ✅ **9 个 Embedding Provider** — 本地: none / ollama / onnx；国际: OpenAI / Cohere / Voyage AI；国内: 阿里DashScope / 智谱GLM / 百度ERNIE
- ✅ **工厂模式** — `NewProviderFromConfig()` 根据配置自动创建对应 Provider，无需修改 main.go
- ✅ **百度 OAuth2** — 自动缓存 access_token（30天有效期），自动刷新
- ✅ **默认模型 all-minilm** — 23MB，比 nomic-embed-text 快 30x
- ✅ **自动网络检测** — 离线时自动隐藏 API 选项，仅显示本地模式

### 💰 v3.0 商业化版本 (2026-05-06)

- ✅ **套餐系统** — Free (≤1GB) / Pro / Enterprise 三级
- ✅ **存储追踪** — `cortex usage` 查看用量和套餐
- ✅ **License Key** — 许可证生成/激活/验证系统
- ✅ **中文分词优化** — CJK 2-gram 分词器，搜索质量提升
- ✅ **搜索结果高亮** — CLI 匹配关键词用颜色标记
- ✅ **Web Admin 记忆管理** — 管理界面支持写入和搜索记忆
- ✅ **启动无警告** — `PRAGMA table_info` 智能迁移
- ✅ **测试覆盖** — 10/10 包通过，新增 storage/index 测试

### 🚀 v2.4 全面迭代完成 (2026-05-06)

- ✅ **三层去重体系** — 内容哈希 + 向量语义 (-46 chunks) + MinHash 近似去重
- ✅ **6 个 MCP 工具** — 新增 `cortex_memory_delete_batch` 批量删除
- ✅ **11 个 Storage 子接口** — DocumentStore / Searcher / CacheStore / UserStore 等
- ✅ **索引排除规则** — 自动跳过 `node_modules/.git/.opencode/` 等噪声目录
- ✅ **Web 管理界面** — `/admin` 嵌入式单页 (Go embed)，已接线可用
- ✅ **Docker 多阶段构建** — 镜像从 500MB → 30MB
- ✅ **CLI JSON 输出** — `--json` 标志支持结构化输出
- ✅ **MCP 错误码标准化** — 统一 `IsError` 协议响应
- ✅ **暴力破解防护** — `/auth` 端点 IP 级令牌桶限流 (5 req/s)
- ✅ **二进制优化** — `-ldflags="-s -w"` 压缩至 **34.2 MB**
- ✅ **分页搜索 / Grafana 监控模板 / Release 自动化 / 54 项迭代**
- ✅ 详见 [SYSTEM_EVALUATION.md](./SYSTEM_EVALUATION.md)

### 🔥 v2.3 迭代升级 (2026-05-06)

- ✅ **7 个 P0 紧急 Bug 修复** — Float32FromBytes 位转换、索引进度竞争、HNSW 并发、API Key 泄露、Token 过期等
- ✅ **10 项安全加固** — MCP 参数校验、密码策略增强、BM25 评分修正、ConfigWatcher 热加载、路径规范化
- ✅ **12 项质量提升** — 缓存 LRU 改造、余弦相似度收敛、RRF k 值配置化、死代码清理、Prometheus 自定义 Registry
- ✅ **CI/CD 全面升级** — Go 1.25 同步、govulncheck、测试覆盖率、Makefile 增强、VSCode 调试配置
- ✅ **29 项优化总计** — 详见 [TEST_REPORT.md](./TEST_REPORT.md) 和 [ITERATION_PLAN.md](./ITERATION_PLAN.md)

### 🧠 v2.2 MCP 记忆工具 (2026-05-05)

- ✅ **5 个 MCP 工具** — `cortex_search` / `cortex_context` / `cortex_memory_write` / `cortex_memory_search` / `cortex_memory_delete`
- ✅ **嵌入式零依赖模式** — `embedding.provider: none`，无需 Ollama，FTS5 全文搜索
- ✅ **纯 Go SQLite 驱动** — 改用 `modernc.org/sqlite`，无需 CGO/gcc，`go build` 即用
- ✅ **MCP 优雅关闭** — Signal 处理，Ctrl+C 安全退出
- ✅ **MCP 单元测试** — 11 个测试用例，覆盖全部工具边界条件

### 🔥 v2.1 生产环境修复 (2026-04-25)

- ✅ **L1+L2 两级缓存** — 内存 + SQLite 缓存，搜索速度提升 10x
- ✅ **Graceful Shutdown** — 优雅关闭，30s 内处理完现有请求
- ✅ **请求超时控制** — 默认 30s，搜索 60s，索引 5min
- ✅ **API 限流** — 令牌桶算法，100 req/s，突发 200
- ✅ **36 个测试用例** — 覆盖存储/认证/搜索核心模块

### ✨ v2.0 核心功能

- ✅ **记忆系统 API** — 完整的记忆写入/搜索/上下文/删除接口
- ✅ **认证持久化** — 用户/Token/APIKey 存储到 SQLite，重启不丢失
- ✅ **Prometheus 监控** — 39 个指标，端口 9090

---

## ⚡ 快速开始

```bash
# 🚀 新手一秒上手（自动创建配置 + 索引文档）
./cortex install ~/my-docs

# 如果需要交互式配置 embedding 提供商
./cortex setup

# 🎯 一条命令启动所有服务（推荐）
cortex start
# → 自动检测可用端口（2021→2022→...）
# → 同时启动 MCP stdio + REST API
# → 自动注册到 Claude Code / Cursor / OpenCode / Trae
# → 打印接入指引面板

# 仅注册到 AI 工具（不启动服务）
cortex start --register-only

# 自定义端口或禁用某模式
cortex start --port 3000           # 指定 REST API 端口
cortex start --no-mcp              # 仅启动 REST API
cortex start --no-rest             # 仅启动 MCP
cortex start --no-register         # 跳过自动注册

# 索引文档（显示实时进度条，支持断点续传）
cortex index ~/my-docs

# 高级索引选项
cortex index ~/my-docs --force           # 从头重新索引
cortex index ~/my-docs --timeout 30m     # 30 分钟超时
cortex index ~/my-docs --workers 32      # 32 个并发 worker

# 文件监听（自动增量索引）
cortex watch ~/my-docs                    # 文件变更自动重新索引

# 启动 MCP 服务器（供 AI Agent 使用）
cortex mcp

# 搜索
cortex search "如何实现 Go 并发"

# 配置热加载（serve 模式下自动启用）
cortex serve                              # 修改 config.yaml 无需重启

# 知识库去重
cortex dedup                    # 内容哈希去重
cortex dedup --mode vector      # 向量语义去重

# 查看用量
cortex usage                    # 存储用量和套餐信息
```

---

## ✨ 核心特性

<div align="center">
<table>
<tr>
<td align="center" width="33%">
<h3>🚀 单二进制</h3>
<p><sub>下载即运行，无 Python/Node/Docker 依赖<br><code>curl → cortex mcp</code></sub></p>
</td>
<td align="center" width="33%">
<h3>🔌 MCP 原生</h3>
<p><sub>8 个 MCP 工具，AI Agent 开箱即用<br><code>cortex_search</code> · <code>cortex_memory_write</code></sub></p>
</td>
<td align="center" width="33%">
<h3>🌐 三模接入</h3>
<p><sub>MCP stdio · MCP over SSE · REST API<br><code>cortex start</code> 全自动适配</sub></p>
</td>
<td align="center" width="33%">
<h3>🧠 记忆系统</h3>
<p><sub>长期记忆 + RAG 上下文<br>Agent 跨会话记住用户偏好</sub></p>
</td>
</tr>
<tr>
<td align="center" width="33%">
<h3>🔍 混合搜索</h3>
<p><sub>向量语义 + BM25 关键词<br>RRF 融合排序，精准召回</sub></p>
</td>
<td align="center" width="33%">
<h3>🤖 自动注册</h3>
<p><sub>一键注册 Claude Code / Cursor<br>OpenCode / Trae，备份保护</sub></p>
</td>
<td align="center" width="33%">
<h3>📊 Prometheus</h3>
<p><sub>39 个监控指标<br><code>:9090/metrics</code></sub></p>
</td>
</tr>
<tr>
<td align="center" width="33%">
<h3>🗑️ 三层去重</h3>
<p><sub>内容哈希 + 向量语义 + MinHash<br><code>cortex dedup</code> 一键清理</sub></p>
</td>
<td align="center" width="33%">
<h3>🔒 安全加固</h3>
<p><sub>IP 限流 / 密码策略 / Header认证<br>路径保护 / 文件大小限制</sub></p>
</td>
<td align="center" width="33%">
<h3>🐳 Docker 优化</h3>
<p><sub>多阶段构建<br>镜像 ~30MB</sub></p>
</td>
</tr>
</table>
</div>

### 🔌 MCP 工具一览

| 工具 | 说明 | 对应 REST API | 传输方式 |
|------|------|--------------|---------|
| `cortex_search` | 混合搜索（向量 + BM25） | `GET /v1/search` | stdio / SSE |
| `cortex_context` | RAG 上下文组装 | `GET /v1/context` | stdio / SSE |
| `cortex_memory_write` | 写入记忆条目 | `POST /v1/memory` | stdio / SSE |
| `cortex_memory_search` | 搜索记忆条目 | `GET /v1/memory/search` | stdio / SSE |
| `cortex_memory_delete` | 删除单条记忆 | `DELETE /v1/memory/:id` | stdio / SSE |
| `cortex_memory_delete_batch` | 批量删除记忆 | — | stdio / SSE |
| `cortex_health` | 健康检测 + 状态统计 | `GET /health` | stdio / SSE |
| `cortex_suggest` | 预联想搜索建议 | `GET /v1/suggest` | stdio / SSE |

> **传输方式**：  
> - `stdio` — `cortex mcp` / `cortex start`（标准 MCP 客户端如 Claude Code）  
> - `SSE` — `POST /mcp` 端点（`cortex start` / `cortex serve` 自动挂载，任意 HTTP 客户端可用）

---

## 🏗️ 系统架构

```
┌──────────────────────────────────────────────────────────┐
│                      Cortex CLI                          │
├──────────────────────────────────────────────────────────┤
│  index  │  search  │  start  │  serve  │  mcp  │  watch  │
├──────────────────────────────────────────────────────────┤
│                  混合搜索引擎                              │
│      向量搜索 (内存索引)    │      FTS5 (BM25)              │
├──────────────────────────────────────────────────────────┤
│               L1+L2 两级缓存 (v2.1)                       │
│          go-cache (内存)   │     SQLite                   │
├──────────────────────────────────────────────────────────┤
│                    SQLite 存储层                           │
│    文档表   │  分块表   │  向量表   │  缓存表   │  用户表  │
├──────────────────────────────────────────────────────────┤
│               MCP 协议 · REST API · SSE                  │
│   MCP stdio  │  MCP over SSE (/mcp)  │  15+ REST 端点    │
│   8 个 MCP 工具  │  Prometheus  │  AI 工具自动注册        │
└──────────────────────────────────────────────────────────┘
```

**技术栈：**
- **语言**: Go 1.25+ — 单二进制跨平台（纯 Go，无 CGO）
- **存储**: SQLite + WAL — 零配置嵌入式存储
- **向量**: 内存索引 — 轻量级 O(n) 暴力搜索（HNSW 备选）
- **嵌入**: Ollama / ONNX / None（FTS5-only）
- **协议**: MCP SDK — AI Agent 原生通信
- **监控**: Prometheus — 39 个内置指标 + Grafana 模板

---

## 📡 REST API

| 类别 | 端点 | 方法 | 说明 |
|------|------|------|------|
| **搜索** | `/v1/search` | GET | 混合搜索（向量 + FTS） |
| | `/v1/context` | GET | RAG 上下文构建 |
| **MCP** | `/mcp` | GET/POST | MCP Streamable HTTP (SSE) |
| **记忆** | `/v1/memory` | POST | 写入单条记忆 |
| | `/v1/memory/batch` | POST | 批量写入记忆 |
| | `/v1/memory/search` | GET | 搜索记忆 |
| | `/v1/memory/context` | GET | 记忆 RAG 上下文 |
| | `/v1/memory/:id` | DELETE | 删除记忆 |
| **认证** | `/auth/register` | POST | 注册用户 |
| | `/auth/login` | POST | 登录 |
| | `/auth/logout` | POST | 登出 |
| **监控** | `/health` | GET | 健康检查 |
| | `/metrics` | GET | Prometheus 指标（:9090） |

---

## 🔧 配置

```yaml
# ~/.cortex/config.yaml
cortex:
  db_path: ~/.cortex/cortex.db
  log_level: info

embedding:
  provider: none              # none | ollama | openai | cohere | voyage | dashscope | zhipu | baidu
  auto_update: true           # 联网时自动检查最新模型

  # 本地 Ollama（离线场景默认）
  ollama:
    base_url: http://localhost:11434
    model: all-minilm         # 推荐: all-minilm(23MB), nomic-embed-text(137MB)

  # 国外 API
  openai:
    api_key: sk-xxx
    model: text-embedding-3-small
    dimension: 512
  cohere:
    api_key: co-xxx
    model: embed-multilingual-v3.0
    dimension: 1024
  voyage:
    api_key: pa-xxx
    model: voyage-3-lite
    dimension: 512

  # 国内 API
  dashscope:
    api_key: sk-xxx
    model: text-embedding-v2
    dimension: 1536
  zhipu:
    api_key: xxx
    model: embedding-3
    dimension: 2048
  baidu:
    api_key: xxx
    secret_key: xxx
    model: Embedding-V1
    dimension: 384

index:
  workers: 4
  max_tokens: 512

search:
  cache_ttl: 5m
  default_top_k: 20

# cortex start 命令配置
starter:
  start_port: 0              # REST API 起始端口（0=自动检测）
  auto_register: true        # 自动注册到 AI 工具
  register_skip_tools: []    # 跳过的工具列表

prometheus:
  enabled: true
  port: 9090
```

---

## 📦 支持的文件格式

| 格式 | 支持 | 分块方式 |
|------|------|---------|
| Markdown (.md) | ✅ | 层级分块，标题路径追溯 |
| PDF (.pdf) | ✅ | 文本提取，自动分块 |
| Word (.docx) | ✅ | 段落解析，结构保持 |
| 纯文本 (.txt) | ✅ | 按行/段落分块 |
| 代码 (.go/.py/.js/.ts/.java 等) | ✅ | 按函数/类分块 |
| 配置 (.yaml/.json/.toml/.ini) | ✅ | 结构化提取 |

---

## 🛠️ 开发

```bash
git clone https://github.com/lh123aa/cortex.git
cd cortex
go build -ldflags="-s -w" -o cortex ./cmd/cortex   # 纯 Go，无需 CGO

# 一步安装（自动配置 + 索引）
./cortex install ./docs

# 或手动启动服务
./cortex serve

# 运行测试
go test ./...                     # 15 个包全部通过

# 一键去重
./cortex dedup
```

Windows 小白一键脚本（无需 Go 环境）：
```powershell
.\scripts\install.ps1 -DocDir "D:\我的文档"
```

---

## 📊 性能

| 指标 | 值 |
|------|-----|
| 记忆写入 | < 200ms / 条 |
| 记忆搜索 P50 | < 50ms（缓存命中 < 1ms） |
| 记忆搜索 P95 | < 100ms |
| RAG 上下文 | 精确 Token 裁剪 |
| 删除操作 | < 50ms / 条 |
| 缓存命中率 | > 60%（L1+L2 两级） |
| MinHash 去重 | ~12μs / chunk (102k ops/sec) |
| 向量去重效果 | 46/689 chunks 移除 (6.7%) |
| 启动时间（全部 CLI 命令） | **< 0.5s**（FTS5-only 模式，HNSW 按需异步加载） |
| FTS5 搜索 P50 | **< 50ms**（20k 分块） |
| FTS5 搜索 P95 | **< 200ms**（20k 分块） |
| 中文搜索 | **单字分词，FTS5 精确匹配** |
| 索引吞吐量（FTS5-only） | **> 1000 files/min**（优化后 **~460 files/s**） |
| 全量索引（18k 文件） | **~40 秒** |
| 索引失败率 | **< 5%**（优化后，自动跳过二进制/多媒体） |
| 实时进度反馈 | **动态进度条 · 速度/ETA/当前文件** |
| 中断恢复 | **Checkpoint 自动保存，重启续跑** |
| 文件监听 | **`cortex watch` · fsnotify 实时增量索引** |
| 单文件超时保护 | **5 分钟 Embedding 超时** |
| 配置热加载 | **`cortex serve` 修改配置无需重启** |
| 自动备份 | **24h 间隔 · 保留最近 10 份** |
| 内存占用 | ~30 MB / 进程 |
| 二进制大小 | 34.2 MB (strip 优化) |
| 测试覆盖率 | 10/10 包通过 · 全部核心逻辑有测试 |
| MCP 工具 | **8 个**（search/context/memory×4/health/suggest）<br>**双传输**: stdio + SSE (/mcp 端点) |
| Bug 状态 | 0 · FG 循环验证通过 |

---

## 🤝 贡献

- 🐛 发现 Bug？[提交 Issue](https://github.com/lh123aa/cortex/issues)
- 💡 有好想法？[讨论区](https://github.com/lh123aa/cortex/discussions)
- ⭐ 项目对你有帮助？点个 Star 支持

## 📄 许可证

**MIT License** — 可自由使用、修改、商业化。

---

## ⭐ Star History

[![Star History Chart](https://api.star-history.com/svg?repos=lh123aa/cortex&type=Date)](https://star-history.com/#lh123aa/cortex&Date)

---

## 💬 社区 & 支持

- ⭐ **给个 Star** — 支持项目最好的方式
- 🐛 **报告 Bug** — [提交 Issue](https://github.com/lh123aa/cortex/issues)
- 💡 **功能建议** — [发起 Discussion](https://github.com/lh123aa/cortex/discussions)
- 📣 **分享推荐** — 在掘金、知乎、V2EX 推荐 Cortex
- 🤝 **贡献代码** — 阅读 [CONTRIBUTING.md](CONTRIBUTING.md)

## 🔗 相关资源

- [MCP 协议规范](https://modelcontextprotocol.io) — AI Agent 通信标准
- [Ollama](https://ollama.ai) — 本地 LLM & Embedding
- [OpenCode](https://opencode.ai) — AI Agent 框架
- [Awesome MCP Servers](https://github.com/lh123aa/awesome-mcp-servers) — MCP 服务器资源列表

---

<p align="center">
  <strong>Cortex 不是「又一个知识库工具」，而是 AI Agent 时代的基础设施。</strong>
  <br>
  <sub>在 MCP 协议生态快速发展的当下，它解决了一个真实且普遍的需求——给 AI Agent 装上持久记忆。单二进制、零配置、MCP 原生、MIT 开源，任何人都可以自由使用。</sub>
</p>

---

<p align="center">
  <strong>🧠 Cortex v3.5 — 让 AI Agent 拥有记忆</strong>
  <br>
  <sub>单二进制 · 零配置 · MCP 原生 · 完全本地 · MIT 开源</sub>
</p>

<p align="center">
  <a href="https://github.com/lh123aa/cortex/stargazers">
    <img src="https://img.shields.io/github/stars/lh123aa/cortex?style=social" alt="Star">
  </a>
  <a href="https://github.com/lh123aa/cortex/forks">
    <img src="https://img.shields.io/github/forks/lh123aa/cortex?style=social" alt="Fork">
  </a>
  <a href="https://github.com/lh123aa/cortex">
    <img src="https://img.shields.io/github/followers/lh123aa?style=social" alt="Follow">
  </a>
</p>
