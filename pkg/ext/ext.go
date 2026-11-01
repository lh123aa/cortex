// Package ext 是 Cortex 开源核心与闭源企业版之间的唯一边界。
//
// 设计目标（open-core）：
//   - 开源仓库（本仓库）只包含「标准能力」：索引、检索、MCP、REST、控制台、License 校验。
//   - 企业版放在私有仓库，通过实现 Extension 接口把高级能力注入核心，
//     不需要 fork 核心代码，也不需要把闭源代码放进公库。
//   - 核心在启动时调用 Register*，按许可证里的 feature 决定是否启用。
//
// 约束：
//   - 本包不能 import 企业版任何代码（私有仓库反向依赖它）。
//   - 扩展失败不能拖垮核心：任一扩展注册出错只记录日志并跳过。
package ext

import (
	"fmt"
	"log"
	"sort"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/lh123aa/cortex/pkg/license"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Extension 是企业版实现的能力扩展。
type Extension interface {
	// Name 是扩展名，用于日志与错误信息（例如 "audit-log"）。
	Name() string
	// RequiredFeatures 返回该扩展需要的许可 feature。
	// 为空表示始终可用（通常只有开源核心自己用）。
	RequiredFeatures() []string
	// RegisterMCP 向 MCP server 注册额外工具。
	RegisterMCP(s *mcp.Server) error
	// RegisterHTTP 向 REST router 注册额外路由。
	RegisterHTTP(r *gin.Engine) error
}

var (
	mu         sync.RWMutex
	extensions []Extension
	// licenseInfo 是解析后的当前许可，由 SetLicense 注入。
	licenseInfo *license.Info
	// logger 允许宿主替换日志实现。
	logger = log.Printf
)

// SetLogger 替换日志函数（便于测试静默）。
func SetLogger(fn func(string, ...any)) {
	if fn == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	logger = fn
}

// Register 注册一个扩展。通常在私有仓库的 init() 中调用。
func Register(e Extension) {
	if e == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	extensions = append(extensions, e)
	sort.Slice(extensions, func(i, j int) bool {
		return extensions[i].Name() < extensions[j].Name()
	})
}

// Registered 返回已注册扩展（按名称排序）。
func Registered() []Extension {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Extension, len(extensions))
	copy(out, extensions)
	return out
}

// SetLicense 注入当前许可证。传 nil 表示 Free。
func SetLicense(info *license.Info) {
	mu.Lock()
	defer mu.Unlock()
	licenseInfo = info
}

func currentLicense() *license.Info {
	mu.RLock()
	defer mu.RUnlock()
	return licenseInfo
}

// enabled 判断扩展是否满足许可要求。
func enabled(e Extension) (bool, string) {
	req := e.RequiredFeatures()
	if len(req) == 0 {
		return true, ""
	}
	info := currentLicense()
	if info == nil {
		return false, "no license"
	}
	missing := make([]string, 0, len(req))
	for _, f := range req {
		if !info.HasFeature(f) {
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		return false, fmt.Sprintf("missing feature(s): %v", missing)
	}
	return true, ""
}

// RegisterMCP 把所有满足许可的扩展注册到 MCP server。
// 返回成功启用的扩展名列表，便于启动日志展示。
func RegisterMCP(s *mcp.Server) []string {
	var enabledNames []string
	for _, e := range Registered() {
		ok, reason := enabled(e)
		if !ok {
			logf("skip extension %q: %s", e.Name(), reason)
			continue
		}
		if err := e.RegisterMCP(s); err != nil {
			logf("extension %q failed to register MCP tools: %v", e.Name(), err)
			continue
		}
		enabledNames = append(enabledNames, e.Name())
	}
	return enabledNames
}

// RegisterHTTP 把所有满足许可的扩展注册到 REST router。
func RegisterHTTP(r *gin.Engine) []string {
	var enabledNames []string
	for _, e := range Registered() {
		ok, reason := enabled(e)
		if !ok {
			logf("skip extension %q: %s", e.Name(), reason)
			continue
		}
		if err := e.RegisterHTTP(r); err != nil {
			logf("extension %q failed to register HTTP routes: %v", e.Name(), err)
			continue
		}
		enabledNames = append(enabledNames, e.Name())
	}
	return enabledNames
}

func logf(format string, args ...any) {
	mu.RLock()
	fn := logger
	mu.RUnlock()
	fn(format, args...)
}

// ResetForTest 清空注册表（仅测试使用）。
func ResetForTest() {
	mu.Lock()
	defer mu.Unlock()
	extensions = nil
	licenseInfo = nil
}
