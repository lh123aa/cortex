package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	cfgPath        string
	logLevel       string
	topK           int
	mode           string
	tokenBudget    int
	jsonOutput     bool
	dedupMode      string
	dedupThreshold float64
	forceReindex   bool
	indexTimeout   time.Duration
	indexWorkers   int
)

var RootCmd = &cobra.Command{
	Use:   "cortex",
	Short: "Cortex - Agent Knowledge Base",
	Long: `Cortex is a local knowledge base system for AI Agents.
It supports hybrid search (vector + BM25), multiple file formats,
and MCP protocol for AI Agent integration.`,
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
