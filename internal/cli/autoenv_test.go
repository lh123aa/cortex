package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveConfigPath_HomeDir(t *testing.T) {
	path := resolveConfigPath("~/some/dir/config.json")
	if path == "" {
		t.Fatal("expected non-empty path")
	}
	home, _ := os.UserHomeDir()
	if !strings.HasPrefix(path, home) {
		t.Errorf("expected path to start with home dir %q, got %q", home, path)
	}
	if !strings.Contains(path, "some") || !strings.Contains(path, "config.json") {
		t.Errorf("expected path to contain 'some' and 'config.json', got %q", path)
	}
}

func TestResolveConfigPath_Relative(t *testing.T) {
	origDir, _ := os.Getwd()
	path := resolveConfigPath("./test.json")
	if path == "" {
		t.Fatal("expected non-empty path")
	}
	expected := filepath.Join(origDir, "test.json")
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestResolveConfigPath_Raw(t *testing.T) {
	path := resolveConfigPath("/absolute/path/config.json")
	if path != "/absolute/path/config.json" {
		t.Errorf("expected '/absolute/path/config.json', got %q", path)
	}
}

func TestRegisterTool_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "mcp.json")

	err := registerTool(configPath, "TestTool")
	if err != nil {
		t.Fatalf("registerTool failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	mcpServers, ok := result["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatal("expected mcpServers key")
	}

	cortex, ok := mcpServers["cortex"].(map[string]interface{})
	if !ok {
		t.Fatal("expected cortex entry in mcpServers")
	}

	if cortex["command"] != "cortex" {
		t.Errorf("expected command 'cortex', got %v", cortex["command"])
	}
}

func TestRegisterTool_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "mcp.json")

	existing := map[string]interface{}{
		"otherTool": map[string]interface{}{
			"command": "other",
			"args":    []interface{}{"run"},
		},
	}
	initialData, _ := json.MarshalIndent(existing, "", "  ")
	if err := os.WriteFile(configPath, initialData, 0644); err != nil {
		t.Fatalf("failed to write initial config: %v", err)
	}

	err := registerTool(configPath, "TestTool")
	if err != nil {
		t.Fatalf("registerTool failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if _, ok := result["otherTool"]; !ok {
		t.Error("expected otherTool to still exist at root level")
	}

	mcpServers, ok := result["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatal("expected mcpServers key")
	}

	if _, ok := mcpServers["cortex"]; !ok {
		t.Error("expected cortex entry to be added to mcpServers")
	}
}

func TestRegisterTool_BackupCreated(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "mcp.json")

	initialData := []byte(`{"existing": true}`)
	if err := os.WriteFile(configPath, initialData, 0644); err != nil {
		t.Fatalf("failed to write initial config: %v", err)
	}

	err := registerTool(configPath, "TestTool")
	if err != nil {
		t.Fatalf("registerTool failed: %v", err)
	}

	backupPath := configPath + ".bak"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("expected backup file to exist")
	}

	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("failed to read backup: %v", err)
	}
	if string(backupData) != string(initialData) {
		t.Error("backup content does not match original")
	}
}

func TestRegisterTool_SkipTools(t *testing.T) {
	tmpDir := t.TempDir()

	claudePath := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudePath, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	claudeConfig := filepath.Join(claudePath, "claude_desktop_config.json")
	os.WriteFile(claudeConfig, []byte(`{}`), 0644)

	cursorPath := filepath.Join(tmpDir, ".cursor")
	if err := os.MkdirAll(cursorPath, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	cursorConfig := filepath.Join(cursorPath, "mcp.json")
	os.WriteFile(cursorConfig, []byte(`{}`), 0644)
}

func TestResolveConfigPath_EmptyOnError(t *testing.T) {
	original := "~/nonexistent_file.json"
	path := resolveConfigPath(original)
	if path == "" {
		t.Fatal("expected non-empty path even when file doesn't exist")
	}
	home, _ := os.UserHomeDir()
	if !strings.HasPrefix(path, home) {
		t.Errorf("expected path to start with home dir")
	}
}
