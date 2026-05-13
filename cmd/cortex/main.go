package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/lh123aa/cortex/internal/api"
	"github.com/lh123aa/cortex/internal/auth"
	"github.com/lh123aa/cortex/internal/config"
	"github.com/lh123aa/cortex/internal/embedding"
	"github.com/lh123aa/cortex/internal/index"
	"github.com/lh123aa/cortex/internal/log"
	"github.com/lh123aa/cortex/internal/metrics"
	"github.com/lh123aa/cortex/internal/models"
	"github.com/lh123aa/cortex/internal/search"
	"github.com/lh123aa/cortex/internal/storage"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	cfgPath         string
	logLevel        string
	topK            int
	mode            string
	tokenBudget     int
	jsonOutput      bool
	dedupMode       string
	dedupThreshold  float64
	forceReindex    bool
	indexTimeout    time.Duration
	indexWorkers    int
)

// highlightText 高亮文本中的匹配关键词（ANSI 黄色标记）
func highlightText(text, query string) string {
	if query == "" || text == "" {
		return text
	}
	terms := strings.Fields(query)
	result := text
	for _, term := range terms {
		if len(term) < 2 {
			continue
		}
		result = strings.ReplaceAll(result, term, "\033[33m"+term+"\033[0m")
		upper := strings.ToUpper(term)
		if upper != term {
			result = strings.ReplaceAll(result, upper, "\033[33m"+upper+"\033[0m")
		}
	}
	return result
}

var rootCmd = &cobra.Command{
	Use:   "cortex",
	Short: "Cortex - Agent Knowledge Base",
	Long: `Cortex is a local knowledge base system for AI Agents.
It supports hybrid search (vector + BM25), multiple file formats,
and MCP protocol for AI Agent integration.`,
}

var indexCmd = &cobra.Command{
	Use:   "index <path>",
	Short: "Index documents from a directory",
	Args:  cobra.ExactArgs(1),
	Run:   runIndex,
}

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search indexed documents",
	Args:  cobra.ExactArgs(1),
	Run:   runSearch,
}

var contextCmd = &cobra.Command{
	Use:   "context <query>",
	Short: "Generate RAG context for a query",
	Args:  cobra.ExactArgs(1),
	Run:   runContext,
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server for AI Agent integration",
	Run:   runMCP,
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start REST API server",
	Run:   runServe,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show indexing status",
	Run:   runStatus,
}

var dedupCmd = &cobra.Command{
	Use:   "dedup",
	Short: "Deduplicate chunks in the knowledge base",
	Long: `Scan all chunks and remove duplicates.
Without flags: dedup by content hash (exact match).
With --vector: dedup by vector similarity (semantic match).`,
	Run: runDedup,
}

var watchCmd = &cobra.Command{
	Use:   "watch <path>",
	Short: "Watch a directory for changes and auto-index",
	Long: `Monitor a directory for file changes (create, modify, delete)
and automatically update the index in real-time.

Uses filesystem notifications (fsnotify) for instant updates.
Supports the same file types as 'cortex index'.

Examples:
  cortex watch ~/my-docs
  cortex watch /path/to/project`,
	Args: cobra.ExactArgs(1),
	Run:  runWatch,
}

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Show storage usage and plan info",
	Run:   runUsage,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Cortex v%s (commit: %s, built: %s)\n", api.Version, api.Commit, api.Date)
	},
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive embedding provider configuration wizard",
	Long: `Run the interactive setup wizard to configure your embedding provider.

Supports local (Ollama), international API (OpenAI, Cohere, Voyage),
and domestic API (DashScope, Zhipu, Baidu) providers.
Run 'cortex setup' anytime to reconfigure.`,
	Run: runSetup,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", "", "config file path")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "", "log level (debug/info/warn/error)")

	indexCmd.Flags().BoolVarP(&forceReindex, "force", "f", false, "Force re-index from scratch (ignore checkpoint)")
	indexCmd.Flags().DurationVarP(&indexTimeout, "timeout", "t", 0, "Maximum time for indexing (e.g. 30m, 1h). 0 = no limit")
	indexCmd.Flags().IntVarP(&indexWorkers, "workers", "w", 0, "Number of indexing workers (default 16)")

	searchCmd.Flags().IntVarP(&topK, "top-k", "k", 10, "number of results to return")
	searchCmd.Flags().StringVarP(&mode, "mode", "m", "hybrid", "search mode (vector/bm25/hybrid)")
	searchCmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "output as JSON")
	statusCmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "output as JSON")

	contextCmd.Flags().IntVarP(&tokenBudget, "tokens", "t", 4000, "token budget for context")

	dedupCmd.Flags().StringVarP(&dedupMode, "mode", "m", "hash", "dedup mode: hash (exact) | vector (semantic)")
	dedupCmd.Flags().Float64VarP(&dedupThreshold, "threshold", "t", 0.95, "similarity threshold for vector dedup (0.0-1.0)")

	rootCmd.AddCommand(indexCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(contextCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(dedupCmd)
	rootCmd.AddCommand(watchCmd)
	rootCmd.AddCommand(usageCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(setupCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func loadConfig() (*config.Config, *zap.Logger, error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	levelStr := cfg.Cortex.LogLevel
	if logLevel != "" {
		levelStr = logLevel
	}

	var level zapcore.Level
	if err := level.UnmarshalText([]byte(levelStr)); err != nil {
		level = zapcore.InfoLevel
	}

	logger := log.NewLogger(level)

	return cfg, logger, nil
}

func initStorage(cfg *config.Config, logger *zap.Logger) (storage.Storage, error) {
	dbDir := filepath.Dir(cfg.Cortex.DBPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	st, err := storage.NewSQLiteStorage(cfg.Cortex.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to init storage: %w", err)
	}

	// 设置结构化日志
	st.SetLogger(logger)

	// 构建 HNSW 索引（如果数据库中已有向量）
	if err := st.BuildHNSWIndex(); err != nil {
		logger.Warn("failed to build HNSW index, using brute force search", zap.Error(err))
	}

	logger.Info("storage initialized", zap.String("path", cfg.Cortex.DBPath))
	return st, nil
}

// initStorageLight 轻量版初始化，跳过 HNSW 索引构建
// 用于 index/status/usage 等不需要向量搜索的命令
// HNSW 只在需要搜索时加载，索引类命令无需加载存量向量
func initStorageLight(cfg *config.Config, logger *zap.Logger) (storage.Storage, error) {
	dbDir := filepath.Dir(cfg.Cortex.DBPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	st, err := storage.NewSQLiteStorage(cfg.Cortex.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to init storage: %w", err)
	}

	st.SetLogger(logger)
	logger.Info("lightweight storage initialized (HNSW skipped)", zap.String("path", cfg.Cortex.DBPath))
	return st, nil
}

func initEmbedding(cfg *config.Config, logger *zap.Logger) (embedding.EmbeddingProvider, error) {
	// 使用工厂模式从配置创建 Provider
	provider, err := embedding.NewProviderFromConfig(convertConfig(cfg))
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding provider: %w", err)
	}
	if provider == nil {
		logger.Info("embedding provider disabled, search will use FTS-only mode")
		return nil, nil
	}
	logger.Info("embedding provider initialized",
		zap.String("provider", cfg.Embedding.Provider),
		zap.String("model", provider.Name()),
	)
	return provider, nil
}

// convertConfig 将 config.Config 转换为 embedding.ProviderConfig
func convertConfig(cfg *config.Config) embedding.ProviderConfig {
	pc := embedding.ProviderConfig{Provider: cfg.Embedding.Provider}
	pc.Ollama.BaseURL = cfg.Embedding.Ollama.BaseURL
	pc.Ollama.Model = cfg.Embedding.Ollama.Model
	pc.OpenAI.APIKey = cfg.Embedding.OpenAI.APIKey
	pc.OpenAI.Model = cfg.Embedding.OpenAI.Model
	pc.OpenAI.BaseURL = cfg.Embedding.OpenAI.BaseURL
	pc.OpenAI.Dimension = cfg.Embedding.OpenAI.Dimension
	pc.Cohere.APIKey = cfg.Embedding.Cohere.APIKey
	pc.Cohere.Model = cfg.Embedding.Cohere.Model
	pc.Cohere.BaseURL = cfg.Embedding.Cohere.BaseURL
	pc.Cohere.Dimension = cfg.Embedding.Cohere.Dimension
	pc.Voyage.APIKey = cfg.Embedding.Voyage.APIKey
	pc.Voyage.Model = cfg.Embedding.Voyage.Model
	pc.Voyage.BaseURL = cfg.Embedding.Voyage.BaseURL
	pc.Voyage.Dimension = cfg.Embedding.Voyage.Dimension
	pc.DashScope.APIKey = cfg.Embedding.DashScope.APIKey
	pc.DashScope.Model = cfg.Embedding.DashScope.Model
	pc.DashScope.BaseURL = cfg.Embedding.DashScope.BaseURL
	pc.DashScope.Dimension = cfg.Embedding.DashScope.Dimension
	pc.Zhipu.APIKey = cfg.Embedding.Zhipu.APIKey
	pc.Zhipu.Model = cfg.Embedding.Zhipu.Model
	pc.Zhipu.BaseURL = cfg.Embedding.Zhipu.BaseURL
	pc.Zhipu.Dimension = cfg.Embedding.Zhipu.Dimension
	pc.Baidu.APIKey = cfg.Embedding.Baidu.APIKey
	pc.Baidu.SecretKey = cfg.Embedding.Baidu.SecretKey
	pc.Baidu.Model = cfg.Embedding.Baidu.Model
	pc.Baidu.BaseURL = cfg.Embedding.Baidu.BaseURL
	pc.Baidu.Dimension = cfg.Embedding.Baidu.Dimension
	return pc
}

// runSetup 运行交互式配置向导
func runSetup(cmd *cobra.Command, args []string) {
	cfg, err := embedding.RunSetupWizard()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.WriteConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing config: %v\n", err)
		os.Exit(1)
	}

	// 如果选的是 ollama 但未检测到服务，提示安装
	if cfg.Provider == "ollama" {
		testCfg, _ := config.Load(cfgPath)
		if testCfg != nil {
			// 简单测试 Ollama 是否可达
			client := &http.Client{Timeout: 3 * time.Second}
			resp, err := client.Get("http://localhost:11434/api/tags")
			if err != nil {
				embedding.SuggestInstallOllama()
			} else {
				resp.Body.Close()
			}
		}
	}
}

func initIndexer(st storage.Storage, emb embedding.EmbeddingProvider, cfg *config.Config, logger *zap.Logger) (*index.Indexer, error) {
	idx, err := index.NewIndexer(st, emb, cfg.Index.Workers)
	if err != nil {
		return nil, fmt.Errorf("failed to init indexer: %w", err)
	}
	logger.Info("indexer initialized", zap.Int("workers", cfg.Index.Workers))
	return idx, nil
}

func initSearchEngine(st storage.Storage, emb embedding.EmbeddingProvider, logger *zap.Logger) (*search.HybridSearchEngine, error) {
	se := search.NewHybridSearchEngine(st, emb)
	logger.Info("search engine initialized")
	return se, nil
}

func runIndex(cmd *cobra.Command, args []string) {
	cfg, logger, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// index 命令无需 HNSW 索引，用轻量版跳过向量加载
	st, err := initStorageLight(cfg, logger)
	if err != nil {
		logger.Fatal("failed to init storage", zap.Error(err))
	}
	defer st.Close()

	emb, err := initEmbedding(cfg, logger)
	if err != nil {
		logger.Fatal("failed to init embedding", zap.Error(err))
	}

	// --workers 标志覆盖配置中的 worker 数
	if indexWorkers > 0 {
		cfg.Index.Workers = indexWorkers
		logger.Info("index workers overridden by CLI flag", zap.Int("workers", indexWorkers))
	}

	idx, err := initIndexer(st, emb, cfg, logger)
	if err != nil {
		logger.Fatal("failed to init indexer", zap.Error(err))
	}
	idx.Force = forceReindex

	path := args[0]
	logger.Info("starting indexing", zap.String("path", path))

	// --force 标志：清除已有 checkpoint，重新索引
	if forceReindex {
		progress, _ := st.GetIndexProgress(path)
		if progress != nil {
			st.DeleteIndexProgress(progress.ID)
		}
		logger.Info("force re-index: cleared existing checkpoint, all content will be re-processed", zap.String("path", path))
	}

	// 创建可取消 context（支持超时 + Ctrl+C）
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if indexTimeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), indexTimeout)
		defer cancel()
		logger.Info("index timeout set", zap.Duration("timeout", indexTimeout))
	}

	// 信号处理：SIGINT/SIGTERM → 优雅退出
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case sig := <-sigCh:
			fmt.Printf("\n  ⚠️  Received %v, saving checkpoint and exiting gracefully...\n", sig)
			cancel()
			// 给 goroutine 一点时间完成 ctx 传播
			time.Sleep(200 * time.Millisecond)
		case <-ctx.Done():
			// 超时触达，不输出额外信息
		}
	}()

	// 设置实时进度条
	lastLineLen := 0
	idx.OnProgress = func(evt models.IndexProgressEvent) {
		completed := evt.Indexed + evt.Skipped + evt.Failed
		pct := float64(completed) / float64(evt.Total) * 100

		// 20 字符进度条
		barWidth := 20
		filled := int(pct / 100 * float64(barWidth))
		if filled > barWidth {
			filled = barWidth
		}
		filledStr := strings.Repeat("█", filled)
		emptyStr := strings.Repeat("░", barWidth-filled)

		// 截断文件名
		filename := evt.CurrentFile
		if len(filename) > 45 {
			filename = "..." + filename[len(filename)-42:]
		}

		// 格式化时间
		etaStr := ""
		if evt.ETA > 0 && completed < evt.Total {
			etaStr = " · ETA " + formatDuration(evt.ETA)
		}
		elapsedStr := formatDuration(evt.Elapsed)

		// 构建一行进度文本
		line := fmt.Sprintf("\r  Indexing [%s%s] %3.0f%%  %d/%d · %s · %.1f/s%s · %s",
			filledStr, emptyStr, pct, completed, evt.Total, elapsedStr, evt.Speed, etaStr, filename)

		// 用空格填充覆盖旧内容（避免残影）
		if len(line) < lastLineLen {
			fmt.Print(line + strings.Repeat(" ", lastLineLen-len(line)))
		} else {
			fmt.Print(line)
		}
		lastLineLen = len(line)
	}

	// 使用支持断点恢复的 checkpoint 版本（带 Context 超时/取消）
	result, err := idx.IndexDirectoryWithCheckpoint(ctx, path, "")
	if err != nil {
		fmt.Println() // 换行结束进度行
		if errors.Is(err, context.Canceled) {
			logger.Warn("indexing cancelled by user or timeout")
			fmt.Printf("  ⚠️  Indexing interrupted (checkpoint saved, use 'cortex index' to resume)\n")
		} else {
			logger.Error("indexing failed", zap.Error(err))
		}
		os.Exit(1)
	}

	// 覆盖进度行，输出最终结果
	fmt.Printf("\r%s\r", strings.Repeat(" ", lastLineLen))
	fmt.Printf("  ✅ Indexing complete! %d indexed, %d skipped, %d failed · %s\n",
		result.Indexed, result.Skipped, result.Failed,
		time.Duration(result.Duration)*time.Millisecond)
	if result.Failed > 0 {
		fmt.Printf("     (see logs for details on failed files)\n")
	}
}

// formatDuration 格式化 time.Duration 为人类可读字符串
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dh %dm %ds", h, m, s)
}

func runSearch(cmd *cobra.Command, args []string) {
	cfg, logger, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// search 使用 light 版初始化（HNSW 降级到暴力搜索，FTS 不受影响）
	st, err := initStorageLight(cfg, logger)
	if err != nil {
		logger.Fatal("failed to init storage", zap.Error(err))
	}
	defer st.Close()

	emb, err := initEmbedding(cfg, logger)
	if err != nil {
		logger.Fatal("failed to init embedding", zap.Error(err))
	}

	se, err := initSearchEngine(st, emb, logger)
	if err != nil {
		logger.Fatal("failed to init search engine", zap.Error(err))
	}

	query := args[0]
	opts := models.SearchOptions{
		TopK:   topK,
		Mode:   mode,
		UserID: "", // CLI mode - no user isolation
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := se.Search(ctx, query, opts)
	if err != nil {
		logger.Error("search failed", zap.Error(err))
		os.Exit(1)
	}

	fmt.Printf("\n🔍 Search results for: %s\n\n", query)
	if len(results) == 0 {
		fmt.Println("   No results found.")
		return
	}

	if jsonOutput {
		type jsonResult struct {
			Rank    int     `json:"rank"`
			Score   float64 `json:"score"`
			Path    string  `json:"path"`
			Section string  `json:"section"`
			Content string  `json:"content"`
		}
		var jr []jsonResult
		for i, r := range results {
			jr = append(jr, jsonResult{
				Rank:    i + 1,
				Score:   r.Score,
				Path:    r.Chunk.DocumentID,
				Section: r.Chunk.HeadingPath,
				Content: r.Chunk.ContentRaw,
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(jr)
		return
	}

	for i, r := range results {
		fmt.Printf("%d. [Score: %.4f] %s\n", i+1, r.Score, r.Chunk.HeadingPath)
		fmt.Printf("   %s\n", r.Chunk.DocumentID)
		content := r.Chunk.ContentRaw
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		content = highlightText(content, query)
		fmt.Printf("   %s\n\n", content)
	}
}

func runContext(cmd *cobra.Command, args []string) {
	cfg, logger, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// context 使用 light 版初始化
	st, err := initStorageLight(cfg, logger)
	if err != nil {
		logger.Fatal("failed to init storage", zap.Error(err))
	}
	defer st.Close()

	emb, err := initEmbedding(cfg, logger)
	if err != nil {
		logger.Fatal("failed to init embedding", zap.Error(err))
	}

	se, err := initSearchEngine(st, emb, logger)
	if err != nil {
		logger.Fatal("failed to init search engine", zap.Error(err))
	}

	query := args[0]
	opts := models.SearchOptions{
		TopK:   20,
		Mode:   "hybrid",
		UserID: "", // CLI mode - no user isolation
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := se.Search(ctx, query, opts)
	if err != nil {
		logger.Error("search failed", zap.Error(err))
		os.Exit(1)
	}

	fmt.Printf("\n📝 RAG Context (budget: %d tokens):\n\n", tokenBudget)
	fmt.Println("---")
	for i, r := range results {
		fmt.Printf("[%d] %s\n", i+1, r.Chunk.HeadingPath)
		fmt.Printf("Source: %s\n", r.Chunk.DocumentID)
		fmt.Printf("%s\n\n", r.Chunk.ContentRaw)
	}
	fmt.Println("---")
}

func runMCP(cmd *cobra.Command, args []string) {
	cfg, logger, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	st, err := initStorage(cfg, logger)
	if err != nil {
		logger.Fatal("failed to init storage", zap.Error(err))
	}
	defer st.Close()

	emb, err := initEmbedding(cfg, logger)
	if err != nil {
		logger.Fatal("failed to init embedding", zap.Error(err))
	}

	se, err := initSearchEngine(st, emb, logger)
	if err != nil {
		logger.Fatal("failed to init search engine", zap.Error(err))
	}

	mcpServer := api.NewMCPServer(se, st, emb, logger)

	logger.Info("starting MCP server",
		zap.String("protocol", api.MCPProtocolVersion),
	)

	// Graceful Shutdown 通道
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info("received shutdown signal, stopping MCP server", zap.String("signal", sig.String()))
		os.Exit(0)
	}()

	if err := mcpServer.Run(); err != nil {
		logger.Error("MCP server error", zap.Error(err))
		os.Exit(1)
	}
}

func runServe(cmd *cobra.Command, args []string) {
	cfg, logger, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// 启动配置热加载（配置文件变更自动生效）
	if err := config.WatchConfig(func(newCfg *config.Config) {
		logger.Info("config file changed, new settings loaded",
			zap.String("embedding.provider", newCfg.Embedding.Provider),
			zap.String("log_level", newCfg.Cortex.LogLevel),
		)
	}); err != nil {
		logger.Warn("failed to start config watcher, config hot-reload disabled", zap.Error(err))
	}

	st, err := initStorage(cfg, logger)
	if err != nil {
		logger.Fatal("failed to init storage", zap.Error(err))
	}

	emb, err := initEmbedding(cfg, logger)
	if err != nil {
		logger.Fatal("failed to init embedding", zap.Error(err))
	}

	se, err := initSearchEngine(st, emb, logger)
	if err != nil {
		logger.Fatal("failed to init search engine", zap.Error(err))
	}

	// 创建认证服务（使用持久化存储）
	authService := auth.NewAuthServiceWithStorage(st)

	// 根据配置决定是否启用认证
	var restServer *api.RESTServer
	if cfg.Cortex.AuthEnabled {
		restServer = api.NewRESTServerWithAuth(se, st, emb, logger, authService)
		logger.Info("auth enabled", zap.Bool("auth", cfg.Cortex.AuthEnabled))
	} else {
		restServer = api.NewRESTServer(se, st, emb, logger)
	}

	logger.Info("starting REST API server", zap.String("addr", ":8080"))

	// 启动 Prometheus metrics 服务器
	metricsServer := metrics.StartMetricsServer(":9090")
	logger.Info("metrics server started", zap.String("addr", ":9090"))

	// 启动自动备份（默认 24 小时间隔，保留最多 10 份）
	if cfg.Backup.AutoBackup {
		backupMgr := storage.NewBackupManager(cfg.Cortex.DBPath)
		if cfg.Backup.MaxBackups > 0 {
			backupMgr.SetMaxBackups(cfg.Backup.MaxBackups)
		}
		backupMgr.StartAutoBackup(24 * time.Hour)
		defer backupMgr.StopAutoBackup()
		logger.Info("auto backup enabled", zap.Int("max_backups", cfg.Backup.MaxBackups))
	}

	// Graceful Shutdown 通道
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待信号
	sig := <-sigChan
	logger.Info("received shutdown signal", zap.String("signal", sig.String()))

	// 创建 shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 关闭 REST server（接受新请求但等待现有请求完成）
	logger.Info("shutting down REST server...")
	if err := restServer.Shutdown(ctx); err != nil {
		logger.Warn("REST server shutdown error", zap.Error(err))
	}

	// 关闭 metrics server
	logger.Info("shutting down metrics server...")
	if err := metrics.ShutdownMetricsServer(metricsServer, 5*time.Second); err != nil {
		logger.Warn("metrics server shutdown error", zap.Error(err))
	}

	// 关闭 storage（保存所有未决数据）
	logger.Info("closing storage...")
	st.Close()

	logger.Info("graceful shutdown complete")
}

func runStatus(cmd *cobra.Command, args []string) {
	cfg, logger, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// status 仅查询计数，无需 HNSW
	st, err := initStorageLight(cfg, logger)
	if err != nil {
		logger.Fatal("failed to init storage", zap.Error(err))
	}
	defer st.Close()

	docCount, _ := st.GetDocumentsCount("")
	chunkCount, _ := st.GetChunksCount("")
	vectorCount, _ := st.GetVectorsCount("")

	if jsonOutput {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"database":   cfg.Cortex.DBPath,
			"documents":  docCount,
			"chunks":     chunkCount,
			"vectors":    vectorCount,
			"embedding":  cfg.Embedding.Provider,
			"model":      cfg.Embedding.Ollama.Model,
			"ollama_url": cfg.Embedding.Ollama.BaseURL,
		})
		return
	}

	fmt.Println("\n📊 Cortex Status")
	fmt.Println("================")
	fmt.Printf("Database:     %s\n", cfg.Cortex.DBPath)
	fmt.Printf("Documents:    %d\n", docCount)
	fmt.Printf("Chunks:       %d\n", chunkCount)
	fmt.Printf("Vectors:      %d\n", vectorCount)
	fmt.Printf("Embedding:    %s\n", cfg.Embedding.Provider)
	if cfg.Embedding.Provider == "ollama" {
		fmt.Printf("  Model:      %s\n", cfg.Embedding.Ollama.Model)
		fmt.Printf("  URL:        %s\n", cfg.Embedding.Ollama.BaseURL)
	}
}

func runDedup(cmd *cobra.Command, args []string) {
	cfg, logger, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	st, err := initStorage(cfg, logger)
	if err != nil {
		logger.Fatal("failed to init storage", zap.Error(err))
	}
	defer st.Close()

	switch dedupMode {
	case "hash":
		removed, groups, err := st.DedupChunks()
		if err != nil {
			logger.Fatal("dedup failed", zap.Error(err))
		}
		if groups == 0 {
			fmt.Println("✅ No duplicate chunks found by content hash.")
		} else {
			fmt.Printf("✅ Content hash dedup complete: %d groups, %d chunks removed.\n", groups, removed)
		}

	case "vector":
		if dedupThreshold < 0 || dedupThreshold > 1 {
			logger.Fatal("threshold must be between 0.0 and 1.0")
		}
		removed, candidates, err := st.DedupByVector(dedupThreshold)
		if err != nil {
			logger.Fatal("vector dedup failed", zap.Error(err))
		}
		if removed == 0 {
			fmt.Printf("✅ No semantic duplicates found (threshold=%.2f, scanned %d chunks).\n", dedupThreshold, candidates)
		} else {
			fmt.Printf("✅ Vector dedup complete: removed %d / %d chunks (threshold=%.2f).\n", removed, candidates, dedupThreshold)
		}

	case "minhash":
		if dedupThreshold < 0 || dedupThreshold > 1 {
			logger.Fatal("threshold must be between 0.0 and 1.0")
		}
		removed, candidates, err := st.DedupByMinHash(dedupThreshold)
		if err != nil {
			logger.Fatal("minhash dedup failed", zap.Error(err))
		}
		if removed == 0 {
			fmt.Printf("✅ No minhash duplicates found (threshold=%.2f, scanned %d chunks).\n", dedupThreshold, candidates)
		} else {
			fmt.Printf("✅ MinHash dedup complete: removed %d / %d chunks (threshold=%.2f).\n", removed, candidates, dedupThreshold)
		}

	default:
		logger.Fatal("unknown dedup mode", zap.String("mode", dedupMode))
	}
}

func runUsage(cmd *cobra.Command, args []string) {
	cfg, logger, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// usage 只查询存储用量，无需 HNSW
	st, err := initStorageLight(cfg, logger)
	if err != nil {
		logger.Fatal("failed to init storage", zap.Error(err))
	}
	defer st.Close()

	used, err := st.CalculateStorageUsed("")
	if err != nil {
		logger.Fatal("failed to calculate storage", zap.Error(err))
	}

	limit := models.TierFree.StorageLimit()
	tier := string(models.TierFree)
	pct := float64(used) / float64(limit) * 100

	fmt.Println("\n📊 Cortex Usage")
	fmt.Println("================")
	fmt.Printf("Storage:   %s / %s (%.1f%%)\n", formatBytes(used), formatBytes(limit), pct)
	fmt.Printf("Tier:      %s\n", tier)
	if used > limit {
		fmt.Println("\n⚠️  Storage limit exceeded. Upgrade at https://cortex.ai/pricing")
	}
}

func runWatch(cmd *cobra.Command, args []string) {
	cfg, logger, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	st, err := initStorage(cfg, logger)
	if err != nil {
		logger.Fatal("failed to init storage", zap.Error(err))
	}
	defer st.Close()

	emb, err := initEmbedding(cfg, logger)
	if err != nil {
		logger.Fatal("failed to init embedding", zap.Error(err))
	}

	idx, err := initIndexer(st, emb, cfg, logger)
	if err != nil {
		logger.Fatal("failed to init indexer", zap.Error(err))
	}

	rootPath := args[0]
	logger.Info("starting file watcher", zap.String("path", rootPath))

	watcher, err := index.NewIncrementalWatcher(idx, rootPath, "")
	if err != nil {
		logger.Fatal("failed to create watcher", zap.Error(err))
	}

	if err := watcher.Start(); err != nil {
		logger.Fatal("failed to start watcher", zap.Error(err))
	}

	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	logger.Info("received shutdown signal, stopping watcher", zap.String("signal", sig.String()))
	watcher.Stop()
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	switch exp {
	case 0:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(div))
	case 1:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(div))
	case 2:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(div))
	default:
		return fmt.Sprintf("%.1f TB", float64(b)/float64(div))
	}
}
