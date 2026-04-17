package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/lh123aa/cortex/internal/api"
	"github.com/lh123aa/cortex/internal/config"
	"github.com/lh123aa/cortex/internal/embedding"
	"github.com/lh123aa/cortex/internal/index"
	"github.com/lh123aa/cortex/internal/log"
	"github.com/lh123aa/cortex/internal/models"
	"github.com/lh123aa/cortex/internal/search"
	"github.com/lh123aa/cortex/internal/storage"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	cfgPath    string
	logLevel   string
	topK       int
	mode       string
	tokenBudget int
)

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

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", "", "config file path")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "", "log level (debug/info/warn/error)")

	searchCmd.Flags().IntVarP(&topK, "top-k", "k", 10, "number of results to return")
	searchCmd.Flags().StringVarP(&mode, "mode", "m", "hybrid", "search mode (vector/bm25/hybrid)")

	contextCmd.Flags().IntVarP(&tokenBudget, "tokens", "t", 4000, "token budget for context")

	rootCmd.AddCommand(indexCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(contextCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(statusCmd)
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

	logger.Info("storage initialized", zap.String("path", cfg.Cortex.DBPath))
	return st, nil
}

func initEmbedding(cfg *config.Config, logger *zap.Logger) (embedding.EmbeddingProvider, error) {
	var primary embedding.EmbeddingProvider

	if cfg.Embedding.Provider == "ollama" || cfg.Embedding.Provider == "" {
		ollama := embedding.NewOllamaEmbedding(
			cfg.Embedding.Ollama.BaseURL,
			cfg.Embedding.Ollama.Model,
			768,
		)
		primary = ollama
		logger.Info("embedding provider initialized",
			zap.String("provider", "ollama"),
			zap.String("model", cfg.Embedding.Ollama.Model),
		)
	}

	if cfg.Embedding.Provider == "onnx" {
		return nil, fmt.Errorf("ONNX provider not yet implemented")
	}

	if primary != nil {
		return embedding.NewProviderManager(primary, nil), nil
	}

	return nil, fmt.Errorf("no embedding provider configured")
}

func initIndexer(st storage.Storage, emb embedding.EmbeddingProvider, logger *zap.Logger) (*index.Indexer, error) {
	idx, err := index.NewIndexer(st, emb)
	if err != nil {
		return nil, fmt.Errorf("failed to init indexer: %w", err)
	}
	logger.Info("indexer initialized")
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

	st, err := initStorage(cfg, logger)
	if err != nil {
		logger.Fatal("failed to init storage", zap.Error(err))
	}
	defer st.Close()

	emb, err := initEmbedding(cfg, logger)
	if err != nil {
		logger.Fatal("failed to init embedding", zap.Error(err))
	}

	idx, err := initIndexer(st, emb, logger)
	if err != nil {
		logger.Fatal("failed to init indexer", zap.Error(err))
	}

	path := args[0]
	logger.Info("starting indexing", zap.String("path", path))

	result, err := idx.IndexDirectory(path)
	if err != nil {
		logger.Error("indexing failed", zap.Error(err))
		os.Exit(1)
	}

	fmt.Printf("\n✅ Indexing complete:\n")
	fmt.Printf("   Total:   %d files\n", result.Total)
	fmt.Printf("   Indexed: %d files\n", result.Indexed)
	fmt.Printf("   Skipped: %d files (unchanged)\n", result.Skipped)
	fmt.Printf("   Failed:  %d files\n", result.Failed)
	fmt.Printf("   Time:    %s\n", time.Duration(result.Duration)*time.Millisecond)
}

func runSearch(cmd *cobra.Command, args []string) {
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

	query := args[0]
	opts := models.SearchOptions{
		TopK: topK,
		Mode: mode,
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

	for i, r := range results {
		fmt.Printf("%d. [Score: %.4f] %s\n", i+1, r.Score, r.Chunk.HeadingPath)
		fmt.Printf("   %s\n", r.Chunk.DocumentID)
		if len(r.Chunk.ContentRaw) > 200 {
			fmt.Printf("   %s...\n\n", r.Chunk.ContentRaw[:200])
		} else {
			fmt.Printf("   %s\n\n", r.Chunk.ContentRaw)
		}
	}
}

func runContext(cmd *cobra.Command, args []string) {
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

	query := args[0]
	opts := models.SearchOptions{
		TopK: 20,
		Mode: "hybrid",
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

	mcpServer := api.NewMCPServer(se, st, logger)

	logger.Info("starting MCP server",
		zap.String("protocol", api.MCPProtocolVersion),
	)

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

	restServer := api.NewRESTServer(se, st, logger)

	logger.Info("starting REST API server", zap.String("addr", ":8080"))

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("shutting down REST server...")
		os.Exit(0)
	}()

	if err := restServer.ListenAndServe(":8080"); err != nil {
		logger.Error("REST server error", zap.Error(err))
		os.Exit(1)
	}
}

func runStatus(cmd *cobra.Command, args []string) {
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

	docCount, _ := st.GetDocumentsCount()
	chunkCount, _ := st.GetChunksCount()
	vectorCount, _ := st.GetVectorsCount()

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
