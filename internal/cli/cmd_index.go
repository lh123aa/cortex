package cli

import (
	"github.com/spf13/cobra"
)

func NewIndexCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "index <path>",
		Short: "Index documents from a directory",
		Args:  cobra.ExactArgs(1),
		Run:   runIndex,
	}
	cmd.Flags().BoolVarP(&forceReindex, "force", "f", false, "Force re-index from scratch (ignore checkpoint)")
	cmd.Flags().DurationVarP(&indexTimeout, "timeout", "t", 0, "Maximum time for indexing (e.g. 30m, 1h). 0 = no limit")
	cmd.Flags().IntVarP(&indexWorkers, "workers", "w", 0, "Number of indexing workers (default 16)")
	return cmd
}
