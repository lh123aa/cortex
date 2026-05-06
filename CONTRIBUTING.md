# 贡献指南

感谢你考虑为 Cortex 贡献代码！🎉

无论你是修 Bug、加功能、写文档，还是提建议，都欢迎。

## 快速导航

- [行为准则](#行为准则)
- [如何报告 Bug](#如何报告-bug)
- [如何提功能建议](#如何提功能建议)
- [如何提交代码](#如何提交代码)
- [开发环境搭建](#开发环境搭建)
- [代码规范](#代码规范)
- [提交信息规范](#提交信息规范)
- [发布流程](#发布流程)

## 行为准则

本项目采用 [Contributor Covenant](CODE_OF_CONDUCT.md) 行为准则。参与即表示同意遵守。

## 如何报告 Bug

1. **先搜一下** — 去 [Issues](https://github.com/lh123aa/cortex/issues) 看看有没有人报过同样的 Bug
2. **新建 Issue** — 选择 Bug Report 模板，填：
   - 你做了什么
   - 期望结果 vs 实际结果
   - 运行环境（OS、Go 版本）
   - 日志或截图（如果有）

## 如何提功能建议

1. **先搜一下** — 看有没有人提过类似需求
2. **新建 Issue** — 选择 Feature Request 模板，写清楚：
   - 你想解决什么问题
   - 你期望的方案是什么
   - 有没有参考的其他项目

## 如何提交代码

### 新手友好流程

```bash
# 1. Fork 仓库（在 GitHub 网页上点 Fork）

# 2. 克隆到本地
git clone https://github.com/你的用户名/cortex.git
cd cortex

# 3. 创建功能分支（不要直接在 main 上改）
git checkout -b feat/你的功能名

# 4. 改代码... 然后确保测试通过
go test ./...

# 5. 提交（提交信息格式见下方规范）
git add .
git commit -m "feat: 简洁描述你的改动"

# 6. 推送到你的仓库
git push origin feat/你的功能名

# 7. 在 GitHub 上创建 Pull Request
```

### Pull Request 注意事项

- PR 标题要清晰，例如 `fix: 修复搜索时内存泄露`
- 如果 PR 关联某个 Issue，在描述里写 `Closes #123`
- 确保所有 CI 检查通过
- 保持 PR 范围集中，一个 PR 只做一件事

## 开发环境搭建

### 前置要求

- Go 1.21 或更高版本
- （可选）Ollama，用于向量嵌入功能

### 本地编译

```bash
# 纯 Go 编译，无需 CGO
git clone https://github.com/lh123aa/cortex.git
cd cortex
go build -o cortex ./cmd/cortex

# 运行
./cortex serve

# 全部测试
go test ./...
```

### 常用命令

```bash
go build -o cortex ./cmd/cortex   # 编译
go test ./...                      # 跑测试
go fmt ./...                       # 格式化代码
go vet ./...                       # 代码检查
```

## 代码规范

- **Go 标准** — 遵循 [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- **`go fmt`** — 提交前必须运行，确保格式统一
- **错误处理** — 不要吞错误，不要 panic（除非极特殊情况）
- **测试** — 新功能要带测试，修 Bug 要加测试验证
- **性能** — 关注内存分配和并发安全

## 提交信息规范

我们使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```
<类型>: <简短描述>

<可选详细说明>
```

| 类型 | 什么时候用 |
|------|-----------|
| `feat` | 新功能 |
| `fix` | 修 Bug |
| `docs` | 文档变更 |
| `refactor` | 重构（不改功能、不修 Bug） |
| `test` | 加测试 |
| `perf` | 性能优化 |
| `chore` | 构建、工具链等杂项 |

示例：

```
feat: 添加 ONNX 嵌入支持

新增 ONNX Runtime 作为嵌入引擎选项，
用户可通过 embedding.provider: onnx 配置。
```

## 发布流程

维护者操作：

1. 确认所有测试通过
2. 更新版本号
3. 运行 `make changelog` 生成更新日志
4. 打 Git Tag 并推送：`git tag v2.3.0 && git push --tags`
5. CI 会自动构建并创建 Release Draft

## 项目结构

```
cortex/
├── cmd/cortex/       # 主入口
├── internal/         # 内部包
│   ├── api/         # REST API 层
│   ├── auth/        # 认证模块
│   ├── cache/       # 缓存（L1+L2）
│   ├── config/      # 配置
│   ├── embedding/   # 嵌入引擎
│   ├── indexer/     # 文档索引
│   ├── mcp/         # MCP 服务器
│   ├── memory/      # 记忆系统
│   ├── search/      # 搜索引擎
│   ├── storage/     # 存储层
│   └── metrics/     # Prometheus 监控
└── docs/            # 文档
```

## 需要帮助？

- 看 [README](README.md) 了解项目概览
- 看 [Issues](https://github.com/lh123aa/cortex/issues) 找可以做的事
- 在 [Discussions](https://github.com/lh123aa/cortex/discussions) 提问

---

**Cortex** — 让 AI Agent 拥有记忆 🧠
