package cli

import (
	"fmt"

	"github.com/lh123aa/cortex/internal/version"
	"github.com/spf13/cobra"
)

func NewStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show indexing status",
		Run:   runStatus,
	}
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "output as JSON")
	return cmd
}

func NewUsageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "usage",
		Short: "Show storage usage and plan info",
		Run:   runUsage,
	}
}

func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Cortex %s\n", version.Full())
		},
	}
}
