package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-marketer/internal/db"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start a Model Context Protocol (MCP) server on stdio",
	Long:  `Enables AI agents (Claude, Gemini) to use musu-marketer as a native tool.`,
	Run: func(cmd *cobra.Command, args []string) {
		startMCPServer()
	},
}

func startMCPServer() {
	s := server.NewMCPServer("musu-marketer", Version)

	// Tool: Draft
	draftTool := mcp.NewTool("draft_campaign",
		mcp.WithDescription("Draft a strategic marketing campaign grounded in the crawl-ai wiki using the project's persona and the Strategist+Copywriter+Critic crew"),
		mcp.WithString("topic",
			mcp.Required(),
			mcp.Description("Campaign topic to draft from grounded wiki sources"),
		),
		mcp.WithString("project",
			mcp.Description("Project scope (default: 'default')"),
		),
		mcp.WithString("persona",
			mcp.Description("Persona name under projects/<project>/personas/<name>.md (default: 'default')"),
		),
	)
	s.AddTool(draftTool, handleDraft)

	// Tool: List
	listTool := mcp.NewTool("list_campaigns",
		mcp.WithDescription("List all drafted marketing campaigns for a project"),
		mcp.WithString("project",
			mcp.Description("Project scope (default: 'default')"),
		),
	)
	s.AddTool(listTool, handleList)

	fmt.Fprintf(os.Stderr, "🚀 musu-marketer MCP Server %s started on stdio\n", Version)
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "❌ MCP Server failed: %v\n", err)
	}
}

func handleDraft(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments.(map[string]interface{})
	topic, _ := args["topic"].(string)
	if strings.TrimSpace(topic) == "" {
		return mcp.NewToolResultError("topic is required"), nil
	}
	project, ok := args["project"].(string)
	if !ok { project = "default" }
	persona, ok := args["persona"].(string)
	if !ok { persona = "default" }
	
	wikiDir := viper.GetString("wiki_dir")
	dbPath := filepath.Join("projects", project, "data", "marketer.db")
	
	err := ExecuteDraft(topic, wikiDir, dbPath, "llama3", persona, project)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	
	return mcp.NewToolResultText("Campaign draft completed and saved to database."), nil
}

func handleList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments.(map[string]interface{})
	project, ok := args["project"].(string)
	if !ok { project = "default" }
	dbPath := filepath.Join("projects", project, "data", "marketer.db")

	store, err := db.NewStore(dbPath)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	campaigns, err := store.ListCampaigns()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	data, _ := json.MarshalIndent(campaigns, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
