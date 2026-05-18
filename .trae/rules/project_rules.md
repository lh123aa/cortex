# Cortex - Trae IDE 项目规则

## MCP 服务配置

Cortex 提供了 MCP 服务（stdio 模式），供 AI Agent 调用知识库搜索和记忆功能。

### 在 Trae 中配置

Trae 设置 → MCP 服务器 → 添加 MCP 服务器：

| 字段 | 值 |
|:----|:----|
| 名称 | `Cortex` |
| 命令 | `E:\程序\Cortex\bin\cortex.exe` |
| 参数 | `mcp` |
| 类型 | `stdio` |

### 可用的 MCP 工具

| 工具 | 功能 |
|:----|:------|
| `cortex_search` | 混合搜索（向量 + FTS） |
| `cortex_context` | RAG 上下文组装 |
| `cortex_memory_write` | 写入记忆条目 |
| `cortex_memory_search` | 搜索记忆条目 |
| `cortex_memory_delete` | 删除记忆条目 |
| `cortex_memory_delete_batch` | 批量删除记忆 |
| `cortex_health` | 健康检测 + 状态统计 |
| `cortex_suggest` | 预联想搜索建议 |

### 构建命令

```powershell
cd E:\程序\Cortex
go build -ldflags="-s -w" -o bin\cortex.exe .\cmd\cortex
```

### 常用命令

```powershell
# 一步安装（无 Go 环境也能用 install.ps1）
.\scripts\install.ps1 -DocDir "D:\文档"
# 或直接用 cortex install（需已编译）
.\bin\cortex.exe install <目录>
.\bin\cortex.exe install   # 默认当前目录

# 索引文档
.\bin\cortex.exe index <目录>

# 搜索
.\bin\cortex.exe search <关键词>

# 状态查看
.\bin\cortex.exe status

# 启动 MCP 服务（供 Trae AI 调用）
.\bin\cortex.exe mcp

# 配置向导（设置 embedding 提供商）
.\bin\cortex.exe setup
```

## 推送规则（强制）

### 核心原则：先检查，再推送

**任何推送到 GitHub 的代码，必须在本地先通过完整的 CI 检查。** 这条规则是硬性的，由 git pre-push hook 自动强制执行。

### git pre-push hook（自动安全网）

项目已内置 pre-push hook，推送前自动运行 `make ci-check`，未通过则阻止推送。

**安装 hook（只需执行一次）：**

```powershell
git config core.hooksPath .githooks
```

安装后，每次执行 `git push` 都会自动触发检查，通不过就推不上去。

如需紧急跳过（不推荐）：`git push --no-verify`

### 更新三件套工作流

每次处理"更新三件套"（README更新 → 推GitHub → 更新本地软件）时，严格遵循以下流程：

```text
1. 修改代码 / 更新 README
2. 运行 make ci-check（或手动逐项检查）
       ↓ 通过
3. git add + git commit
4. git push（hook 自动二次验证）
       ↓ CI 通过
5. 更新本地软件（编译 + 替换二进制）
```

任何一个步骤失败 → **不得继续下一步**，先修复问题。

### 手动检查命令（兜底）

如果 hook 没装或需要单独排查：

```powershell
# 一键 CI 模拟（推荐）
make ci-check

# 手动逐项
go mod tidy               # 确保 go.mod/go.sum 一致
go vet ./...              # 静态检查（零警告）
$env:CGO_ENABLED=1; go test -count=1 -timeout=300s ./...   # CGO 模式测试
$env:CGO_ENABLED=0; go test -count=1 -timeout=300s ./...   # 纯 Go 模式测试
go build ./cmd/cortex     # 编译验证
```

### 常见 CI 失败根因（历史教训）

| 问题 | 根因 | 预防 |
|:----|:-----|:-----|
| `go vet` 报 missing import | 新增代码引用了包但没加 import | `make ci-check` 中 `go vet` 会抓到 |
| CGO 测试文件没跑 | 3 个文件带 `//go:build cgo` 约束，`CGO_ENABLED=0` 时不执行 | 必须用 `CGO_ENABLED=1` 跑一遍 |
| SQLite DSN 参数拼接错误 | Windows 的 modernc.org/sqlite 和 Linux CI 的 CGO SQLite 行为不同 | 两种模式都测 |
| 外键约束测试失败 | CGO 模式下 SQLite 外键生效 | 保存 chunks 前先 save document |
| SaveIndexProgress 空 ID | `INSERT OR REPLACE` 传空 ID 导致主键碰撞 | 用 `UPDATE+INSERT` 模式 |

### 数据位置

- 数据库: `C:\Users\49046\.cortex\cortex.db`
- 配置文件: `C:\Users\49046\.cortex\config.yaml`
- 当前状态: 27147 个文档, 84537 个分块, FTS5-only 模式（零外部依赖）
