package ext

import (
	"errors"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lh123aa/cortex/pkg/license"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeExt struct {
	name     string
	features []string
	mcpCalls int
	httpCall int
	failMCP  bool
	failHTTP bool
}

func (f *fakeExt) Name() string               { return f.name }
func (f *fakeExt) RequiredFeatures() []string { return f.features }
func (f *fakeExt) RegisterMCP(s *mcp.Server) error {
	f.mcpCalls++
	if f.failMCP {
		return errors.New("boom")
	}
	return nil
}
func (f *fakeExt) RegisterHTTP(r *gin.Engine) error {
	f.httpCall++
	if f.failHTTP {
		return errors.New("boom")
	}
	return nil
}

func newMCP() *mcp.Server {
	return mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0"}, nil)
}

func silentLogger() {
	SetLogger(func(string, ...any) {})
}

func TestRegisterAndList(t *testing.T) {
	ResetForTest()
	silentLogger()
	defer ResetForTest()

	Register(&fakeExt{name: "zeta"})
	Register(&fakeExt{name: "alpha"})
	Register(nil)

	got := Registered()
	if len(got) != 2 {
		t.Fatalf("registered = %d, want 2", len(got))
	}
	if got[0].Name() != "alpha" || got[1].Name() != "zeta" {
		t.Fatalf("extensions should be sorted by name, got %s,%s", got[0].Name(), got[1].Name())
	}
}

// TestFeatureGating 只有许可证包含所需 feature 时才启用扩展。
func TestFeatureGating(t *testing.T) {
	ResetForTest()
	silentLogger()
	defer ResetForTest()

	free := &fakeExt{name: "audit", features: []string{"audit-log"}}
	basic := &fakeExt{name: "core-extra"}
	Register(free)
	Register(basic)

	// 无许可：需要 feature 的被跳过
	enabled := RegisterMCP(newMCP())
	if len(enabled) != 1 || enabled[0] != "core-extra" {
		t.Fatalf("without license only core-extra should run, got %v", enabled)
	}
	if free.mcpCalls != 0 {
		t.Fatal("free extension should not be called without license")
	}

	// Team 许可包含 audit-log
	ResetForTest()
	silentLogger()
	Register(free)
	SetLicense(&license.Info{Plan: license.PlanTeam})
	enabled = RegisterMCP(newMCP())
	if len(enabled) != 1 || enabled[0] != "audit" {
		t.Fatalf("team license should enable audit, got %v", enabled)
	}
}

// TestEnterpriseWildcard Enterprise 的 "*" 应放行任意 feature。
func TestEnterpriseWildcard(t *testing.T) {
	ResetForTest()
	silentLogger()
	defer ResetForTest()

	e := &fakeExt{name: "sso", features: []string{"sso-oidc", "something-not-listed"}}
	Register(e)
	SetLicense(&license.Info{Plan: license.PlanEnterprise})

	enabled := RegisterHTTP(gin.New())
	if len(enabled) != 1 {
		t.Fatalf("enterprise wildcard should enable extension, got %v", enabled)
	}
}

// TestExplicitFeatureInPayload 许可里显式列出的 feature 也应生效。
func TestExplicitFeatureInPayload(t *testing.T) {
	ResetForTest()
	silentLogger()
	defer ResetForTest()

	e := &fakeExt{name: "extra", features: []string{"custom-feature"}}
	Register(e)
	SetLicense(&license.Info{Plan: license.PlanPro, Features: []string{"custom-feature"}})

	if enabled := RegisterMCP(newMCP()); len(enabled) != 1 {
		t.Fatalf("payload feature should enable extension, got %v", enabled)
	}
}

// TestExtensionFailureIsIsolated 单个扩展失败不能拖垮其它扩展。
func TestExtensionFailureIsIsolated(t *testing.T) {
	ResetForTest()
	silentLogger()
	defer ResetForTest()

	bad := &fakeExt{name: "bad", failMCP: true, failHTTP: true}
	good := &fakeExt{name: "good"}
	Register(bad)
	Register(good)

	mcpEnabled := RegisterMCP(newMCP())
	if len(mcpEnabled) != 1 || mcpEnabled[0] != "good" {
		t.Fatalf("failed extension should be skipped, got %v", mcpEnabled)
	}
	httpEnabled := RegisterHTTP(gin.New())
	if len(httpEnabled) != 1 || httpEnabled[0] != "good" {
		t.Fatalf("failed extension should be skipped for HTTP, got %v", httpEnabled)
	}
}

func TestResetClearsState(t *testing.T) {
	ResetForTest()
	silentLogger()
	defer ResetForTest()

	Register(&fakeExt{name: "x"})
	SetLicense(&license.Info{Plan: license.PlanPro})
	ResetForTest()

	if len(Registered()) != 0 {
		t.Fatal("ResetForTest should clear extensions")
	}
	if enabled := RegisterMCP(newMCP()); len(enabled) != 0 {
		t.Fatalf("no extensions expected, got %v", enabled)
	}
}
