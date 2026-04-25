# Cortex OpenCode 测试脚本

## 测试概述

本目录包含用于验证 Cortex 项目功能的测试脚本。

## 测试列表

### 1. health_check.ps1
检查服务健康状态

```powershell
.\health_check.ps1
```

### 2. memory_test.ps1
测试记忆系统

```powershell
.\memory_test.ps1
```

### 3. search_test.ps1
测试搜索功能

```powershell
.\search_test.ps1
```

### 4. metrics_check.ps1
检查 Prometheus 指标

```powershell
.\metrics_check.ps1
```

### 5. full_test.ps1
完整测试套件

```powershell
.\full_test.ps1
```

---

## 前置要求

1. 启动 Ollama:
   ```bash
   ollama serve
   ```

2. 启动 Cortex:
   ```bash
   go run cmd/cortex/main.go serve
   ```

3. 索引测试文档:
   ```bash
   go run cmd/cortex/main.go index ./docs
   ```

---

## 运行测试

```powershell
# 进入测试目录
cd test_scripts

# 运行所有测试
.\full_test.ps1

# 运行单个测试
.\health_check.ps1
```

---

## 测试结果

| 测试 | 状态 |
|------|------|
| 健康检查 | ✅ |
| 搜索功能 | ✅ |
| 记忆系统 | ✅ |
| 监控指标 | ✅ |

---

*生成时间: 2026-04-25*
