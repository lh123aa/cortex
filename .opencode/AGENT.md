# Cortex - Agent Guide

## 项目结构（已清理）

```
E:\程序\Cortex/
├── cmd/                    # 入口点
├── internal/               # 核心包
├── bin/                    # 编译输出（cortex.exe 等）
├── docs/
│   ├── demo/               # 演示文档
│   ├── planning/           # 迭代计划/PRD/评估报告
│   └── private/            # 商业/客户信息（.gitignore 排除，不进 Git）
├── deploy/                 # 部署配置（prometheus.yml）
├── scripts/                # 工具脚本
├── test_framework/         # 测试框架
├── downloads/              # 发布包
├── _temp/                  # 临时文件（用完即删）
├── go.mod / go.sum
├── Makefile
├── Dockerfile
├── docker-compose.yml
└── README.md
```

## 根目录文件保留规则

根目录只允许以下文件存在：
- `cmd/` `internal/` — 核心源码
- `go.mod` `go.sum` `Makefile` — 构建配置
- `README.md` `README.en.md` — 项目门面
- `CHANGELOG.md` `CONTRIBUTING.md` `CODE_OF_CONDUCT.md` `SECURITY.md` — 社区文档
- `Dockerfile` `docker-compose.yml` — 容器化（根目录是标准）
- `.gitignore` `.github/` `.vscode/` `.opencode/` — 配置/CI

## 禁止行为（强制规则）

1. **编译产物不进根目录** — `make build` 已配置输出到 `bin/`，任何 `.exe` 出现在根目录即为违规
2. **规划文档不进根目录** — PRD、迭代计划、评估报告 → `docs/planning/`
3. **商业保密文件不进 Git** — COMMERCIAL_PLAN.md 等 → `docs/private/`（.gitignore 已排除）
4. **临时文件用完必删** — 测试/调试文件 → `_temp/`，完成后立即 `rm -rf _temp/`
5. **不产生孤立文件** — AI 生成的文件必须写入已有目录结构

## 构建命令

```bash
make build          # 编译到 bin/cortex
make test           # 运行测试
make clean          # 清理 bin/ 和数据库
```

## MCP 配置

Cortex 已注册为 MCP 工具，工具名: `cortex`
可用 MCP 工具: `cortex_search`, `cortex_context`, `cortex_memory_write`, `cortex_memory_search`, `cortex_memory_delete`, `cortex_memory_delete_batch`
