// Package version 是 Cortex 全项目唯一的版本号来源。
//
// 版本信息在构建时通过 -ldflags 注入，例如：
//
//	go build -ldflags "-X github.com/lh123aa/cortex/internal/version.Version=v4.0.0 \
//	  -X github.com/lh123aa/cortex/internal/version.Commit=$(git rev-parse --short HEAD) \
//	  -X github.com/lh123aa/cortex/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
//	  ./cmd/cortex
//
// 未注入时回退到开发默认值（dev/none/unknown），保证本地 go run/go test 可用。
package version

import "fmt"

// 构建时可通过 -ldflags 覆盖的版本变量。
var (
	// Version 是发布版本号，例如 "v4.0.0"；未注入时为 "dev"。
	Version = "dev"
	// Commit 是构建所用的 git commit（短 SHA）；未注入时为 "none"。
	Commit = "none"
	// Date 是构建时间（RFC3339 UTC）；未注入时为 "unknown"。
	Date = "unknown"
)

// Full 返回人类可读的完整版本信息，
// 例如 "v4.0.0 (commit abc1234, built 2026-10-22T10:00:00Z)"。
func Full() string {
	return fmt.Sprintf("%s (commit %s, built %s)", Version, Commit, Date)
}
