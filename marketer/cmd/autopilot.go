package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var autopilotCmd = &cobra.Command{
	Use:   "autopilot [topic/subreddit]",
	Short: "Automatically spot trends, research, and draft campaigns",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		topic := args[0]
		project := viper.GetString("project")
		wikiDir := viper.GetString("wiki_dir")
		dbPath := filepath.Join("projects", project, "data", "marketer.db")
		model, _ := cmd.Flags().GetString("model")
		
		aiProvider := viper.GetString("ai_provider")
		aiURL := viper.GetString("ai_url")

		// Decoupled: Read crawl path from config
		crawlPath := viper.GetString("crawl_path")
		if crawlPath == "" {
			crawlPath = "../musu-crawl-ai/musu-crawl.exe"
		}

		crawlProject := "autopilot-" + project

		fmt.Printf("🚀 Starting Autopilot for topic '%s' in project '%s'\n", topic, project)

		// 1. Spot Trend via the JSON contract (robust)
		fmt.Println("🔍 Step 1: Spotting trends...")
		spotArgs := []string{
			"spot", topic, 
			"--limit", "1", 
			"--json", 
			"--model", model, 
			"--ai-provider", aiProvider, 
			"--ai-url", aiURL,
		}

		spotCmd := exec.Command(crawlPath, spotArgs...)
		out, err := spotCmd.Output()
		if err != nil {
			fmt.Printf("❌ Spot command execution failed: %v\n", err)
			return
		}

		var spotResp struct {
			Status string `json:"status"`
			Data   []struct {
				Title string `json:"title"`
				URL   string `json:"url"`
				Score int    `json:"score"`
			} `json:"data"`
		}
		if err := json.Unmarshal(out, &spotResp); err != nil {
			fmt.Printf("❌ Could not parse spot JSON output: %v\n", err)
			return
		}
		if spotResp.Status != "success" || len(spotResp.Data) == 0 {
			fmt.Println("❌ No trends found or mission aborted by crawler.")
			return
		}
		topTrendTitle := spotResp.Data[0].Title
		fmt.Printf("   🔥 Top trend spotted: %q (Score: %d)\n", topTrendTitle, spotResp.Data[0].Score)

		// 2. Deep researching
		fmt.Printf("\n🧠 Step 2: Deep researching top trend: %q\n", topTrendTitle)
		researchArgs := []string{
			"research", topTrendTitle, 
			"--project", crawlProject, 
			"--depth", "2",
			"--model", model, 
			"--ai-provider", aiProvider, 
			"--ai-url", aiURL,
		}
		researchCmd := exec.Command(crawlPath, researchArgs...)
		researchCmd.Stdout = os.Stdout
		researchCmd.Stderr = os.Stderr
		if err := researchCmd.Run(); err != nil {
			fmt.Printf("❌ Research failed: %v\n", err)
			return
		}

		// 3. Draft Campaigns
		fmt.Println("\n✍️  Step 3: Drafting campaigns for personas...")
		personaDir := filepath.Join("projects", project, "personas")
		files, _ := os.ReadDir(personaDir)
		
		var personas []string
		for _, f := range files {
			if filepath.Ext(f.Name()) == ".md" {
				personas = append(personas, strings.TrimSuffix(f.Name(), ".md"))
			}
		}
		
		if len(personas) == 0 {
			personas = append(personas, "default")
		}

		for _, p := range personas {
			fmt.Printf("\n--- Drafting for Persona: %s ---\n", p)
			if err := ExecuteDraft(topTrendTitle, wikiDir, dbPath, model, p, project); err != nil {
				fmt.Printf("   ⚠️  Failed to draft for %s: %v\n", p, err)
			}
		}

		fmt.Printf("\n🏁 Autopilot mission complete for project '%s'.\n", project)
	},
}

func init() {
	autopilotCmd.Flags().String("model", "llama3", "Ollama model for copywriting")
	rootCmd.AddCommand(autopilotCmd)
}
