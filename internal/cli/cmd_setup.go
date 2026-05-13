package cli

import (
	"github.com/spf13/cobra"
)

func NewSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Interactive embedding provider configuration wizard",
		Long: `Run the interactive setup wizard to configure your embedding provider.

Supports local (Ollama), international API (OpenAI, Cohere, Voyage),
and domestic API (DashScope, Zhipu, Baidu) providers.
Run 'cortex setup' anytime to reconfigure.`,
		Run: runSetup,
	}
}
