package cli

import (
	"github.com/spf13/cobra"
)

func NewWatchCmd() *cobra.Command {
	cmd := &cobra.Command{
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
	return cmd
}
