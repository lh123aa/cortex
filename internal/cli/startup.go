package cli

import (
	"fmt"
	"net"
	"strings"

	"github.com/lh123aa/cortex/internal/config"
)

func findAvailablePort(startPort int) (int, error) {
	if startPort <= 0 {
		startPort = 2021
	}
	for port := startPort; port < startPort+100; port++ {
		addr := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", addr)
		if err == nil {
			listener.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port found in range %d-%d", startPort, startPort+99)
}

func printStartupBanner(port int, cfg *config.Config, docCount int, registeredTools []string) {
	fmt.Println()
	fmt.Println("  🧠  Cortex is starting...")
	fmt.Println("  " + strings.Repeat("─", 50))
	fmt.Println()

	fmt.Printf("  REST API   →  http://localhost:%d\n", port)
	fmt.Printf("  Health     →  http://localhost:%d/health\n", port)
	fmt.Printf("  Search     →  http://localhost:%d/v1/search?q=your+query\n", port)
	fmt.Println()

	if cfg.Embedding.Provider == "none" {
		fmt.Println("  💡  FTS-only mode. Install Ollama + nomic-embed-text for semantic search.")
		fmt.Println()
	}

	if len(registeredTools) > 0 {
		fmt.Println("  Registered AI tools:")
		for _, t := range registeredTools {
			fmt.Printf("    ✅ %s\n", t)
		}
		fmt.Println()
	}

	if docCount > 0 {
		fmt.Printf("  📚  %d documents indexed\n", docCount)
	} else {
		fmt.Println("  📚  No documents indexed yet. Run: cortex index <path>")
	}
	fmt.Println()

	fmt.Println("  Next steps:")
	fmt.Println()
	fmt.Printf("    Search:    curl http://localhost:%d/v1/search?q=your+query\n", port)
	fmt.Println("    Status:    cortex status")
	fmt.Println()
	fmt.Println("  Press Ctrl+C to stop all services")
	fmt.Println("  " + strings.Repeat("─", 50))
	fmt.Println()
}
