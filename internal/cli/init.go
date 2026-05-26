package cli

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
	"gopkg.in/yaml.v3"
)

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

func initStorage(cfg *config.Config, logger *zap.Logger, buildIndex bool) (storage.Storage, error) {
	dbDir := filepath.Dir(cfg.Cortex.DBPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	st, err := storage.NewSQLiteStorage(cfg.Cortex.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to init storage: %w", err)
	}

	st.SetLogger(logger)

	if buildIndex {
		logger.Info("building vector index", zap.Int("vectors", func() int {
			count, _ := st.GetVectorsCount("")
			return count
		}()))

		if err := st.BuildHNSWIndex(); err != nil {
			logger.Warn("vector index build failed, search may fallback to DB-only", zap.Error(err))
		} else {
			logger.Info("vector index built successfully")
		}
	} else {
		logger.Info("skipping vector index build (no embedding provider configured)")
	}

	logger.Info("storage initialized", zap.String("path", cfg.Cortex.DBPath))
	return st, nil
}

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

func initPrefetchEngine(se *search.HybridSearchEngine) *search.PrefetchEngine {
	pe := search.NewPrefetchEngine(se)
	se.SetPrefetchEngine(pe)
	return pe
}

func runIndex(cmd *cobra.Command, args []string) {
	cfg, logger, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	st, err := initStorageLight(cfg, logger)
	if err != nil {
		logger.Fatal("failed to init storage", zap.Error(err))
	}
	defer st.Close()

	emb, err := initEmbedding(cfg, logger)
	if err != nil {
		logger.Fatal("failed to init embedding", zap.Error(err))
	}

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

	if forceReindex {
		progress, _ := st.GetIndexProgress(path)
		if progress != nil {
			st.DeleteIndexProgress(progress.ID)
		}
		logger.Info("force re-index: cleared existing checkpoint", zap.String("path", path))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if indexTimeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), indexTimeout)
		defer cancel()
		logger.Info("index timeout set", zap.Duration("timeout", indexTimeout))
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case sig := <-sigCh:
			fmt.Printf("\n  ⚠️  Received %v, saving checkpoint and exiting gracefully...\n", sig)
			cancel()
			time.Sleep(200 * time.Millisecond)
		case <-ctx.Done():
		}
	}()

	lastLineLen := 0
	idx.OnProgress = func(evt models.IndexProgressEvent) {
		completed := evt.Indexed + evt.Skipped + evt.Failed
		pct := float64(completed) / float64(evt.Total) * 100

		barWidth := 20
		filled := int(pct / 100 * float64(barWidth))
		if filled > barWidth {
			filled = barWidth
		}
		filledStr := strings.Repeat("█", filled)
		emptyStr := strings.Repeat("░", barWidth-filled)

		filename := evt.CurrentFile
		if len(filename) > 45 {
			filename = "..." + filename[len(filename)-42:]
		}

		etaStr := ""
		if evt.ETA > 0 && completed < evt.Total {
			etaStr = " · ETA " + formatDuration(evt.ETA)
		}
		elapsedStr := formatDuration(evt.Elapsed)

		line := fmt.Sprintf("\r  Indexing [%s%s] %3.0f%%  %d/%d · %s · %.1f/s%s · %s",
			filledStr, emptyStr, pct, completed, evt.Total, elapsedStr, evt.Speed, etaStr, filename)

		if len(line) < lastLineLen {
			fmt.Print(line + strings.Repeat(" ", lastLineLen-len(line)))
		} else {
			fmt.Print(line)
		}
		lastLineLen = len(line)
	}

	result, err := idx.IndexDirectoryWithCheckpoint(ctx, path, "")
	if err != nil {
		fmt.Println()
		if errors.Is(err, context.Canceled) {
			logger.Warn("indexing cancelled by user or timeout")
			fmt.Printf("  ⚠️  Indexing interrupted (checkpoint saved, use 'cortex index' to resume)\n")
		} else {
			logger.Error("indexing failed", zap.Error(err))
		}
		os.Exit(1)
	}

	fmt.Printf("\r%s\r", strings.Repeat(" ", lastLineLen))
	fmt.Printf("  ✅ Indexing complete! %d indexed, %d skipped, %d failed · %s\n",
		result.Indexed, result.Skipped, result.Failed,
		time.Duration(result.Duration)*time.Millisecond)
	if result.Failed > 0 {
		fmt.Printf("     (see logs for details on failed files)\n")
	}
}

func runSearch(cmd *cobra.Command, args []string) {
	cfg, logger, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

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
		UserID: "",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := se.Search(ctx, query, opts)
	if err != nil {
		logger.Error("search failed", zap.Error(err))
		os.Exit(1)
	}

	fmt.Printf("\n🔍 Search results for: %s\n\n", query)

	if cfg.Embedding.Provider == "none" {
		fmt.Println("  💡  FTS-only mode. Install Ollama + nomic-embed-text for semantic search.")
		fmt.Println()
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
			content := r.Chunk.ContentRaw
			if len(content) > 300 {
				content = content[:300] + "..."
			}
			jr = append(jr, jsonResult{
				Rank:    i + 1,
				Score:   r.Score,
				Path:    r.Chunk.DocumentID,
				Section: r.Chunk.HeadingPath,
				Content: content,
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(jr)
		return
	}

	if len(results) == 0 {
		fmt.Println("   No results found.")
		return
	}

	for i, r := range results {
		fmt.Printf("%d. [Score: %.4f] %s\n", i+1, r.Score, r.Chunk.HeadingPath)
		fmt.Printf("   %s\n", r.Chunk.DocumentID)
		content := r.Chunk.ContentRaw
		if len(content) > 300 {
			content = content[:300] + "..."
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
		UserID: "",
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

	mcpServer := api.NewMCPServer(se, st, emb, logger)

	logger.Info("starting MCP server",
		zap.String("protocol", api.MCPProtocolVersion),
	)

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

	if err := config.WatchConfig(func(newCfg *config.Config) {
		logger.Info("config file changed, new settings loaded",
			zap.String("embedding.provider", newCfg.Embedding.Provider),
			zap.String("log_level", newCfg.Cortex.LogLevel),
		)
	}); err != nil {
		logger.Warn("failed to start config watcher, config hot-reload disabled", zap.Error(err))
	}

	emb, err := initEmbedding(cfg, logger)
	if err != nil {
		logger.Fatal("failed to init embedding", zap.Error(err))
	}

	st, err := initStorage(cfg, logger, emb != nil)
	if err != nil {
		logger.Fatal("failed to init storage", zap.Error(err))
	}

	se, err := initSearchEngine(st, emb, logger)
	if err != nil {
		logger.Fatal("failed to init search engine", zap.Error(err))
	}

	authService := auth.NewAuthServiceWithStorage(st)

	var restServer *api.RESTServer
	if cfg.Cortex.AuthEnabled {
		restServer = api.NewRESTServerWithAuth(se, st, emb, logger, authService)
		logger.Info("auth enabled", zap.Bool("auth", cfg.Cortex.AuthEnabled))
	} else {
		restServer = api.NewRESTServer(se, st, emb, logger)
	}

	addr := fmt.Sprintf(":%d", servePort)
	logger.Info("starting REST API server", zap.String("addr", addr))

	metricsServer := metrics.StartMetricsServer(":9090")
	logger.Info("metrics server started", zap.String("addr", ":9090"))

	go func() {
		if err := restServer.ListenAndServe(addr); err != nil && err != http.ErrServerClosed {
			logger.Fatal("REST server failed", zap.Error(err))
		}
	}()
	logger.Info("REST API server started", zap.String("addr", addr))

	if cfg.Backup.AutoBackup {
		backupMgr := storage.NewBackupManager(cfg.Cortex.DBPath)
		if cfg.Backup.MaxBackups > 0 {
			backupMgr.SetMaxBackups(cfg.Backup.MaxBackups)
		}
		backupMgr.StartAutoBackup(24 * time.Hour)
		defer backupMgr.StopAutoBackup()
		logger.Info("auto backup enabled", zap.Int("max_backups", cfg.Backup.MaxBackups))
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	logger.Info("received shutdown signal", zap.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Info("shutting down REST server...")
	if err := restServer.Shutdown(ctx); err != nil {
		logger.Warn("REST server shutdown error", zap.Error(err))
	}

	logger.Info("shutting down metrics server...")
	if err := metrics.ShutdownMetricsServer(metricsServer, 5*time.Second); err != nil {
		logger.Warn("metrics server shutdown error", zap.Error(err))
	}

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

	st, err := initStorage(cfg, logger, true)
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

	st, err := initStorage(cfg, logger, true)
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

	se, err := initSearchEngine(st, emb, logger)
	if err != nil {
		logger.Warn("failed to init search engine for prefetch", zap.Error(err))
	} else {
		pe := initPrefetchEngine(se)
		watcher.SetPrefetch(pe)
		logger.Info("prefetch engine enabled for watcher")
	}

	if err := watcher.Start(); err != nil {
		logger.Fatal("failed to start watcher", zap.Error(err))
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	logger.Info("received shutdown signal, stopping watcher", zap.String("signal", sig.String()))
	watcher.Stop()
}

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

	if cfg.Provider == "ollama" {
		testCfg, _ := config.Load(cfgPath)
		if testCfg != nil {
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

func runInstall(cmd *cobra.Command, args []string) {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".cortex", "config.yaml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Println("  📝 No config found, creating default (FTS5-only, zero dependencies)...")
		cfgDir := filepath.Dir(configPath)
		if err := os.MkdirAll(cfgDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating config dir: %v\n", err)
			os.Exit(1)
		}
		defaultCfg := map[string]interface{}{
			"cortex": map[string]interface{}{
				"db_path":   filepath.Join(cfgDir, "cortex.db"),
				"log_level": "info",
			},
			"embedding": map[string]interface{}{
				"provider":    "none",
				"auto_update": false,
			},
			"index": map[string]interface{}{
				"max_tokens":     512,
				"overlap_tokens": 64,
				"min_chars":      50,
				"workers":        8,
			},
			"search": map[string]interface{}{
				"cache_ttl":     "5m",
				"default_top_k": 10,
			},
		}
		data, err := yaml.Marshal(defaultCfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating config: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  ✅ Config created: %s\n", configPath)
	} else {
		fmt.Println("  ✅ Config found, using existing configuration")
	}

	cfg, logger, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	st, err := initStorageLight(cfg, logger)
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

	docDir := "."
	if len(args) > 0 {
		docDir = args[0]
	}

	absDir, err := filepath.Abs(docDir)
	if err != nil {
		logger.Fatal("failed to resolve path", zap.Error(err))
	}

	fmt.Printf("  📂 Indexing documents from: %s\n\n", absDir)
	idx.OnProgress = func(evt models.IndexProgressEvent) {
		completed := evt.Indexed + evt.Skipped + evt.Failed
		if evt.Total <= 0 {
			return
		}
		pct := float64(completed) / float64(evt.Total) * 100
		barWidth := 20
		filled := int(pct / 100 * float64(barWidth))
		if filled > barWidth {
			filled = barWidth
		}
		filledStr := strings.Repeat("█", filled)
		emptyStr := strings.Repeat("░", barWidth-filled)
		filename := evt.CurrentFile
		if len(filename) > 45 {
			filename = "..." + filename[len(filename)-42:]
		}
		elapsedStr := formatDuration(evt.Elapsed)
		line := fmt.Sprintf("\r  Indexing [%s%s] %3.0f%%  %d/%d · %s · %.1f/s · %s",
			filledStr, emptyStr, pct, completed, evt.Total, elapsedStr, evt.Speed, filename)
		fmt.Print(line)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case sig := <-sigCh:
			fmt.Printf("\n  ⚠️  Received %v, saving checkpoint...\n", sig)
			cancel()
		case <-ctx.Done():
		}
	}()

	result, err := idx.IndexDirectoryWithCheckpoint(ctx, absDir, "")
	if err != nil {
		logger.Error("indexing failed", zap.Error(err))
		os.Exit(1)
	}

	fmt.Print("\r" + strings.Repeat(" ", 80) + "\r")
	fmt.Printf("  ✅ Indexing complete! %d indexed, %d skipped, %d failed · %dms\n",
		result.Indexed, result.Skipped, result.Failed, result.Duration)

	fmt.Println()
	fmt.Println("  ─────────────────────────────────────────────")
	fmt.Println("  🎉  Cortex is ready!")
	fmt.Println()
	fmt.Println("  📖  Next steps:")
	fmt.Println()
	fmt.Printf("      Search:  cortex search \"your query\"\n")
	fmt.Println("      Status:  cortex status")
	fmt.Println()
	fmt.Println("  🔌  Trae MCP config (.trae/mcp.json):")
	fmt.Println(`      { "mcpServers": { "cortex": { "command": "cortex", "args": ["mcp"] } } }`)
	fmt.Println()
	fmt.Println("  ─────────────────────────────────────────────")
}

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
