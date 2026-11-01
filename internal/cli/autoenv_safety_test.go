package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestRegisterToolPreservesExistingServers 回归测试：
// 注册 Cortex 时绝不能丢掉用户已有的其它 MCP server 配置。
func TestRegisterToolPreservesExistingServers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")

	original := `{
  "mcpServers": {
    "existing": {"command": "echo", "args": ["hello"]},
    "other":    {"command": "node", "args": ["server.js"]}
  },
  "globalShortcut": "Cmd+Shift+C"
}`
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	if err := registerTool(path, "TestTool"); err != nil {
		t.Fatalf("registerTool failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("result is not valid JSON: %v\n%s", err, data)
	}

	servers, ok := cfg["mcpServers"].(map[string]any)
	if !ok {
		t.Fatalf("mcpServers missing: %v", cfg)
	}
	for _, name := range []string{"existing", "other", "cortex"} {
		if _, ok := servers[name]; !ok {
			t.Errorf("mcpServers lost entry %q; got %v", name, keysOf(servers))
		}
	}
	if cfg["globalShortcut"] != "Cmd+Shift+C" {
		t.Errorf("unrelated top-level key was dropped: %v", cfg["globalShortcut"])
	}
}

// TestRegisterToolHandlesBOMConfig 回归测试：
// Windows 上 PowerShell/记事本写入的 JSON 常带 UTF-8 BOM，encoding/json 会拒绝解析。
// 早期实现忽略该错误 → existing 为空 → 写回时把用户其它 MCP 配置全部抹掉。
func TestRegisterToolHandlesBOMConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")

	body := `{"mcpServers":{"existing":{"command":"echo"}}}`
	withBOM := append([]byte{0xEF, 0xBB, 0xBF}, []byte(body)...)
	if err := os.WriteFile(path, withBOM, 0644); err != nil {
		t.Fatal(err)
	}

	if err := registerTool(path, "TestTool"); err != nil {
		t.Fatalf("registerTool should tolerate BOM, got: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(stripBOM(data), &cfg); err != nil {
		t.Fatalf("result is not valid JSON: %v\n%s", err, data)
	}
	servers, _ := cfg["mcpServers"].(map[string]any)
	if _, ok := servers["existing"]; !ok {
		t.Fatalf("existing server was lost when config had a BOM; got %v", keysOf(servers))
	}
	if _, ok := servers["cortex"]; !ok {
		t.Fatalf("cortex entry not added; got %v", keysOf(servers))
	}
}

// TestRegisterToolRefusesToClobberInvalidJSON 确保坏文件绝不被覆盖。
func TestRegisterToolRefusesToClobberInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")

	broken := `{"mcpServers": {"existing": {"command": "echo"` // 截断的 JSON
	if err := os.WriteFile(path, []byte(broken), 0644); err != nil {
		t.Fatal(err)
	}

	err := registerTool(path, "TestTool")
	if err == nil {
		t.Fatal("expected error when existing config is invalid JSON")
	}

	// 原文件必须保持原样
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(data) != broken {
		t.Fatalf("invalid config was overwritten; got %q", data)
	}
}

// TestRegisterToolCreatesBackup 验证写入前会备份原始内容。
func TestRegisterToolCreatesBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	original := `{"mcpServers":{"a":{"command":"x"}}}`
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	if err := registerTool(path, "TestTool"); err != nil {
		t.Fatal(err)
	}
	backup, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatalf("expected backup file: %v", err)
	}
	if string(backup) != original {
		t.Errorf("backup content = %q, want %q", backup, original)
	}
}

// TestRegisterToolRollsBackup 第二次注册不应覆盖掉原始备份。
func TestRegisterToolRollsBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	original := `{"mcpServers":{"a":{"command":"x"}}}`
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	if err := registerTool(path, "TestTool"); err != nil {
		t.Fatal(err)
	}
	if err := registerTool(path, "TestTool"); err != nil {
		t.Fatal(err)
	}

	// 第一次的备份应滚动为 .bak.1
	rolled, err := os.ReadFile(path + ".bak.1")
	if err != nil {
		t.Fatalf("expected rolled backup .bak.1: %v", err)
	}
	if string(rolled) != original {
		t.Errorf("rolled backup = %q, want original %q", rolled, original)
	}
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
