package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lh123aa/cortex/internal/api"
	"github.com/lh123aa/cortex/internal/auth"
	"github.com/lh123aa/cortex/internal/config"
	"github.com/lh123aa/cortex/internal/metrics"
	"github.com/lh123aa/cortex/internal/storage"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	startDisableMCP    bool
	startDisableREST   bool
	startPort          int
	startNoRegister    bool
	startRegisterOnly  bool
)

func NewStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start all services (MCP + REST API) in one command",
		Long: `Start both MCP stdio server and REST API server simultaneously.
Automatically detects available ports and registers with AI tools.

Examples:
  cortex start                    # Start with default settings
  cortex start --port 3000        # Start REST API on port 3000
  cortex start --no-mcp           # Start REST API only
  cortex start --no-rest          # Start MCP only
  cortex start --register-only    # Only register with AI tools, no services`,
		Run: runStart,
	}
	cmd.Flags().BoolVar(&startDisableMCP, "no-mcp", false, "Disable MCP stdio mode")
	cmd.Flags().BoolVar(&startDisableREST, "no-rest", false, "Disable REST API mode")
	cmd.Flags().IntVarP(&startPort, "port", "p", 0, "REST API start port (0 = auto-detect)")
	cmd.Flags().BoolVar(&startNoRegister, "no-register", false, "Skip auto-registration with AI tools")
	cmd.Flags().BoolVar(&startRegisterOnly, "register-only", false, "Only register with AI tools, do not start services")
	return cmd
}

func runStart(cmd *cobra.Command, args []string) {
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

	docCount, _ := st.GetDocumentsCount("")

	if startRegisterOnly {
		registered := registerAITools(cfg.Starter.RegisterSkipTools)
		fmt.Println()
		fmt.Println("  Registration complete!")
		fmt.Println()
		if len(registered) > 0 {
			fmt.Println("  Registered AI tools:")
			for _, t := range registered {
				fmt.Printf("    ✅ %s\n", t)
			}
		} else {
			fmt.Println("  No AI tools detected or skipped.")
		}
		fmt.Println()
		st.Close()
		return
	}

	autoRegister := cfg.Starter.AutoRegister && !startNoRegister

	if !startDisableMCP || !startDisableREST {
		actualPort := startPort
		if !startDisableREST {
			startSearchPort := cfg.Starter.StartPort
			if startPort > 0 {
				startSearchPort = startPort
			}
			if startSearchPort <= 0 {
				startSearchPort = 2021
			}
			actualPort, err = findAvailablePort(startSearchPort)
			if err != nil {
				logger.Fatal("failed to find available port", zap.Error(err))
			}
		}

		if !startDisableMCP {
			go func() {
				mcpServer := api.NewMCPServer(se, st, emb, logger)
				logger.Info("starting MCP server",
					zap.String("protocol", api.MCPProtocolVersion),
				)
				if err := mcpServer.Run(); err != nil {
					logger.Error("MCP server error", zap.Error(err))
				}
			}()
			logger.Info("MCP server started")
		}

		if !startDisableREST {
			authService := auth.NewAuthServiceWithStorage(st)

			var restServer *api.RESTServer
			if cfg.Cortex.AuthEnabled {
				restServer = api.NewRESTServerWithAuth(se, st, emb, logger, authService)
				logger.Info("auth enabled", zap.Bool("auth", cfg.Cortex.AuthEnabled))
			} else {
				restServer = api.NewRESTServer(se, st, emb, logger)
			}

			addr := fmt.Sprintf(":%d", actualPort)
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

			var registeredTools []string
			if autoRegister {
				registeredTools = registerAITools(cfg.Starter.RegisterSkipTools)
			}

			printStartupBanner(actualPort, cfg, docCount, registeredTools)

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
		} else {
			var registeredTools []string
			if autoRegister {
				registeredTools = registerAITools(cfg.Starter.RegisterSkipTools)
			}

			printStartupBanner(actualPort, cfg, docCount, registeredTools)

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			<-sigChan
			logger.Info("received shutdown signal")
			st.Close()
		}
	}
}
