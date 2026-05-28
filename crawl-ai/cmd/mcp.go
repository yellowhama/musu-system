package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yellowhama/musu-crawl-ai/internal/agent"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start a Model Context Protocol (MCP) server on stdio",
	Long:  `Enables AI agents (Claude, Gemini) to use musu-crawl-ai as a native tool.`,
	Run: func(cmd *cobra.Command, args []string) {
		agent.StartMCPServer(Version)
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
