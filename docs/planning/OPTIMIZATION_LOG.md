# Cortex 优化执行日志

> 开始时间: 2026-05-13
> 基于评估报告逐步执行优化方案，记录每个步骤和发现的问题。

## 阶段1-P0: 添加文件排除规则

### 目标
减少索引时扫描二进制/多媒体文件的无效尝试，降低索引失败率。

### 执行过程
1. 在 `internal/index/indexer.go` 中添加 `defaultSkipExts` 映射表，覆盖 50+ 种无需索引的扩展名
2. 添加 `isSkippableExt(path)` 函数，在文件遍历阶段提前过滤
3. 在 3 个 `filepath.Walk` 位置全部加入过滤
4. 在 `defaultExcludeDirs` 中添加 `bin`/`out`/`.terraform`/`target` 目录

### 对比结果
| 指标 | 优化前 | 优化后 | 改善 |
|:----|:-----:|:-----:|:----:|
| 索引失败 | 174 | 8 | **-95%** |
| 索引耗时 | 311ms | 89ms | **3.5x 提速** |
| 编译 | — | 通过 | ✅ |

### 问题
- 剩余 8 个失败，需要检查未被覆盖的文件类型

---

## 阶段1-P0: 实现 cortex watch 文件监听命令

### 目标
利用已有的 `fsnotify` 依赖实现文件变更监听，文件修改后自动增量索引。

### 执行过程
1. 在 `cmd/cortex/main.go` 中添加 `watchCmd` 命令定义
2. 添加 `runWatch` 函数，调用已有的 `index.NewIncrementalWatcher`
3. 注册到 `rootCmd.AddCommand(watchCmd)`
4. `IncrementalWatcher` 和 `fsnotify` 依赖均已存在，无需新增依赖

### 结果
- 编译通过 ✅
- 命令注册成功 ✅
- `cortex watch <path>` 可用

### 用法
```bash
cortex watch e:\程序\魔盒系统\INFOSYS   # 监听项目目录
cortex watch e:\程序\Cortex              # 监听自身项目
```

---

## 阶段1-P1: setup 向导默认开启认证

### 目标
新安装用户默认开启认证，提高安全性。

### 执行过程
1. 在 `internal/embedding/setup.go` 的 `WriteConfig()` 中，新增 cortex 配置段
2. 首次 setup 时自动设置 `auth_enabled: true`、默认 db_path 和 log_level
3. 已有配置不受影响（`existing` map 保留旧值）

### 结果
- 编译通过 ✅
- 新用户 setup 后自动开启认证

---

## 阶段2: 中文搜索优化（2-gram → 单字分词）

### 问题
中文 2-gram 分词导致搜索词和索引词不一致。例如搜"商品价格"展开为"商品 品价 价格"，但索引存的是"商品 品销 销售 售价 价格"，"品价"在索引中不存在 → FTS5 AND 模式全不命中。

### 修复
1. 索引端 `chinese.go`: `segmentChineseRunes()` 从 2-gram 改为单字分词
   - 旧: "商品销售价格" → "商品 品销 销售 售价 价格"
   - 新: "商品销售价格" → "商 品 销 售 价 格"
2. 搜索端 `search.go`: `expandChineseQuery()` 从 2-gram 改为单字展开
   - 旧: "商品价格" → "商品 品价 价格"（"品价"不存在→不命中）
   - 新: "商品价格" → "商 品 价 格"（完美匹配）
3. `Indexer` 结构体新增 `Force` 字段，`--force` 标志跳过内容哈希检查，使新分词生效
4. FTS5 验证:
   - `"信 息 管 理"` → 3 条匹配 ✅
   - `"商 品 库 存"` → 7 条匹配 ✅

### 问题
- 旧索引数据仍用 2-gram，需要 `--force` 全量重建

---

## 阶段2: 配置热加载

### 说明
`internal/config/config.go` 的 `WatchConfig()` 已实现完整的热加载逻辑（基于 viper 的 `OnConfigChange` + `fsnotify`），但从未被调用。

### 修复
在 `cmd/cortex/main.go` 的 `runServe()` 中添加 `config.WatchConfig()` 调用，`cortex serve` 启动后修改 `~/.cortex/config.yaml` 自动生效，无需重启。

### 结果
- 编译通过 ✅
- `cortex serve` 自动启用配置热加载

---

## 阶段3-P0: 自动定时备份

### 改动
1. `storage/backup.go`: 添加 `StartAutoBackup(interval)` 方法，后台 goroutine 定时备份
2. `BackupManager` 新增 `maxKeep`、`stopCh` 字段，自动清理旧备份
3. `cmd/cortex/main.go` 的 `runServe()` 中，配置 `backup.auto_backup: true` 时自动启用

### 结果
- 默认保留最近 10 份备份
- 备份路径: `~/.cortex/backups/cortex_YYYYMMDD_HHMMSS.db`

---

## 阶段3-P1: MCP 健康检测工具

### 改动
1. `internal/api/mcp.go`: 新增 `cortex_health` MCP 工具
2. 返回服务器版本、运行时间、文档数量、状态

### 结果
- 共 7 个 MCP 工具（原 6 个 + health）
- 在 Trae 中可直接调用 `cortex_health` 检查连接

---

## 阶段3-P2: 全量重建索引

### 说明
运行 `cortex index --force e:\程序` 全量重建，使单字分词生效。

### 结果
| 指标 | 重建前 | 重建后 |
|:----|:-----:|:-----:|
| 文档数 | 26,738 | 26,956 |
| 分块数 | 83,924 | 83,492 |
| 索引耗时 | — | 40.4s |
| 中文搜索 | ❌ 不命中 | ✅ 正常匹配 |

### 最终状态
所有 6 项优化完成，Cortex 已编译并重建索引，可直接在 Trae 中配置 MCP 使用。

---