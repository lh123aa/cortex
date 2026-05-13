package cli

import (
	"github.com/spf13/cobra"
)

func NewSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search indexed documents",
		Args:  cobra.ExactArgs(1),
		Run:   runSearch,
	}
	cmd.Flags().IntVarP(&topK, "top-k", "k", 10, "number of results to return")
	cmd.Flags().StringVarP(&mode, "mode", "m", "hybrid", "search mode (vector/bm25/hybrid)")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "output as JSON")
	return cmd
}

func NewContextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context <query>",
		Short: "Generate RAG context for a query",
		Args:  cobra.ExactArgs(1),
		Run:   runContext,
	}
	cmd.Flags().IntVarP(&tokenBudget, "tokens", "t", 4000, "token budget for context")
	return cmd
}
