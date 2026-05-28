package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yellowhama/musu-crawl-ai/internal/agent"
	"github.com/yellowhama/musu-crawl-ai/internal/utils"
)

var compileCmd = &cobra.Command{
	Use:   "compile [file_path]",
	Short: "Compile verified knowledge into interlinked Wiki pages",
	Run: func(cmd *cobra.Command, args []string) {
		out, _ := cmd.Flags().GetString("out")
		model, _ := cmd.Flags().GetString("model")
		project, _ := cmd.Flags().GetString("project")
		conf, _ := utils.LoadConfig(project)
		ollama := agent.NewAgentClient(conf.AIBaseURL, model, "", out, project)
		compiler, err := agent.NewCompiler(ollama, out)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}
		defer compiler.Close()

		if len(args) > 0 {
			path := args[0]
			fmt.Printf("Compiling document: %s...\n", path)
		} else {
			fmt.Println("Compiling entire Wiki directory...")
		}
	},
}

func init() {
	compileCmd.Flags().String("out", "./wiki", "Wiki directory to compile")
	compileCmd.Flags().String("model", "llama3", "Ollama model for analysis")
	compileCmd.Flags().String("project", "default", "Project name")
	compileCmd.Flags().Bool("force", false, "Force re-compilation of existing links")
	rootCmd.AddCommand(compileCmd)
}
