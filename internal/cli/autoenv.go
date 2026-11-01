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

	var originalData []byte
	if data, err := os.ReadFile(configPath); err == nil {
		originalData = data
		if len(data) > 0 {
			cleaned := stripBOM(data)
			if err := json.Unmarshal(cleaned, &existing); err != nil {
				// 关键安全行为：解析失败时绝不覆盖用户原文件。
				// 早期实现忽略了这个错误，导致「带 BOM 的配置」或格式不规范的配置
				// 被当成空对象，随后写回时把用户已有的其它 MCP server 全部抹掉。
				return fmt.Errorf("refusing to overwrite %s: existing config is not valid JSON (%w)", configPath, err)
			}
		}
	}

	// 备份：每次写入前保存上一版（保留 .bak.1 这种滚动备份，避免第二次注册覆盖掉原始备份）。
	backupPath := configPath + ".bak"
	if len(originalData) > 0 {
		if _, err := os.Stat(backupPath); err == nil {
			// 已有备份则滚动，防止用户真正的原始配置被覆盖
			_ = os.Rename(backupPath, backupPath+".1")
		}
		if err := os.WriteFile(backupPath, originalData, 0644); err != nil {
			return fmt.Errorf("failed to back up config for %s: %w", toolName, err)
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
	data = append(data, '\n')

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config for %s: %w", toolName, err)
	}

	return nil
}

// stripBOM 去掉 UTF-8 BOM。
// Windows 上 PowerShell 5 的 `Set-Content -Encoding UTF8`、记事本等工具默认会写 BOM，
// 而 encoding/json 不接受 BOM 开头的输入 —— 这是「用户配置被清空」的真实触发路径。
func stripBOM(data []byte) []byte {
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return data[3:]
	}
	return data
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
