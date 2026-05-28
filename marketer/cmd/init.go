package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-marketer/internal/db"
)

func writeIfMissing(path string, content string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func bootstrapProject(project string, verbose bool) (string, string, error) {
	baseDir := filepath.Join("projects", project)
	dirs := []string{
		filepath.Join(baseDir, "campaigns"),
		filepath.Join(baseDir, "personas"),
		filepath.Join(baseDir, "data"),
		filepath.Join(baseDir, "published"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return "", "", fmt.Errorf("create directory %s: %w", d, err)
		}
		if verbose {
			fmt.Printf("✅ Directory ready: ./%s\n", d)
		}
	}

	dbPath := filepath.Join(baseDir, "data", "marketer.db")
	store, err := db.NewStore(dbPath)
	if err != nil {
		return "", "", fmt.Errorf("initialize database: %w", err)
	}
	if err := store.Close(); err != nil {
		return "", "", fmt.Errorf("close database after initialization: %w", err)
	}
	if verbose {
		fmt.Printf("✅ Database ready: %s\n", dbPath)
	}

	defaultPersonaPath := filepath.Join(baseDir, "personas", "default.md")
	if _, err := os.Stat(defaultPersonaPath); os.IsNotExist(err) {
		defaultPersona := `Role: Professional AI Marketing Expert
Tone: Informative, authoritative, and slightly enthusiastic.
Framework: AIDA
Formatting: Clean Markdown with bold headers.`
		if err := os.WriteFile(defaultPersonaPath, []byte(defaultPersona), 0644); err != nil {
			return "", "", fmt.Errorf("write default persona: %w", err)
		}
		if verbose {
			fmt.Printf("✅ Default persona created: %s\n", defaultPersonaPath)
		}
	}

	configPath := filepath.Join(baseDir, "config.yaml")
	config := fmt.Sprintf("db_path: %s\nwiki_dir: %s\nai_provider: %s\nai_url: %s\n",
		dbPath,
		viper.GetString("wiki_dir"),
		viper.GetString("ai_provider"),
		viper.GetString("ai_url"),
	)
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		return "", "", fmt.Errorf("write config: %w", err)
	}
	if verbose {
		fmt.Printf("✅ Project configuration saved: %s\n", configPath)
	}

	nextStepsPath := filepath.Join(baseDir, "NEXT_STEPS.md")
	nextStepsContent := fmt.Sprintf(`# Next Steps: %s

1. Run `+"`musu-marketer doctor --project %s`"+` to verify wiki and AI readiness.
2. Review or edit the persona in `+"`personas/default.md`"+`.
3. Draft your first campaign with `+"`musu-marketer draft <topic> --project %s --persona default`"+`.
4. Publish finished work from `+"`campaigns/`"+` through `+"`publish`"+` when ready.
`, project, project, project)
	if err := writeIfMissing(nextStepsPath, nextStepsContent); err != nil {
		return "", "", fmt.Errorf("write next steps guide: %w", err)
	}
	if verbose {
		fmt.Printf("✅ Next steps guide ready: %s\n", nextStepsPath)
	}
	return baseDir, dbPath, nil
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize musu-marketer environment for a project",
	RunE: func(cmd *cobra.Command, args []string) error {
		project := viper.GetString("project")
		jsonMode := viper.GetBool("json")
		if !jsonMode {
			fmt.Printf("🚀 Initializing musu-marketer for project '%s' (Version %s)...\n", project, Version)
		}

		baseDir, dbPath, err := bootstrapProject(project, !jsonMode)
		if err != nil {
			return err
		}

		configPath := filepath.Join(baseDir, "config.yaml")
		defaultPersonaPath := filepath.Join(baseDir, "personas", "default.md")
		result := map[string]interface{}{
			"project":              project,
			"project_dir":          baseDir,
			"db_path":              dbPath,
			"config_path":          configPath,
			"default_persona_path": defaultPersonaPath,
			"project_next_steps_path": filepath.Join(baseDir, "NEXT_STEPS.md"),
			"wiki_dir":             viper.GetString("wiki_dir"),
			"ai_provider":          viper.GetString("ai_provider"),
			"ai_url":               viper.GetString("ai_url"),
			"next_steps": []string{
				fmt.Sprintf("run 'musu-marketer doctor --project %s'", project),
				fmt.Sprintf("run 'musu-marketer draft [topic] --project %s'", project),
			},
		}
		if !jsonMode {
			fmt.Printf("\n✨ Project '%s' initialized! Run 'musu-marketer doctor --project %s' next.\n", project, project)
		}
		printJSONSuccess("Project initialized", result)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
