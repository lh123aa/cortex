package cli

import (
	"github.com/spf13/cobra"
)

func NewInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install [doc-dir]",
		Short: "One-step setup: auto-config + index documents",
		Long: `One-command installation for beginners.

Automatically creates a minimal config (FTS5-only, zero dependencies)
and indexes the specified directory — all in one step.

Examples:
  cortex install ./docs
  cortex install C:\MyNotes
  cortex install          # (uses current directory)`,
		Args: cobra.MaximumNArgs(1),
		Run:  runInstall,
	}
}
