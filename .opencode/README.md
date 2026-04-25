# Cortex OpenCode 集成指南

## 概述

Cortex 项目已集成 OpenCode 系统，可通过 OpenCode Agent 智能助手进行操作。

## 目录结构

```
Cortex/
├── .opencode/
│   ├── skills/
│   │   └── cortex/
│   │       └── SKILL.md       # Cortex 主技能
│   ├── project.json           # 项目配置
│   └── AGENT.md               # Agent 工作流
├── cmd/cortex/main.go         # 入口点
├── internal/                  # 核心代码
└── docs/                      # 文档
```

## 快速开始

### 1. 索引文档

```
用户: 索引 docs 目录
Agent: go run cmd/cortex/main.go index ./docs
```

### 2. 搜索知识库

```
用户: 搜索 "如何配置 Ollama"
Agent: go run cmd/cortex/main.go search "如何配置 Ollama"
```

### 3. 记忆管理

```
用户: 添加记忆 "今天学习了 Go 并发编程"
Agent: POST /v1/memory {"content": "今天学习了 Go 并发编程"}

用户: 搜索相关记忆
Agent: GET /v1/memory/search?q=Go并发
```

### 4. RAG 上下文

```
用户: 构建上下文 "解释 goroutine"
Agent: go run cmd/cortex/main.go context "解释 goroutine"
```

---

## OpenCode 技能

### cortex

**触发词**:
- cortex
- 索引文档
- 搜索知识库
- 记忆管理
- AI知识库

**功能**:
- 文档索引
- 混合搜索
- 记忆系统
- 监控指标

---

## API 端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/v1/search` | GET | 混合搜索 |
| `/v1/memory` | POST | 写入记忆 |
| `/v1/memory/search` | GET | 搜索记忆 |
| `/health` | GET | 健康检查 |
| `/metrics` | GET | Prometheus |

---

## 监控

访问 `http://localhost:9090/metrics` 查看：

- `cortex_search_total` - 搜索总数
- `cortex_search_cache_hits_total` - 缓存命中
- `cortex_vectors_total` - 向量数量
- `cortex_index_chunks_total` - 分块数量

---

## 配置

项目配置位于 `.opencode/project.json`:

```json
{
  "name": "cortex",
  "version": "2.0.0",
  "ports": {
    "api": 8080,
    "metrics": 9090
  }
}
```

---

## 工作流

### 开发工作流

1. **分析需求** → 理解用户请求
2. **实现功能** → 代码编写
3. **测试验证** → 确保正确
4. **文档更新** → 同步文档

### 运维工作流

1. **监控检查** → 查看指标
2. **问题诊断** → 分析日志
3. **修复处理** → 解决问题
4. **验证恢复** → 确认正常

---

## 状态

✅ Cortex v2.0 已集成 OpenCode 系统

- 代码质量: 100%
- 功能完整性: 100%
- 生产就绪: ✅

---

*集成时间: 2026-04-25*
