package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage project secrets and API keys",
}

var authSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a secret for the current project",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key := strings.ToUpper(args[0])
		value := args[1]
		project, _ := cmd.Flags().GetString("project")
		out, _ := cmd.Flags().GetString("out")

		if project == "default" || project == "all" || project == "" {
			fmt.Println("❌ Secrets must be scoped to a specific project. Use --project [name]")
			return
		}

		projectDir := filepath.Join(out, "projects", project)
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			fmt.Printf("❌ Failed to create project dir: %v\n", err)
			return
		}

		envPath := filepath.Join(projectDir, ".env")

		// Load existing
		lines := []string{}
		if data, err := os.ReadFile(envPath); err == nil {
			oldLines := strings.Split(string(data), "\n")
			for _, line := range oldLines {
				if !strings.HasPrefix(line, key+"=") && strings.TrimSpace(line) != "" {
					lines = append(lines, line)
				}
			}
		}

		lines = append(lines, fmt.Sprintf("%s=%s", key, value))

		err := os.WriteFile(envPath, []byte(strings.Join(lines, "\n")), 0600)
		if err != nil {
			fmt.Printf("❌ Failed to save secret: %v\n", err)
			return
		}

		fmt.Printf("✅ Secret '%s' saved for project '%s'\n", key, project)

		// Ensure .env is ignored if project is in a git repo
		ensureGitIgnore(projectDir)
	},
}

func ensureGitIgnore(dir string) {
	ignorePath := filepath.Join(dir, ".gitignore")
	if data, err := os.ReadFile(ignorePath); err == nil {
		if strings.Contains(string(data), ".env") {
			return
		}
	}
	f, _ := os.OpenFile(ignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	f.WriteString("\n.env\n")
}

func init() {
	authCmd.PersistentFlags().StringP("project", "p", "default", "Project name")
	authCmd.PersistentFlags().String("out", "./wiki", "Wiki directory")
	authCmd.AddCommand(authSetCmd)
	rootCmd.AddCommand(authCmd)
}
