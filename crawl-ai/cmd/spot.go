package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-crawl-ai/internal/harvester"
	"github.com/yellowhama/musu-crawl-ai/internal/utils"
)

var spotCmd = &cobra.Command{
	Use:   "spot [topic/subreddit]",
	Short: "Spot trending topics on Reddit/X",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("json", cmd.Flags().Lookup("json"))
		platform, _ := cmd.Flags().GetString("platform")
		limit, _ := cmd.Flags().GetInt("limit")
		project, _ := cmd.Flags().GetString("project")
		autoResearch, _ := cmd.Flags().GetBool("research")

		utils.PrintInfo("🔍 Spotting trends on %s for: %s", platform, args[0])

		if strings.ToLower(platform) == "reddit" {
			f := &harvester.RedditFetcher{}
			trends, err := f.Spot(args[0], limit)
			if err != nil {
				utils.PrintError(fmt.Errorf("failed to spot trends: %v", err), "")
				return
			}

			utils.PrintInfo("\n🔥 Top %d Trending Topics in r/%s:", len(trends), args[0])
			var results []map[string]interface{}

			for i, t := range trends {
				if !viper.GetBool("json") {
					fmt.Printf("%d. [%d 👍] %s\n   🔗 %s\n", i+1, t.Score, t.Title, t.URL)
				}
				results = append(results, map[string]interface{}{
					"title": t.Title,
					"url":   t.URL,
					"score": t.Score,
				})
			}

			utils.PrintJSON("Trends spotted", results)

			if autoResearch && len(trends) > 0 {
				topTrend := trends[0]
				utils.PrintInfo("\n🚀 Auto-triggering research for top trend: %q", topTrend.Title)
				if viper.GetBool("json") {
					// In JSON mode, we don't just print tips, we include it in the success data if possible
					// But PrintJSON already exited or printed. 
					// Let's ensure the user knows what to run.
				}
				fmt.Printf("👉 Run: .\\musu-crawl.exe research %q --project %s\n", topTrend.Title, project)
			}
		} else {
			utils.PrintInfo("⚠️  Platform not supported yet. Only 'reddit' is available.")
		}
	},
}

func init() {
	spotCmd.Flags().Bool("json", false, "Output in machine-readable JSON format")
	spotCmd.Flags().String("platform", "reddit", "Platform to spot trends (reddit, x)")
	spotCmd.Flags().Int("limit", 5, "Number of trending topics to show")
	spotCmd.Flags().StringP("project", "p", "trends", "Project name for research")
	spotCmd.Flags().Bool("research", false, "Automatically trigger research for the top trend")
	rootCmd.AddCommand(spotCmd)
}
