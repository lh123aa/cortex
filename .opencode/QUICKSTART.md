# Cortex + OpenCode 快速开始

## 1. 环境准备

### 安装 Go
```bash
# Windows
winget install Go

# 验证
go version
```

### 安装 Ollama
```bash
# Windows/macOS/Linux
curl -fsSL https://ollama.com/install.sh | sh

# 启动服务
ollama serve

# 下载 embedding 模型
ollama pull nomic-embed-text
```

## 2. 启动 Cortex

```bash
# 进入项目目录
cd E:\程序\Cortex

# 索引文档
go run cmd/cortex/main.go index ./docs

# 启动 API 服务
go run cmd/cortex/main.go serve
```

## 3. OpenCode 集成

### 触发 Cortex 技能

在 OpenCode 中使用以下触发词：

- `cortex` - 激活 Cortex 技能
- `索引文档` - 索引新文档
- `搜索知识库` - 搜索内容
- `记忆管理` - 管理记忆
- `AI知识库` - 通用知识库操作

### 示例对话

```
用户:帮我索引项目文档
Agent: go run cmd/cortex/main.go index ./docs

用户:搜索关于配置的内容
Agent: go run cmd/cortex/main.go search "配置"

用户:添加一条记忆
Agent: curl -X POST http://localhost:8080/v1/memory ...
```

## 4. 测试验证

```powershell
# 运行完整测试
cd .opencode/test_scripts
.\full_test.ps1
```

## 5. API 使用

### 搜索
```bash
curl "http://localhost:8080/v1/search?q=关键词&top_k=10"
```

### 记忆
```bash
# 写入
curl -X POST http://localhost:8080/v1/memory \
  -H "Content-Type: application/json" \
  -d '{"content": "内容", "tags": ["标签"]}'

# 搜索
curl "http://localhost:8080/v1/memory/search?q=关键词"
```

## 6. 监控

- API: http://localhost:8080
- Metrics: http://localhost:9090/metrics
- Health: http://localhost:8080/health

---

## 故障排除

### Ollama 连接失败
```bash
# 检查 Ollama 状态
curl http://localhost:11434/api/version

# 重启 Ollama
ollama serve
```

### Cortex 启动失败
```bash
# 检查端口占用
netstat -an | grep 8080

# 查看日志
go run cmd/cortex/main.go serve --log-level debug
```

---

*版本: Cortex v2.0 | OpenCode 集成*
