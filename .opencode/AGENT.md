# Cortex OpenCode Agent 配置

## 概述

本配置文件定义了 Cortex AI 知识库系统的 OpenCode Agent 工作流程。

## Agent 类型

### cortex-developer
- **用途**: Cortex 项目开发
- **技能**: Go, REST API, SQLite, 向量搜索

### cortex-operator
- **用途**: Cortex 日常运维
- **技能**: 服务管理, 监控, 备份

---

## 工作流程

### 1. 开发流程

```
需求分析
    ↓
代码实现
    ↓
单元测试
    ↓
集成测试
    ↓
代码审查
    ↓
合并部署
```

### 2. 索引流程

```
接收文档路径
    ↓
文件检测 (markdown, pdf, docx)
    ↓
内容提取
    ↓
分块处理
    ↓
Embedding 生成
    ↓
向量存储
    ↓
索引完成
```

### 3. 搜索流程

```
接收查询
    ↓
Embedding 查询向量
    ↓
并行: 向量搜索 + FTS 搜索
    ↓
RRF 融合
    ↓
重排序 (可选)
    ↓
返回结果
```

### 4. 记忆流程

```
写入记忆 → 内容分块 → Embedding → 存储
搜索记忆 → 向量检索 → 过滤 memory://
上下文 → RAG 构建 → 返回上下文
```

---

## 命令

### 索引
```bash
go run cmd/cortex/main.go index <path>
```

### 搜索
```bash
go run cmd/cortex/main.go search <query>
```

### RAG 上下文
```bash
go run cmd/cortex/main.go context <query>
```

### 启动服务
```bash
go run cmd/cortex/main.go serve
```

### 状态检查
```bash
go run cmd/cortex/main.go status
```

---

## 环境要求

- Go 1.21+
- Ollama (localhost:11434)
- 8GB+ RAM

---

## 监控

- API: http://localhost:8080
- Metrics: http://localhost:9090/metrics
- Ollama: http://localhost:11434

---

## 常见任务

| 任务 | 命令 |
|------|------|
| 索引文档 | `go run cmd/cortex/main.go index ./docs` |
| 搜索 | `go run cmd/cortex/main.go search "query"` |
| 启动API | `go run cmd/cortex/main.go serve` |
| 检查状态 | `go run cmd/cortex/main.go status` |

---

*生成时间: 2026-04-25*
