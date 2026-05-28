package cmd

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-crawl-ai/internal/utils"
)

func writeIfMissing(path string, content string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func bootstrapProjectDirs(out, project string, aiProvider string, aiURL string, verbose bool) error {
	dirs := []string{
		out,
		filepath.Join(out, "projects"),
		filepath.Join(out, "projects", project),
		filepath.Join(out, "projects", project, "youtube"),
		filepath.Join(out, "projects", project, "github"),
		filepath.Join(out, "projects", project, "papers"),
		filepath.Join(out, "projects", project, "web"),
		filepath.Join(out, "projects", project, "twitter"),
		filepath.Join(out, "projects", project, "huggingface"),
		filepath.Join(out, "projects", project, "reddit"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", d, err)
		}
		if verbose {
			fmt.Printf("✅ Directory ready: %s\n", d)
		}
	}

	projectDir := filepath.Join(out, "projects", project)
	configPath := filepath.Join(projectDir, "config.toml")
	configContent := fmt.Sprintf(`language = "ko"
out = "%s"
model = "llama3"
ai_provider = "%s"
ai_url = "%s"
`, out, aiProvider, aiURL)
	if err := writeIfMissing(configPath, configContent); err != nil {
		return fmt.Errorf("write project config: %w", err)
	}
	if verbose {
		fmt.Printf("✅ Project config ready: %s\n", configPath)
	}

	promptPath := filepath.Join(projectDir, "PROMPT.md")
	promptContent := fmt.Sprintf(`# Research Prompt: %s

Describe the project-specific research goals, quality bar, and recurring questions here.

Suggested checklist:
- what sources matter most for %s
- what contradictions should be surfaced
- what claims need citation or cross-checking
`, project, project)
	if err := writeIfMissing(promptPath, promptContent); err != nil {
		return fmt.Errorf("write project prompt: %w", err)
	}
	if verbose {
		fmt.Printf("✅ Research prompt ready: %s\n", promptPath)
	}

	nextStepsPath := filepath.Join(projectDir, "NEXT_STEPS.md")
	nextStepsContent := fmt.Sprintf(`# Next Steps: %s

1. Write project-specific instructions in `+"`PROMPT.md`"+`.
2. Run `+"`musu-crawl doctor --out %s --project %s`"+`.
3. Fetch your first source with `+"`musu-crawl fetch <source> <id> --project %s`"+`.
4. Run `+"`musu-crawl index --out %s --project %s`"+` after new material lands.
`, project, out, project, project, out, project)
	if err := writeIfMissing(nextStepsPath, nextStepsContent); err != nil {
		return fmt.Errorf("write next steps guide: %w", err)
	}
	if verbose {
		fmt.Printf("✅ Next steps guide ready: %s\n", nextStepsPath)
	}
	return nil
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the Wiki directory and check environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		out, _ := cmd.Flags().GetString("out")
		project, _ := cmd.Flags().GetString("project")
		if project == "" {
			project = "default"
		}
		jsonMode := viper.GetBool("json")

		if !jsonMode {
			fmt.Printf("🚀 Initializing musu-crawl-ai (Version %s)...\n", Version)
		}

		aiURL := viper.GetString("ai_url")
		if aiURL == "" {
			aiURL = "http://localhost:11434/v1"
		}

		// 1. Create directory structure
		aiProvider := viper.GetString("ai_provider")
		if err := bootstrapProjectDirs(out, project, aiProvider, aiURL, !jsonMode); err != nil {
			return err
		}

		// 2. Check for AI Service
		aiReachable := false
		aiWarning := ""
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get(aiURL + "/models")
		if err != nil || resp.StatusCode != 200 {
			aiWarning = fmt.Sprintf("AI service not detected at %s. AI features will be limited until a compatible endpoint is started.", aiURL)
			if !jsonMode {
				fmt.Printf("⚠️  %s\n", aiWarning)
				fmt.Println("   Tip: Start the configured Ollama/SGLang/OpenAI-compatible endpoint to enable full intelligence.")
			}
		} else {
			aiReachable = true
			if !jsonMode {
				fmt.Println("✅ AI service detected and active.")
			}
			resp.Body.Close()
		}

		// 3. Check for Search Index
		blevePath := filepath.Join(out, "musu.bleve")
		indexExists := false
		if _, err := os.Stat(blevePath); os.IsNotExist(err) {
			if !jsonMode {
				fmt.Println("ℹ️  Search index not found. You may want to run 'musu-crawl index' once you have fetched content.")
			}
		} else {
			indexExists = true
			if !jsonMode {
				fmt.Println("✅ Search index found.")
			}
		}

		result := map[string]interface{}{
			"wiki_dir":      out,
			"project":       project,
			"project_dir":   filepath.Join(out, "projects", project),
			"project_config_path": filepath.Join(out, "projects", project, "config.toml"),
			"project_prompt_path": filepath.Join(out, "projects", project, "PROMPT.md"),
			"project_next_steps_path": filepath.Join(out, "projects", project, "NEXT_STEPS.md"),
			"ai_provider":   viper.GetString("ai_provider"),
			"ai_url":        aiURL,
			"ai_reachable":  aiReachable,
			"index_exists":  indexExists,
			"ai_warning":    aiWarning,
			"next_steps": []string{
				fmt.Sprintf("run 'musu-crawl doctor --out %s --project %s'", out, project),
				fmt.Sprintf("run 'musu-crawl fetch <source> <id> --project %s'", project),
			},
		}
		if !jsonMode {
			fmt.Println("\n✨ Initialization complete! You are ready to crawl.")
		}
		utils.PrintJSON("Initialization complete", result)
		return nil
	},
}

func init() {
	initCmd.Flags().String("out", "./wiki", "Wiki directory to initialize")
	initCmd.Flags().String("project", "default", "Project directory to initialize under wiki/projects")
	rootCmd.AddCommand(initCmd)
}
