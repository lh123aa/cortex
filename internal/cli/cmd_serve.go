package cli

import (
	"github.com/spf13/cobra"
)

func NewServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start REST API server",
		Run:   runServe,
	}
}

func NewMCPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP server for AI Agent integration",
		Run:   runMCP,
	}
}
