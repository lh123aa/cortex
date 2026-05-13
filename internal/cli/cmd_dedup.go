package cli

import (
	"github.com/spf13/cobra"
)

func NewDedupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dedup",
		Short: "Deduplicate chunks in the knowledge base",
		Long: `Scan all chunks and remove duplicates.
Without flags: dedup by content hash (exact match).
With --vector: dedup by vector similarity (semantic match).`,
		Run: runDedup,
	}
	cmd.Flags().StringVarP(&dedupMode, "mode", "m", "hash", "dedup mode: hash (exact) | vector (semantic)")
	cmd.Flags().Float64VarP(&dedupThreshold, "threshold", "t", 0.95, "similarity threshold for vector dedup (0.0-1.0)")
	return cmd
}
