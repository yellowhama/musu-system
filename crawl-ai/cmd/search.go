package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-crawl-ai/internal/agent"
	"github.com/yellowhama/musu-crawl-ai/internal/utils"
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search the Wiki knowledge base locally",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("json", cmd.Flags().Lookup("json"))
		out, _ := cmd.Flags().GetString("out")
		semantic, _ := cmd.Flags().GetBool("semantic")
		model, _ := cmd.Flags().GetString("model")
		project, _ := cmd.Flags().GetString("project")
		queryStr := args[0]

		orchestrator := agent.NewOrchestrator(out)
		entries, err := orchestrator.SearchAction(queryStr, project, semantic, model)
		if err != nil {
			utils.PrintError(err, "")
			return
		}

		if len(entries) == 0 {
			utils.PrintInfo("No results found for %q", queryStr)
			utils.PrintJSON("No results", nil)
			return
		}

		utils.PrintInfo("Found %d matches for %q:", len(entries), queryStr)
		if !viper.GetBool("json") {
			for i, e := range entries {
				fmt.Printf("%d. [%s] %s (ID: %s, Project: %s)\n", i+1, e.Source, e.Title, e.ID, e.Project)
				if e.Summary != "" {
					fmt.Printf("   Summary: %s\n", e.Summary)
				}
				fmt.Println()
			}
		}

		utils.PrintJSON("Search completed", entries)
	},
}

func init() {
	searchCmd.Flags().Bool("json", false, "Output in machine-readable JSON format")
	searchCmd.Flags().String("out", "./wiki", "Wiki directory to search in")
	searchCmd.Flags().BoolP("semantic", "s", false, "Use vector semantic search")
	searchCmd.Flags().String("model", "nomic-embed-text", "Ollama model for query embedding")
	searchCmd.Flags().StringP("project", "p", "all", "Project to scope search (default 'all')")
	rootCmd.AddCommand(searchCmd)
}
