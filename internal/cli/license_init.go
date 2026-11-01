package cli

import (
	"github.com/lh123aa/cortex/pkg/ext"
	"github.com/lh123aa/cortex/pkg/license"
)

// initLicenseExt 把本地许可证注入扩展注册表。
//
// 开源核心只负责「校验许可 + 决定哪些扩展可用」；
// 具体的高级能力由私有仓库通过 ext.Register 提供。
// 这里必须在构造 MCP/REST server 之前调用，否则扩展会因「没有许可」被跳过。
func initLicenseExt() {
	info, err := license.Current()
	if err != nil {
		// 许可文件损坏不应阻断启动：按 Free 处理，扩展自然不启用。
		return
	}
	ext.SetLicense(info)
}
