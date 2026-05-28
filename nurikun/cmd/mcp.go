package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
	"github.com/yellowhama/musu-system/nurikun/internal/config"
	"github.com/yellowhama/musu-system/nurikun/internal/db"
	"github.com/yellowhama/musu-system/nurikun/internal/preflight"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start a Model Context Protocol (MCP) server on stdio",
	Long: `Exposes nurikun's safe admin / read operations as MCP tools so other
LLM agents can inspect lists, manage consent, and run preflight without
having to shell out to the CLI.

Outbound delivery operations (watch — auto-replies to inbound; campaign — sends
to subscribers) are intentionally NOT exposed via MCP: nurikun's opt-in posture
requires a human operator in the loop for anything that actually puts mail in
flight. Use the CLI for those.`,
	Run: func(cmd *cobra.Command, args []string) {
		startMCPServer()
	},
}

func startMCPServer() {
	s := server.NewMCPServer("musu-nurikun", Version)

	s.AddTool(mcp.NewTool("doctor",
		mcp.WithDescription("Run nurikun preflight for a project (config / mailbox / knowledge / AI reachability)"),
		mcp.WithString("project", mcp.Description("Project scope (default: 'default')")),
	), wrap(handleMCPDoctor))

	s.AddTool(mcp.NewTool("list_lists",
		mcp.WithDescription("List mailing lists for a project"),
		mcp.WithString("project", mcp.Description("Project scope (default: 'default')")),
	), wrap(handleMCPListLists))

	s.AddTool(mcp.NewTool("create_list",
		mcp.WithDescription("Create a new opt-in mailing list"),
		mcp.WithString("name", mcp.Required(), mcp.Description("List name (must be unique within the project)")),
		mcp.WithNumber("cadence_days", mcp.Description("Minimum days between sends to the same subscriber (default: 4)")),
		mcp.WithString("project", mcp.Description("Project scope (default: 'default')")),
	), wrap(handleMCPCreateList))

	s.AddTool(mcp.NewTool("subscribe",
		mcp.WithDescription("Record a pending subscriber (returns the double-opt-in confirm token; the subscriber is NOT mailable until confirmed)"),
		mcp.WithNumber("list_id", mcp.Required(), mcp.Description("Target list ID")),
		mcp.WithString("email", mcp.Required(), mcp.Description("Subscriber email address")),
		mcp.WithString("name", mcp.Description("Optional display name")),
		mcp.WithString("project", mcp.Description("Project scope (default: 'default')")),
	), wrap(handleMCPSubscribe))

	s.AddTool(mcp.NewTool("confirm_subscriber",
		mcp.WithDescription("Confirm a pending subscriber by its double opt-in token"),
		mcp.WithString("token", mcp.Required(), mcp.Description("Confirmation token returned by 'subscribe'")),
		mcp.WithString("project", mcp.Description("Project scope (default: 'default')")),
	), wrap(handleMCPConfirm))

	s.AddTool(mcp.NewTool("list_subscribers",
		mcp.WithDescription("List every subscriber of a list with status (pending / confirmed / unsubscribed)"),
		mcp.WithNumber("list_id", mcp.Required(), mcp.Description("List ID to enumerate")),
		mcp.WithString("project", mcp.Description("Project scope (default: 'default')")),
	), wrap(handleMCPListSubscribers))

	s.AddTool(mcp.NewTool("suppress",
		mcp.WithDescription("Unsubscribe an email and add it to the suppression list (the hard send-time gate)"),
		mcp.WithString("email", mcp.Required(), mcp.Description("Email to suppress")),
		mcp.WithString("reason", mcp.Description("Reason for the suppression (default: 'mcp')")),
		mcp.WithString("project", mcp.Description("Project scope (default: 'default')")),
	), wrap(handleMCPSuppress))

	s.AddTool(mcp.NewTool("messages_by_status",
		mcp.WithDescription("List stored messages filtered by status"),
		mcp.WithString("status",
			mcp.Required(),
			mcp.Description("Status filter"),
			mcp.Enum("received", "drafted", "sent", "escalated"),
		),
		mcp.WithString("project", mcp.Description("Project scope (default: 'default')")),
	), wrap(handleMCPMessagesByStatus))

	fmt.Fprintf(os.Stderr, "🚀 musu-nurikun MCP Server %s started on stdio\n", Version)
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "❌ MCP server failed: %v\n", err)
	}
}

// wrap adapts a map-based handler into the mcp-go callback signature.
type mcpFn func(args map[string]interface{}) (*mcp.CallToolResult, error)

func wrap(fn mcpFn) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		if args == nil {
			args = map[string]interface{}{}
		}
		return fn(args)
	}
}

func projectArg(args map[string]interface{}) string {
	if v, ok := args["project"].(string); ok && v != "" {
		return v
	}
	return "default"
}

func openProjectStore(project string) (*db.Store, error) {
	return db.NewStore(filepath.Join("projects", project, "data", "nurikun.db"))
}

// --- handlers ---

func handleMCPDoctor(args map[string]interface{}) (*mcp.CallToolResult, error) {
	project := projectArg(args)
	cfg, err := config.Load(project)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	res := preflight.EvaluateDoctor(preflight.DoctorOptions{
		Project:    project,
		ConfigPath: filepath.Join("projects", project, "config.yaml"),
		Config:     cfg,
		AutoFix:    false,
	})
	data, _ := json.MarshalIndent(res, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleMCPListLists(args map[string]interface{}) (*mcp.CallToolResult, error) {
	store, err := openProjectStore(projectArg(args))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer store.Close()
	lists, err := store.ListLists()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	data, _ := json.MarshalIndent(lists, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleMCPCreateList(args map[string]interface{}) (*mcp.CallToolResult, error) {
	name, _ := args["name"].(string)
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}
	cadence := 4
	if v, ok := args["cadence_days"].(float64); ok && v > 0 {
		cadence = int(v)
	}
	store, err := openProjectStore(projectArg(args))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer store.Close()
	id, err := store.CreateList(name, cadence)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf(`{"list_id":%d,"name":%q,"cadence_days":%d}`, id, name, cadence)), nil
}

func handleMCPSubscribe(args map[string]interface{}) (*mcp.CallToolResult, error) {
	var listID int
	if v, ok := args["list_id"].(float64); ok {
		listID = int(v)
	}
	email, _ := args["email"].(string)
	if listID == 0 || email == "" {
		return mcp.NewToolResultError("list_id and email are required"), nil
	}
	name, _ := args["name"].(string)
	store, err := openProjectStore(projectArg(args))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer store.Close()
	token, err := store.AddSubscriber(listID, email, name, "mcp")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf(
		`{"status":"pending","email":%q,"confirm_token":%q,"note":"subscriber is NOT yet mailable; call confirm_subscriber with this token or send the user a /confirm link"}`,
		email, token)), nil
}

func handleMCPConfirm(args map[string]interface{}) (*mcp.CallToolResult, error) {
	token, _ := args["token"].(string)
	if token == "" {
		return mcp.NewToolResultError("token is required"), nil
	}
	store, err := openProjectStore(projectArg(args))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer store.Close()
	sub, err := store.ConfirmSubscriber(token)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	data, _ := json.MarshalIndent(sub, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleMCPListSubscribers(args map[string]interface{}) (*mcp.CallToolResult, error) {
	var listID int
	if v, ok := args["list_id"].(float64); ok {
		listID = int(v)
	}
	if listID == 0 {
		return mcp.NewToolResultError("list_id is required"), nil
	}
	store, err := openProjectStore(projectArg(args))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer store.Close()
	subs, err := store.ListSubscribers(listID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	data, _ := json.MarshalIndent(subs, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleMCPSuppress(args map[string]interface{}) (*mcp.CallToolResult, error) {
	email, _ := args["email"].(string)
	if email == "" {
		return mcp.NewToolResultError("email is required"), nil
	}
	reason, _ := args["reason"].(string)
	if reason == "" {
		reason = "mcp"
	}
	store, err := openProjectStore(projectArg(args))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer store.Close()
	if err := store.Unsubscribe(email, reason); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf(`{"status":"suppressed","email":%q,"reason":%q}`, email, reason)), nil
}

func handleMCPMessagesByStatus(args map[string]interface{}) (*mcp.CallToolResult, error) {
	status, _ := args["status"].(string)
	if status == "" {
		return mcp.NewToolResultError("status is required (received|drafted|sent|escalated)"), nil
	}
	store, err := openProjectStore(projectArg(args))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer store.Close()
	msgs, err := store.MessagesByStatus(status)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	data, _ := json.MarshalIndent(msgs, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
