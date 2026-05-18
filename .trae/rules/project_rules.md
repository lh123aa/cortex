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

### 推送前检查清单

改完代码推送到 GitHub **之前**，务必在本地运行以下命令：

```powershell
# 方式一：一键 CI 模拟（推荐）
make ci-check

# 方式二：手动逐项检查
go mod tidy               # 确保 go.mod/go.sum 一致
go vet ./...              # 静态检查（零警告）
$env:CGO_ENABLED=1; go test -count=1 -timeout=300s ./...   # CGO 模式测试
$env:CGO_ENABLED=0; go test -count=1 -timeout=300s ./...   # 纯 Go 模式测试
go build ./cmd/cortex     # 编译验证
```

注意：
- 有 3 个测试文件带 `//go:build cgo` 约束（`mcp_test.go`、`crud_test.go`、`index_progress_test.go`），`CGO_ENABLED=0` 时不会执行
- **改 CGO 测试文件后，必须用 `CGO_ENABLED=1` 跑一遍**，否则推上 CI 才炸
- Windows 本地的 modernc.org/sqlite 行为可能和 Linux CI 的 CGO SQLite 有差异

### 数据位置

- 数据库: `C:\Users\49046\.cortex\cortex.db`
- 配置文件: `C:\Users\49046\.cortex\config.yaml`
- 当前状态: 27147 个文档, 84537 个分块, FTS5-only 模式（零外部依赖）
