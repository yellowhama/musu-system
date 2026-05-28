package agent

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var globalOrchestrator *Orchestrator

// StartMCPServer starts a Model Context Protocol server on stdio.
func StartMCPServer(version string) error {
	globalOrchestrator = NewOrchestrator("./wiki")

	s := server.NewMCPServer(
		"musu-crawl-ai",
		version,
	)

	// Tool: Fetch
	fetchTool := mcp.NewTool("fetch",
		mcp.WithDescription("Fetch and parse content from YouTube, GitHub, Arxiv, or Web into clean Markdown"),
		mcp.WithString("source",
			mcp.Required(),
			mcp.Description("Source type identifying the harvester to use"),
			mcp.Enum("web", "yt", "youtube", "gh", "github", "arxiv", "hf", "huggingface", "reddit", "x", "twitter"),
		),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("URL or identifier (video id, repo path, paper id, etc.) for the harvester"),
		),
		mcp.WithString("project",
			mcp.Description("Project scope to file the harvested document under (default: 'default')"),
		),
	)
	s.AddTool(fetchTool, handleFetch)

	// Tool: Search
	searchTool := mcp.NewTool("search",
		mcp.WithDescription("Search the local knowledge base using keywords (Bleve) or semantic vectors"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query text"),
		),
		mcp.WithString("project",
			mcp.Description("Project to search within ('all' for cross-project, or a project name; default: 'all')"),
		),
		mcp.WithBoolean("semantic",
			mcp.Description("Use semantic vector search instead of keyword Bleve search (requires the index has embeddings)"),
		),
	)
	s.AddTool(searchTool, handleSearch)

	// Tool: Research
	researchTool := mcp.NewTool("research",
		mcp.WithDescription("Perform autonomous, recursive multi-step research on a topic — plan, search, harvest, synthesize, iterate"),
		mcp.WithString("question",
			mcp.Required(),
			mcp.Description("The research question to investigate"),
		),
		mcp.WithString("project",
			mcp.Description("Project to file harvested sources under (default: '')"),
		),
		mcp.WithNumber("depth",
			mcp.Description("Maximum recursive research depth (default: 2)"),
		),
	)
	s.AddTool(researchTool, handleResearch)

	fmt.Fprintf(os.Stderr, "🚀 musu-crawl-ai MCP Server %s started on stdio\n", version)

	return server.ServeStdio(s)
}

func handleFetch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments.(map[string]interface{})
	source, _ := args["source"].(string)
	url, _ := args["url"].(string)
	if strings.TrimSpace(source) == "" || strings.TrimSpace(url) == "" {
		return mcp.NewToolResultError("source and url are required"), nil
	}
	project, ok := args["project"].(string)
	if !ok { project = "default" }

	res, err := globalOrchestrator.FetchAction(source, url, project, "en", "llama3")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(res), nil
}

func handleSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments.(map[string]interface{})
	query, _ := args["query"].(string)
	if strings.TrimSpace(query) == "" {
		return mcp.NewToolResultError("query is required"), nil
	}
	project, ok := args["project"].(string)
	if !ok { project = "all" }
	semantic, _ := args["semantic"].(bool)

	entries, err := globalOrchestrator.SearchAction(query, project, semantic, "nomic-embed-text")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(entries) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf(
			"Found 0 results for %q (project=%q). If the wiki is empty, run 'fetch' first. To widen the scope, pass project='all'.",
			query, project)), nil
	}

	resText := fmt.Sprintf("Found %d results:\n", len(entries))
	for i, e := range entries {
		resText += fmt.Sprintf("%d. [%s] %s (ID: %s, Project: %s)\nSummary: %s\n\n", i+1, e.Source, e.Title, e.ID, e.Project, e.Summary)
	}

	return mcp.NewToolResultText(resText), nil
}

func handleResearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments.(map[string]interface{})
	question, _ := args["question"].(string)
	if strings.TrimSpace(question) == "" {
		return mcp.NewToolResultError("question is required"), nil
	}
	project, _ := args["project"].(string)
	depthVal, ok := args["depth"].(float64)
	depth := 2
	if ok { depth = int(depthVal) }

	res, err := globalOrchestrator.ResearchAction(question, project, depth, "llama3")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(res), nil
}
