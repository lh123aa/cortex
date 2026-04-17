# Cortex

Cortex 是一个用 Go 编写的 Agent 知识库，核心基于 `SQLite`(FTS+向量遍历) 提供 "单二进制极简部署" 体验。

## CLI 快速上手

你可以使用提供的一键安装脚本，或者手动编译。

### 一键安装脚本 (支持 macOS/Linux)
```bash
chmod +x ./scripts/install.sh
./scripts/install.sh
```

### 手动安装
```bash
# 获取源码后本地打包
go mod tidy
go build -o cortex ./cmd/cortex/main.go

# 1. 索引本地目录 (先确保本地起了 Ollama - ollama run nomic-embed-text)
./cortex index ./markdown_docs

# 2. 从已有的大纲混合搜索 (支持 FTS 及 Vector)
./cortex search "你想搜索的内容"

# 3. 构造给大模型的纯粹可读上下文
./cortex context "如何部署?"

# 4. 作为 Cursor / Claude Desktop 的底层 MCP Agent 服务
./cortex mcp
```

## 测试与工程化
项目包含了完整的单元测试和 Makefile 支持：
```bash
make test    # 跑测试用例
make build   # 快速构建
make format  # 统一格式化代码
```

## Docker Compose 部署
```bash
# 一键拉起 Cortex (MCP监听态) + Ollama 本地大模型
docker-compose up -d
```

## Agent (Claude / Cursor) 配置文件

如果你希望 Agent 直连本引擎进行原生交互，可以将以下内容配置到你的 `claude_desktop_config.json` 中：

```json
{
  "mcpServers": {
    "cortex": {
      "command": "/absolute/path/to/compiled/cortex",
      "args": ["mcp"]
    }
  }
}
```
