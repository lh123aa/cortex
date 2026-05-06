# Changelog

## v2.2 (2026-05-05)

### 🚀 MCP 记忆工具

- 新增 5 个 MCP 工具：`cortex_search`、`cortex_context`、`cortex_memory_write`、`cortex_memory_search`、`cortex_memory_delete`
- 新增零依赖模式：`embedding.provider: none`，FTS5 全文搜索，无需 Ollama
- 切换到纯 Go SQLite 驱动 `modernc.org/sqlite`，无需 CGO/gcc
- 新增 MCP 优雅关闭（Signal 处理，Ctrl+C 安全退出）
- 新增 11 个 MCP 单元测试覆盖全部工具边界条件
- 新增产品对比表，双语 README 重构

## v2.1 (2026-04-25)

### 🔥 生产环境强化

- 新增 L1+L2 两级缓存（内存 go-cache + SQLite），搜索速度提升 10x
- 新增 Graceful Shutdown，30s 窗口处理完现有请求
- 新增请求超时控制（默认 30s，搜索 60s，索引 5min）
- 新增 API 限流（令牌桶，100 req/s，突发 200）
- 36 个测试用例覆盖存储/认证/搜索核心模块
- 系统测试报告和生产环境测试脚本

## v2.0 (2026-04)

### ✨ 核心功能

- 完整的记忆系统 API（写入/搜索/上下文/删除）
- 认证持久化（用户/Token/APIKey 存储到 SQLite）
- Prometheus 监控（39 个指标，端口 9090）
- 多用户支持
- 向量 PQ 压缩
- 高级搜索功能

## v1.3 (2026-03)

### 🧪 测试补全

- 全面测试覆盖
- 健康检查端点
- 错误处理强化
- 向量持久化

## v1.2 (2026-02)

### 🔧 基础完善

- HNSW 向量索引
- API 认证
- Prometheus 指标
- 配置热更新
- 断点恢复

## v1.1 (2026-01)

### 📄 文件支持

- 35+ 文件类型支持（Go、Python、JS、TS 等）
- 新增 Go chunker 和 Text chunker
- GitHub Actions CI/CD 跨平台构建
- Windows FTS5 支持

## v1.0 (2025-12)

### 🎉 首个版本

- 基础文档索引和搜索
- 双语 README（中文/英文）
- MCP 工具发现
- ONNX 嵌入支持
- 单元测试
