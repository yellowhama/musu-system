package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yellowhama/musu-crawl-ai/internal/agent"
	"github.com/yellowhama/musu-crawl-ai/internal/processor"
	"github.com/yellowhama/musu-crawl-ai/internal/utils"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch [source] [id]",
	Short: "Fetch content from a source (yt, gh, web, arxiv, etc.)",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		source := args[0]
		id := args[1]
		project, _ := cmd.Flags().GetString("project")
		lang, _ := cmd.Flags().GetString("lang")
		out, _ := cmd.Flags().GetString("out")
		compile, _ := cmd.Flags().GetBool("compile")

		conf, _ := utils.LoadConfig(project)
		model, _ := cmd.Flags().GetString("model")
		if !cmd.Flags().Changed("model") {
			model = conf.AIModel
		}

		proc := processor.NewWikiProcessor(out, project)

		fmt.Printf("Fetching from %s: %s (Project: %s)...\n", source, id, project)
		fname, text, reliability, tags, err := agent.FetchAndSave(source, id, lang, proc, model, conf.AIBaseURL)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}

		fmt.Printf("✅ Saved to: %s (Reliability: %.2f)\n", fname, reliability)

		if compile {
			fmt.Println("🧠 Auto-compiling knowledge graph relationships...")
			ollama := agent.NewAgentClient(conf.AIBaseURL, model, "", out, project)
			compiler, err := agent.NewCompiler(ollama, out)
			if err == nil {
				summary := utils.Summarize(text, 2)
				section, _ := compiler.CompileDocument(fname, text, tags, summary)
				if section != "" {
					compiler.UpdateDocument(fname, section)
					fmt.Println("   ✅ Knowledge links injected.")
				}
				compiler.Close()
			}
		}
	},
}

func init() {
	fetchCmd.Flags().String("lang", "ko", "Preferred language (ko, en)")
	fetchCmd.Flags().String("out", "./wiki", "Output directory")
	fetchCmd.Flags().String("project", "default", "Project name")
	fetchCmd.Flags().String("model", "llama3", "Model for summaries and tags")
	fetchCmd.Flags().Bool("compile", true, "Automatically compile knowledge links")
	rootCmd.AddCommand(fetchCmd)
}
