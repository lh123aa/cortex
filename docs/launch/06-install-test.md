# Cortex 实际安装测试报告

> 测试日期：2026-11-01
> 环境：Windows 11 / x86_64 / PowerShell 5.1+ / `CGO_ENABLED=0`
> 测试方式：模拟真实用户从零安装，每种场景独立目录，避免污染既有环境

---

## 0. 结论

**安装流程已完整验证可用（Windows + Linux 双平台），共发现并修复 7 个真实缺陷。**

| # | 缺陷 | 严重度 | 状态 |
|---|---|---|---|
| 1 | Release 无产物，官方一行安装命令必然失败 | 🔴 阻断 | ✅ 已修（v4.0.0 已发布） |
| 2 | 源码构建装出来版本显示 `dev` | 🟠 高 | ✅ 已修 |
| 3 | `cortex install` 日志走 stderr，导致脚本 exit 1 | 🟠 高 | ✅ 已修 |
| 4 | `.go` 文件索引 100% 失败（bundle 写入列名错误） | 🔴 阻断 | ✅ 已修 + 回归测试 |
| 5 | CLI 搜索结果打印内部 DocumentID 而非文件路径 | 🟠 高 | ✅ 已修 |
| 6 | 过短文档从统计中彻底消失（既不算成功也不算跳过） | 🔴 阻断 | ✅ 已修 + 回归测试 |
| 7 | `license status --json` 文档承诺但未实现 | 🟡 中 | ✅ 已实现 |

### 发布状态

| 项 | 结果 |
|---|---|
| tag | `v4.0.0` → CI **success** |
| release 产物 | `cortex-linux-amd64.tar.gz` / `cortex-windows-amd64.zip` / `SHA256SUMS` 均 200 |
| Windows 一行安装 | ✅ 下载 → SHA256 校验 → 安装 → `v4.0.0` |
| Linux 一行安装（真实 WSL Ubuntu） | ✅ 下载 → SHA256 校验 → 安装 → `v4.0.0` |

---

## 1. 测试场景

| 场景 | 命令 | 结果 |
|---|---|---|
| A. 仓库内源码安装 | `.\scripts\install.ps1 -InstallDir <tmp>\bin` | ✅ 成功 |
| B. 非仓库目录（模拟远程用户） | 同上，脚本复制到空目录 | ❌ **失败**（见缺陷 1） |
| C. 全新语料索引 | `cortex index <corpus>` | ✅ 3/3 成功 |
| D. 搜索验证 | `cortex search "BundleStore" -k 2` | ✅ 命中 .go 内容 |
| E. 状态查询 | `cortex status` | ✅ 27,157 文档 |
| F. 6 平台交叉编译 | `GOOS/GOARCH` 组合 | ✅ 6/6 |
| G. 打包产物 | zip/tar.gz + SHA256 | ✅ 命名与脚本期望一致 |
| H. Free 档配额 | 索引 27k 已有库 | ✅ 正确拒绝并提示升级 |

---

## 2. 缺陷详情

### 缺陷 1：官方一行安装命令不可用（阻断，未修复）

```powershell
irm https://raw.githubusercontent.com/lh123aa/cortex/main/scripts/install.ps1 | iex
```

实际行为：

```
[1/2] Downloading https://github.com/lh123aa/cortex/releases/latest/download/cortex-windows-amd64.zip
⚠️  Download failed: 远程服务器返回错误: (404) 未找到。
⚠️  Falling back to source build...
⚠️  Not a Cortex source checkout: ...
[exit 1]
```

探测结果：

| URL | HTTP |
|---|---|
| `/releases/latest` | 200（存在旧 release） |
| `/releases/latest/download/cortex-windows-amd64.zip` | **404** |
| `/releases/latest/download/SHA256SUMS` | **404** |

**根因**：仓库最新 tag 是 `v1.0.4`，当前代码从未打过 `v4.0.0` tag，所以 CI 的 release 发布作业从未为本版本执行过。

**修复方式**（需发版操作，非代码改动）：

```bash
git tag -a v4.0.0 -m "v4.0.0" origin/master~7   # 指向本次发布的提交
git push origin v4.0.0
```

或先在本地验证 build.yml 的 release 作业能跑通（workflow_dispatch）。

> ⚠️ 这是**首发前必须解决**的问题：README 首屏写的就是这条命令。

### 缺陷 2：源码构建版本显示 dev（已修）

```
$ cortex --version
cortex version dev (commit none, built unknown)
```

**根因**：`scripts/install.ps1` 和 `install.sh` 的 `go build` 没有注入版本 ldflags，而 CI 有注入。用户看到 `dev` 会认为装了个开发版。

**修复**：两个脚本都改用 `git describe --tags --abbrev=0` 取真实版本并注入。

> 刻意没有直接读 `version.go` 里的 `Version` 变量——那里是未注入时的 `dev` 默认值，读了还是 dev。

修复后：

```
cortex version v1.0.4 (commit f72620a, built 2026-11-01T13:28:25Z)
```

### 缺陷 3：脚本 exit 1 造成「安装失败」错觉（已修）

`cortex install` 把 zap 日志写到 stderr。PowerShell 对原生命令的 stderr 会记录为错误流，即使命令成功也把脚本的退出码变成 1。

**修复**：改用 `Start-Process -RedirectStandardOutput/-RedirectStandardError` 显式重定向两个流，过滤 `debug/info` 级别日志，只保留真正错误。

修复后脚本 `EXIT=0`，且能如实报出业务限制：

```
  ⚠️  cortex install exited with code 1
       ❌ document limit reached (27157 documents used of 5000 documents allowed)
```

> 顺带确认了 Free 档配额是真的在生效。

### 缺陷 4：`.go` 文件索引 100% 失败（已修 + 回归测试）

实测：

```
✅ Indexing complete! 2 indexed, 0 skipped, 1 failed
Warning: failed to index bundle.go: saving bundle failed:
  SQL logic error: table symbols has no column named file_path (1)
```

**根因**：我在性能优化时新增的 `internal/storage/bundle.go` 里，符号表列名写成了 `file_path`，而真实 schema（`internal/storage/symbols.go`）是 `path` + `language`。

这个 bug 只影响**代码文件**，因为我之前的 bundle 测试只覆盖了 `.md` 文档——测试覆盖缺口导致没发现。

**修复**：列名改为 `path, name, kind, line, signature, language`，并补 `bundle_test.go` 两个回归测试：
- `TestSaveDocumentBundleWithSymbols`：带符号的文档必须真正落库，并校验 line/path/language
- `TestSaveDocumentBundleReplacesSymbols`：重复索引时符号替换而非累积

修复后：`3 indexed, 0 skipped, 0 failed`，搜索 `BundleStore` 能命中 `.go` 内容。

### 缺陷 5：CLI 打印内部 ID 而非路径（已修）

```
1. [Score: 1.0000] Day 8-10：系统优化与迭代
   2675a7228b6a5755          ← 这是 DocumentID 哈希，用户看不懂
```

**修复**：优先用检索层带出的 `FilePath`，缺失时回查一次文档，最后才退回 ID。

修复后：

```
1. [Score: 1.0000] type BundleStore
   E:\程序\github\my-projects\Cortex\_temp\final\corpus\bundle.go
```

### 缺陷 6：过短文档从统计中彻底消失（已修 + 回归测试）

Linux 实测输出：

```
Indexing [████████████████████] 100%  3/3 · 0s
✅ Indexing complete! 2 indexed, 0 skipped, 0 failed
```

**进度显示 3/3 全部处理，汇总只有 2 个文件 — 第三个既不算成功也不算跳过，凭空消失。**

**根因**：

```go
chunks, err := ck.Chunk(string(content), path)
if err != nil || len(chunks) == 0 {
    return false, false, err   // ← 三个 flag 全 false，任何统计都不计入
}
```

低于 `chunker.min_chars`（默认 50 字符）的文档分块器返回 0 个块，
索引器返回 `(false, false, nil)` —— 不计成功、不计跳过、不计失败。

同类问题在去重分支后还有第二处（返回 `indexed=true` 但什么也没写）。

**修复**：两处都改为 `(false, true, nil)`，并输出明确警告：

```
Warning: skipped short.md: no chunks produced (content shorter than chunker min_chars threshold)
```

新增回归测试 `internal/index/short_doc_test.go`，断言不变式：

- 长文档被索引
- 短文档计入 skipped（而不是被当成成功）
- **`indexed + skipped + failed` 必须等于扫描到的文件数**

> 这个缺陷只有在真实语料（含短文件）下才会暴露。之前的测试语料都是长文档。

### 缺陷 7：`license status --json` 承诺未实现（已实现）

企业版文档里写了用 `--json` + `jq` 做到期监控，实际执行：

```
Error: unknown flag: --json
```

**修复**：实现 `--json`，输出稳定结构供监控使用：

```json
{
  "plan": "team",
  "days_left": 362,
  "expired": false,
  "perpetual": false,
  "trial": false,
  "document_limit": 1000000,
  "storage_limit_bytes": 536870912000,
  "expires_at": "2027-10-30T07:08:18Z"
}
```

`days_left` 语义：**-1 = 永久/Free**，**0 = 已过期**，正数为剩余天数。

> 特意让 Free 档报 `-1` 而不是 `0`：否则监控脚本会把「没有许可」误判成
> 「许可已过期」，触发无意义的告警。

---

## 3. 通过项详情

### 3.1 安装产物

| 项 | 值 |
|---|---|
| 二进制大小 | 35.6 MB（windows/amd64，strip + trimpath） |
| 安装后版本 | `v1.0.4`（真实 git tag + commit + 构建时间） |
| 自动配置 | 写入 AI 工具 MCP 配置（带备份） |
| PATH | 自动追加用户 PATH |

### 3.2 6 平台交叉编译

| 平台 | 体积 | 状态 |
|---|---:|:--:|
| linux/amd64 | 35.1 MB | ✅ |
| linux/arm64 | 32.9 MB | ✅ |
| darwin/amd64 | 35.8 MB | ✅ |
| darwin/arm64 | 33.8 MB | ✅ |
| windows/amd64 | 35.6 MB | ✅ |
| windows/arm64 | 32.9 MB | ✅ |

### 3.3 打包产物与脚本期望一致性

| 产物 | zip 内容 | 脚本期望 | 一致 |
|---|---|---|---|
| `cortex-windows-amd64.zip` | `cortex.exe` | `cortex.exe` | ✅ |
| `cortex-linux-amd64.tar.gz` | `cortex` | `cortex` | ✅ |

### 3.4 索引与检索（独立语料）

语料：3 个文件（README.md 37KB + 2 篇文档/代码）

| 指标 | 值 |
|---|---|
| 索引结果 | 3 indexed, 0 skipped, 0 failed |
| 索引耗时 | 45 ms |
| 分块数 | 47 |
| 检索 | 命中 `.go` 中的 `BundleStore`，score 1.0000 |

### 3.5 Linux 真实环境验证（WSL2 Ubuntu）

在真实 Linux（WSL2 Ubuntu，无 Go 环境）执行官方安装脚本，验证纯下载路径：

```bash
curl -fsSL -o install.sh https://raw.githubusercontent.com/lh123aa/cortex/master/scripts/install.sh
INSTALL_DIR=/tmp/ctest/bin sh install.sh
```

结果：

```
[INFO] 检测到环境：linux/amd64
[INFO] 下载 https://github.com/lh123aa/cortex/releases/latest/download/cortex-linux-amd64.tar.gz
[INFO] SHA256 校验通过
[INFO] 已安装到 /tmp/ctest/bin/cortex
cortex version v4.0.0 (commit a07d18d, built 2026-11-01T10:45:14Z)
```

功能验证（3 类文件：Markdown × 2 + Go × 1）：

| 项 | 结果 |
|---|---|
| 索引 | **3 indexed, 0 skipped, 0 failed**（11 ms） |
| 搜索 Markdown token | ✅ 命中，路径 `corpus/b.md` |
| 搜索 Go token | ✅ 命中（走符号索引路径），路径 `corpus/c.go` |
| 二进制 | linux/amd64，纯静态无依赖 |

> 这轮 Linux 测试直接暴露了缺陷 6：进度条显示 3/3，汇总却是
> 「2 indexed, 0 skipped, 0 failed」——第三个文件凭空消失。
> 只有在真实环境跑才发现的。

### 3.6 企业版安装文档

新增 `Cortex-ee/docs/INSTALL.md`，覆盖：
- 开源版 vs 企业版的版本关系与并排安装
- 许可激活三种场景（在线 / 离线 / 容器）
- 功能验证（`/ee/audit` 门控行为：无 feature 时返回 404 而非 403）
- 到期监控（`license status --json` + `days_left` 语义）
- 升级路径与常见问题

---

## 4. 待办

| 优先级 | 事项 | 状态 |
|---|---|---|
| 🔴 P0 | 打 `v4.0.0` tag 触发 release 构建 | ✅ 完成 |
| 🟠 P1 | 用远程 URL 真实跑一遍安装 | ✅ 完成（Windows + Linux 双平台） |
| 🟡 P2 | 企业版独立安装说明 | ✅ 完成 |
| 🟡 P2 | Linux `install.sh` 真实环境验证 | ✅ 完成 |
| 🟢 P3 | macOS 环境验证（当前无可用机器） | 待办 |
| 🟢 P3 | 企业版独立 CI（当前手动构建） | 待办 |

## 5. 复现命令

```powershell
# 全新安装
$t = "$PWD\_temp\final"
Remove-Item $t -Recurse -Force -ErrorAction SilentlyContinue
$env:CGO_ENABLED = '0'
.\scripts\install.ps1 -InstallDir "$t\bin" -DocDir "$t\corpus"

# 独立索引（隔离配置，不碰真实库）
cortex -c "$t\config.yaml" index "$t\corpus"
cortex -c "$t\config.yaml" search "BundleStore" -k 2

# 6 平台构建
foreach ($p in @('linux/amd64','linux/arm64','darwin/amd64','darwin/arm64','windows/amd64','windows/arm64')) {
  $os,$arch = $p -split '/'
  $env:GOOS=$os; $env:GOARCH=$arch; $env:CGO_ENABLED='0'
  go build -trimpath -ldflags="-s -w" -o "_temp\dist\$os-$arch\cortex$(if($os -eq 'windows'){'.exe'})" ./cmd/cortex
}
```
