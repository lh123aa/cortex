package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type aiTool struct {
	Name       string
	ConfigPath string
}

type mcpConfigWrapper struct {
	MCPServers map[string]mcpServerConfig `json:"mcpServers"`
}

type mcpServerConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

var knownTools = []aiTool{
	{
		Name:       "Claude Code",
		ConfigPath: "~/.claude/claude_desktop_config.json",
	},
	{
		Name:       "Cursor",
		ConfigPath: "~/.cursor/mcp.json",
	},
	{
		Name:       "OpenCode",
		ConfigPath: "./.opencode.json",
	},
}

func registerAITools(skipTools []string) []string {
	skipSet := make(map[string]bool)
	for _, t := range skipTools {
		skipSet[strings.ToLower(t)] = true
	}

	var registered []string

	for _, tool := range knownTools {
		if skipSet[strings.ToLower(tool.Name)] {
			continue
		}

		path := resolveConfigPath(tool.ConfigPath)
		if path == "" {
			continue
		}

		dir := filepath.Dir(path)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		if err := registerTool(path, tool.Name); err != nil {
			continue
		}

		registered = append(registered, tool.Name)
	}

	registered = append(registered, registerTrae()...)

	return registered
}

func registerTool(configPath, toolName string) error {
	existing := make(map[string]interface{})

	if data, err := os.ReadFile(configPath); err == nil && len(data) > 0 {
		json.Unmarshal(data, &existing)
	}

	backupPath := configPath + ".bak"
	if data, err := os.ReadFile(configPath); err == nil {
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			os.WriteFile(backupPath, data, 0644)
		}
	}

	cortexEntry := map[string]interface{}{
		"command": "cortex",
		"args":    []interface{}{"mcp"},
	}

	if mcpServers, ok := existing["mcpServers"].(map[string]interface{}); ok {
		mcpServers["cortex"] = cortexEntry
	} else {
		existing["mcpServers"] = map[string]interface{}{
			"cortex": cortexEntry,
		}
	}

	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config for %s: %w", toolName, err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config for %s: %w", toolName, err)
	}

	return nil
}

func registerTrae() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	traeDir := filepath.Join(home, ".trae")
	entries, err := os.ReadDir(traeDir)
	if err != nil {
		return nil
	}

	var registered []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		mcpJSON := filepath.Join(traeDir, entry.Name(), "mcp.json")
		if _, err := os.Stat(mcpJSON); os.IsNotExist(err) {
			continue
		}
		if err := registerTool(mcpJSON, "Trae"); err == nil {
			registered = append(registered, fmt.Sprintf("Trae (%s)", entry.Name()))
		}
	}

	return registered
}

func resolveConfigPath(raw string) string {
	if strings.HasPrefix(raw, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		return filepath.Join(home, raw[2:])
	}
	if strings.HasPrefix(raw, "./") {
		wd, err := os.Getwd()
		if err != nil {
			return ""
		}
		return filepath.Join(wd, raw[2:])
	}
	return raw
}
