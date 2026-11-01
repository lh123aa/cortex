// Package app 是 Cortex 的公开启动入口。
//
// 为什么需要它：Go 的 internal/ 包不允许外部模块导入，
// 而闭源企业版（独立模块）需要「注册扩展 → 启动核心」的能力。
// 所以把入口收敛成这一行公开 API：
//
//	import (
//	    _ "github.com/lh123aa/cortex-ee/ee/audit" // 扩展在 init() 里 ext.Register
//	    "github.com/lh123aa/cortex/app"
//	)
//	func main() { app.Run() }
package app

import "github.com/lh123aa/cortex/internal/cli"

// Run 启动 Cortex CLI（与官方 cmd/cortex 行为完全一致）。
// 调用前，企业版扩展应已在 init() 中通过 pkg/ext.Register 注册。
func Run() {
	cli.Execute()
}
