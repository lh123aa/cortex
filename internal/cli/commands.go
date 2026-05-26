package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", "", "config file path")
	RootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "", "log level (debug/info/warn/error)")

	RootCmd.AddCommand(NewIndexCmd())
	RootCmd.AddCommand(NewSearchCmd())
	RootCmd.AddCommand(NewContextCmd())
	RootCmd.AddCommand(NewMCPCmd())
	RootCmd.AddCommand(NewServeCmd())
	RootCmd.AddCommand(NewStatusCmd())
	RootCmd.AddCommand(NewDedupCmd())
	RootCmd.AddCommand(NewWatchCmd())
	RootCmd.AddCommand(NewUsageCmd())
	RootCmd.AddCommand(NewVersionCmd())
	RootCmd.AddCommand(NewSetupCmd())
	RootCmd.AddCommand(NewInstallCmd())
	RootCmd.AddCommand(NewStartCmd())
}

func SetupRootHelp() {
	originalHelp := RootCmd.HelpFunc()
	RootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		originalHelp(cmd, args)
		sep := strings.Repeat("─", 50)
		fmt.Println()
		fmt.Println("  " + sep)
		fmt.Println("  🚀  Quick Start")
		fmt.Println()
		fmt.Println("    cortex start              Start all services")
		fmt.Println("    cortex index <path>       Index documents")
		fmt.Println("    cortex search <query>     Search documents")
		fmt.Println("    cortex setup              Interactive setup")
		fmt.Println()
		fmt.Println("  📖  Docs: https://github.com/lh123aa/cortex")
		fmt.Println("  " + sep)
	})
}
