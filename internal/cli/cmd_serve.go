package cli

import (
	"github.com/spf13/cobra"
)

var servePort = 2021

func NewServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start REST API server",
		Run:   runServe,
	}
	cmd.Flags().IntVarP(&servePort, "port", "p", 2021, "server listen port")
	return cmd
}

func NewMCPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP server for AI Agent integration",
		Run:   runMCP,
	}
}
