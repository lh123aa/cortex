package embedding

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"gopkg.in/yaml.v3"
)

// SetupConfig 交互式配置结果
type SetupConfig struct {
	Provider   string
	AutoUpdate bool
	// 各 Provider 的配置
	APIKey    string
	SecretKey string
	Model     string
	Dimension int
	BaseURL   string
	OllamaURL string
	OllamaModel string
}

// RunSetupWizard 运行交互式配置向导
func RunSetupWizard() (*SetupConfig, error) {
	cfg := &SetupConfig{
		AutoUpdate:  true,
		OllamaURL:   "http://localhost:11434",
		OllamaModel: "all-minilm",
	}

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║       🧠 Cortex Embedding 配置向导       ║")
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Println()

	// 1. 检测网络
	online := DetectNetwork()
	if online {
		fmt.Println("🌐 网络状态: 已连接 ✓")
	} else {
		fmt.Println("🌐 网络状态: 离线（仅支持本地模式）")
	}
	fmt.Println()

	// 2. 选择 Provider
	providerOpts := buildProviderOptions(online)
	selected := &survey.Select{
		Message: "请选择 Embedding 提供商:",
		Options: providerOpts.labels,
		Description: func(value string, idx int) string {
			if idx >= 0 && idx < len(providerOpts.descriptions) {
				return providerOpts.descriptions[idx]
			}
			return ""
		},
	}
	var providerIdx int
	if err := survey.AskOne(selected, &providerIdx); err != nil {
		return nil, err
	}

	selectedProvider := providerOpts.providers[providerIdx]
	cfg.Provider = selectedProvider.ID
	fmt.Printf("   → 已选择: %s\n\n", selectedProvider.Name)

	// 3. 按 Provider 类型收集参数
	switch cfg.Provider {
	case "none":
		// 无需配置
		fmt.Println("📝 FTS5-only 模式，无需额外配置 ✓")

	case "ollama":
		if err := cfg.promptOllamaConfig(); err != nil {
			return nil, err
		}

	default:
		// API 类型的 Provider
		if err := cfg.promptAPIConfig(selectedProvider); err != nil {
			return nil, err
		}
	}

	// 4. 自动更新检测
	if online && cfg.Provider != "none" && cfg.Provider != "ollama" {
		autoUpdate := false
		prompt := &survey.Confirm{
			Message: "是否启用自动检测最新模型？（联网时检查更新）",
			Default: true,
		}
		survey.AskOne(prompt, &autoUpdate)
		cfg.AutoUpdate = autoUpdate
	}

	fmt.Println()
	return cfg, nil
}

// providerOptions 选项列表
type providerOptions struct {
	labels       []string
	descriptions []string
	providers    []ProviderInfo
}

func buildProviderOptions(online bool) *providerOptions {
	opts := &providerOptions{}

	// 本地模式始终可选
	for _, p := range RegisteredProviders {
		if p.Category == "local" {
			label := fmt.Sprintf("  📦 %s  — %s", p.Name, p.Description)
			if p.ID == "none" {
				label = "  ⚡ 纯离线模式  — " + p.Description
			}
			opts.labels = append(opts.labels, label)
			opts.descriptions = append(opts.descriptions, p.Description)
			opts.providers = append(opts.providers, p)
		}
	}

	if !online {
		return opts
	}

	// 在线模式下显示国际和国内分类
	internals := GetProvidersByCategory("international")
	if len(internals) > 0 {
		opts.labels = append(opts.labels, "──────────────── 国外 API ────────────────")
		opts.descriptions = append(opts.descriptions, "")
		opts.providers = append(opts.providers, ProviderInfo{})
		for _, p := range internals {
			opts.labels = append(opts.labels, fmt.Sprintf("  🌍 %s  — %s", p.Name, p.Description))
			opts.descriptions = append(opts.descriptions, p.Description)
			opts.providers = append(opts.providers, p)
		}
	}

	domestics := GetProvidersByCategory("domestic")
	if len(domestics) > 0 {
		opts.labels = append(opts.labels, "──────────────── 国内 API ────────────────")
		opts.descriptions = append(opts.descriptions, "")
		opts.providers = append(opts.providers, ProviderInfo{})
		for _, p := range domestics {
			opts.labels = append(opts.labels, fmt.Sprintf("  🏠 %s  — %s", p.Name, p.Description))
			opts.descriptions = append(opts.descriptions, p.Description)
			opts.providers = append(opts.providers, p)
		}
	}

	return opts
}

func (cfg *SetupConfig) promptOllamaConfig() error {
	// Ollama URL
	urlPrompt := &survey.Input{
		Message: "Ollama 服务地址:",
		Default: cfg.OllamaURL,
		Help:    "默认 http://localhost:11434",
	}
	survey.AskOne(urlPrompt, &cfg.OllamaURL)

	// Ollama 模型
	p := GetProviderByID("ollama")
	if p == nil {
		return fmt.Errorf("ollama provider not found in registry")
	}
	modelLabels := make([]string, len(p.Models))
	for i, m := range p.Models {
		defaultMark := ""
		if m.IsDefault {
			defaultMark = " ★推荐"
		}
		modelLabels[i] = fmt.Sprintf("%s  ( %d维 · %s )%s", m.Name, m.Dimension, m.Description, defaultMark)
	}
	modelIdx := 0
	modelPrompt := &survey.Select{
		Message: "选择模型:",
		Options: modelLabels,
	}
	survey.AskOne(modelPrompt, &modelIdx)
	cfg.OllamaModel = p.Models[modelIdx].ID
	cfg.Dimension = p.Models[modelIdx].Dimension

	return nil
}

func (cfg *SetupConfig) promptAPIConfig(provider ProviderInfo) error {
	fmt.Printf("🔑 配置 %s API\n\n", provider.Name)

	// API Key
	var apiKey string
	keyPrompt := &survey.Password{
		Message: "API Key:",
		Help:    "输入你的 API Key，输入内容将被隐藏",
	}
	if err := survey.AskOne(keyPrompt, &apiKey, survey.WithValidator(survey.Required)); err != nil {
		return err
	}
	cfg.APIKey = strings.TrimSpace(apiKey)

	// 百度需要额外的 Secret Key
	if provider.NeedsSecret {
		var secretKey string
		secretPrompt := &survey.Password{
			Message: "Secret Key:",
			Help:    "百度文心需要额外的 Secret Key",
		}
		if err := survey.AskOne(secretPrompt, &secretKey, survey.WithValidator(survey.Required)); err != nil {
			return err
		}
		cfg.SecretKey = strings.TrimSpace(secretKey)
	}

	// 自定义 Base URL（可选）
	if provider.NeedsURL {
		var baseURL string
		urlPrompt := &survey.Input{
			Message: "API 地址（可选，留空使用默认）:",
			Default: getDefaultBaseURL(provider.ID),
			Help:    "如需自定义 API 代理地址可修改此项",
		}
		survey.AskOne(urlPrompt, &baseURL)
		cfg.BaseURL = strings.TrimSpace(baseURL)
	}

	// 选择模型
	if len(provider.Models) > 0 {
		modelLabels := make([]string, len(provider.Models))
		for i, m := range provider.Models {
			defaultMark := ""
			if m.IsDefault {
				defaultMark = " ★推荐"
			}
			modelLabels[i] = fmt.Sprintf("%s  ( %d维 · %s )%s", m.Name, m.Dimension, m.Description, defaultMark)
		}
		modelIdx := 0
		modelPrompt := &survey.Select{
			Message: "选择模型:",
			Options: modelLabels,
		}
		if err := survey.AskOne(modelPrompt, &modelIdx); err != nil {
			return err
		}
		cfg.Model = provider.Models[modelIdx].ID
		cfg.Dimension = provider.Models[modelIdx].Dimension
	}

	// 测试连接
	testConn := true
	confirmPrompt := &survey.Confirm{
		Message: "要测试连接吗？",
		Default: true,
	}
	survey.AskOne(confirmPrompt, &testConn)

	if testConn {
		fmt.Print("   ⏳ 正在测试连接...")
		prov, err := NewProviderFromConfig(cfg.toProviderConfig())
		if err == nil && prov != nil {
			if hErr := prov.Health(); hErr == nil {
				fmt.Println(" ✅ 连接成功")
			} else {
				fmt.Printf(" ❌ 连接失败: %v\n", hErr)
				retry := false
				retryPrompt := &survey.Confirm{
					Message: "连接测试失败，是否继续保存配置？（可在 cortex setup 中重新配置）",
					Default: true,
				}
				survey.AskOne(retryPrompt, &retry)
				if !retry {
					return fmt.Errorf("setup cancelled by user")
				}
			}
		} else if err != nil {
			fmt.Printf(" ❌ 创建 Provider 失败: %v\n", err)
		} else {
			fmt.Println(" ✅ （无需测试，纯文本模式）")
		}
	}

	return nil
}

// WriteConfig 将配置写入 ~/.cortex/config.yaml
func (cfg *SetupConfig) WriteConfig() error {
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".cortex")
	configPath := filepath.Join(configDir, "config.yaml")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	// 读取现有配置保留非 embedding 字段
	existing := make(map[string]interface{})
	if data, err := os.ReadFile(configPath); err == nil {
		yaml.Unmarshal(data, &existing)
	}

	// 构建新的 embedding 配置
	embedMap := map[string]interface{}{
		"provider":    cfg.Provider,
		"auto_update": cfg.AutoUpdate,
	}

	switch cfg.Provider {
	case "ollama":
		embedMap["ollama"] = map[string]interface{}{
			"base_url": cfg.OllamaURL,
			"model":    cfg.OllamaModel,
		}
	case "openai":
		embedMap["openai"] = map[string]interface{}{
			"api_key":   cfg.APIKey,
			"model":     cfg.Model,
			"base_url":  cfg.BaseURL,
			"dimension": cfg.Dimension,
		}
	case "cohere":
		embedMap["cohere"] = map[string]interface{}{
			"api_key":   cfg.APIKey,
			"model":     cfg.Model,
			"base_url":  cfg.BaseURL,
			"dimension": cfg.Dimension,
		}
	case "voyage":
		embedMap["voyage"] = map[string]interface{}{
			"api_key":   cfg.APIKey,
			"model":     cfg.Model,
			"base_url":  cfg.BaseURL,
			"dimension": cfg.Dimension,
		}
	case "dashscope":
		embedMap["dashscope"] = map[string]interface{}{
			"api_key":   cfg.APIKey,
			"model":     cfg.Model,
			"base_url":  cfg.BaseURL,
			"dimension": cfg.Dimension,
		}
	case "zhipu":
		embedMap["zhipu"] = map[string]interface{}{
			"api_key":   cfg.APIKey,
			"model":     cfg.Model,
			"base_url":  cfg.BaseURL,
			"dimension": cfg.Dimension,
		}
	case "baidu":
		embedMap["baidu"] = map[string]interface{}{
			"api_key":    cfg.APIKey,
			"secret_key": cfg.SecretKey,
			"model":      cfg.Model,
			"base_url":   cfg.BaseURL,
			"dimension":  cfg.Dimension,
		}
	}

	existing["embedding"] = embedMap

	// 确保 cortex 配置存在，默认开启认证
	if _, ok := existing["cortex"]; !ok {
		existing["cortex"] = map[string]interface{}{
			"db_path":      filepath.Join(filepath.Dir(configPath), "cortex.db"),
			"log_level":    "info",
			"auth_enabled": true,
		}
	}

	data, err := yaml.Marshal(existing)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("\n✅ 配置已保存到 %s\n", configPath)
	return nil
}

// toProviderConfig 转换为 ProviderConfig（用于测试连接）
func (cfg *SetupConfig) toProviderConfig() ProviderConfig {
	pc := ProviderConfig{Provider: cfg.Provider}
	switch cfg.Provider {
	case "ollama":
		pc.Ollama.BaseURL = cfg.OllamaURL
		pc.Ollama.Model = cfg.OllamaModel
	case "openai":
		pc.OpenAI.APIKey = cfg.APIKey
		pc.OpenAI.Model = cfg.Model
		pc.OpenAI.BaseURL = cfg.BaseURL
		pc.OpenAI.Dimension = cfg.Dimension
	case "cohere":
		pc.Cohere.APIKey = cfg.APIKey
		pc.Cohere.Model = cfg.Model
		pc.Cohere.BaseURL = cfg.BaseURL
		pc.Cohere.Dimension = cfg.Dimension
	case "voyage":
		pc.Voyage.APIKey = cfg.APIKey
		pc.Voyage.Model = cfg.Model
		pc.Voyage.BaseURL = cfg.BaseURL
		pc.Voyage.Dimension = cfg.Dimension
	case "dashscope":
		pc.DashScope.APIKey = cfg.APIKey
		pc.DashScope.Model = cfg.Model
		pc.DashScope.BaseURL = cfg.BaseURL
		pc.DashScope.Dimension = cfg.Dimension
	case "zhipu":
		pc.Zhipu.APIKey = cfg.APIKey
		pc.Zhipu.Model = cfg.Model
		pc.Zhipu.BaseURL = cfg.BaseURL
		pc.Zhipu.Dimension = cfg.Dimension
	case "baidu":
		pc.Baidu.APIKey = cfg.APIKey
		pc.Baidu.SecretKey = cfg.SecretKey
		pc.Baidu.Model = cfg.Model
		pc.Baidu.BaseURL = cfg.BaseURL
		pc.Baidu.Dimension = cfg.Dimension
	}
	return pc
}

func getDefaultBaseURL(providerID string) string {
	urls := map[string]string{
		"openai":    "https://api.openai.com/v1",
		"cohere":    "https://api.cohere.com",
		"voyage":    "https://api.voyageai.com/v1",
		"dashscope": "https://dashscope.aliyuncs.com/api/v1",
		"zhipu":     "https://open.bigmodel.cn/api/paas/v4",
		"baidu":     "https://aip.baidubce.com",
	}
	if u, ok := urls[providerID]; ok {
		return u
	}
	return ""
}

// SuggestInstallOllama 提示安装 Ollama（当选择 ollama 但检测不到时）
func SuggestInstallOllama() {
	fmt.Println()
	fmt.Println("⚠️  检测到 Ollama 服务未运行或未安装")
	fmt.Println()
	fmt.Println("安装 Ollama:")
	fmt.Println("  macOS/Linux: curl -fsSL https://ollama.ai/install.sh | sh")
	fmt.Println("  Windows:     从 https://ollama.ai/download 下载安装")
	fmt.Println()
	fmt.Println("安装后拉取模型:")
	fmt.Println("  ollama pull all-minilm")
	fmt.Println()
	fmt.Println("启动服务:")
	fmt.Println("  ollama serve")
	fmt.Println()

	openBrowser := false
	prompt := &survey.Confirm{
		Message: "是否打开 Ollama 官网下载页面？",
		Default: true,
	}
	survey.AskOne(prompt, &openBrowser)
	if openBrowser {
		openURL("https://ollama.ai/download")
	}
}

func openURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Start()
}
