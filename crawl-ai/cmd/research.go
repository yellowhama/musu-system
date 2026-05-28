package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-crawl-ai/internal/agent"
	"github.com/yellowhama/musu-crawl-ai/internal/utils"
)

var researchCmd = &cobra.Command{
	Use:   "research [question]",
	Short: "Perform autonomous recursive multi-agent research with skepticism",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("json", cmd.Flags().Lookup("json"))
		question := args[0]
		project, _ := cmd.Flags().GetString("project")
		conf, _ := utils.LoadConfig(project)

		model, _ := cmd.Flags().GetString("model")
		if !cmd.Flags().Changed("model") {
			model = conf.AIModel
		}

		out, _ := cmd.Flags().GetString("out")
		if !cmd.Flags().Changed("out") {
			out = conf.WikiDir
		}

		maxDepth, _ := cmd.Flags().GetInt("depth")

		utils.PrintInfo("🚀 Starting autonomous research mission: %q", question)
		
		orchestrator := agent.NewOrchestrator(out)
		report, err := orchestrator.ResearchAction(question, project, maxDepth, model)
		if err != nil {
			utils.PrintError(err, "")
			return
		}

		if !viper.GetBool("json") {
			fmt.Println("\n--- FINAL RESEARCH REPORT ---")
			fmt.Println(report)
		}

		utils.PrintSuccess("Research mission completed.")
		utils.PrintJSON("Research completed", map[string]interface{}{
			"question": question,
			"project":  project,
			"report":   report,
		})
	},
}

func init() {
	researchCmd.Flags().Bool("json", false, "Output in machine-readable JSON format")
	researchCmd.Flags().String("model", "llama3", "Local Ollama model to use")
	researchCmd.Flags().Int("limit", 5, "Maximum number of sources to fetch per hop")
	researchCmd.Flags().Int("depth", 2, "Maximum recursive research depth")
	researchCmd.Flags().String("out", "./wiki", "Output directory")
	researchCmd.Flags().StringP("project", "p", "default", "Project name to scope the research")
	rootCmd.AddCommand(researchCmd)
}
