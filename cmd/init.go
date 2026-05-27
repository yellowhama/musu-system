package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-nurikun/internal/db"
)

func writeIfMissing(path string, content string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func discoverCrawlWiki() string {
	candidates := []string{
		"F:/Aisaak/Projects/musu-crawl-ai/wiki",
		"../musu-crawl-ai/wiki",
		"./wiki",
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return "F:/Aisaak/Projects/musu-crawl-ai/wiki"
}

func bootstrapProject(project string, verbose bool, mailboxProvider string, knowledgeSource string) (string, string, error) {
	baseDir := filepath.Join("projects", project)
	dirs := []string{
		filepath.Join(baseDir, "data"),
		filepath.Join(baseDir, "knowledge"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return "", "", fmt.Errorf("create directory %s: %w", d, err)
		}
		if verbose {
			fmt.Printf("✅ Directory ready: ./%s\n", d)
		}
	}

	dbPath := filepath.Join(baseDir, "data", "nurikun.db")
	store, err := db.NewStore(dbPath)
	if err != nil {
		return "", "", fmt.Errorf("initialize database: %w", err)
	}
	_ = store.Close()
	if verbose {
		fmt.Printf("✅ Database ready: %s\n", dbPath)
	}

	configPath := filepath.Join(baseDir, "config.yaml")
	if mailboxProvider == "" {
		mailboxProvider = "imap"
	}
	if knowledgeSource == "" {
		knowledgeSource = "none"
	}
	if !strings.EqualFold(mailboxProvider, "imap") && !strings.EqualFold(mailboxProvider, "gmail") {
		return "", "", fmt.Errorf("unsupported mailbox provider %q", mailboxProvider)
	}
	if !strings.EqualFold(knowledgeSource, "none") && !strings.EqualFold(knowledgeSource, "crawlai") && !strings.EqualFold(knowledgeSource, "folder") {
		return "", "", fmt.Errorf("unsupported knowledge source %q", knowledgeSource)
	}

	crawlPath := ""
	knowledgeDir := ""
	switch strings.ToLower(knowledgeSource) {
	case "crawlai":
		crawlPath = discoverCrawlWiki()
	case "folder":
		knowledgeDir = fmt.Sprintf("./projects/%s/knowledge", project)
	}

	configTemplate := fmt.Sprintf(`db_path: %s
ai_provider: ollama
ai_model: llama3
ai_url: %s

mailbox_provider: %s
imap_host: ""
imap_port: 993
imap_user: ""
imap_pass: ""
smtp_host: ""
smtp_port: 587
smtp_user: ""
smtp_pass: ""
smtp_from: ""

# Or use Gmail OAuth instead:
# mailbox_provider: gmail
# gmail_credentials: "C:/path/to/credentials.json"
# gmail_token: "C:/path/to/token.json"

knowledge_source: %s
# knowledge_source: crawlai
crawl_path: "%s"
# knowledge_source: folder
knowledge_dir: "%s"

sender_name: ""
sender_address: ""
sender_physical: ""
public_base_url: ""
unsub_secret: ""
`, dbPath, viper.GetString("ai_url"), mailboxProvider, knowledgeSource, crawlPath, knowledgeDir)
	if err := os.WriteFile(configPath, []byte(configTemplate), 0644); err != nil {
		return "", "", fmt.Errorf("write config: %w", err)
	}
	if verbose {
		fmt.Printf("✅ Project configuration saved: %s\n", configPath)
	}

	if knowledgeDir != "" {
		knowledgeReadmePath := filepath.Join(baseDir, "knowledge", "README.md")
		knowledgeReadme := `# Local Knowledge Folder

Drop Markdown or plain-text files here when ` + "`knowledge_source: folder`" + ` is selected.

Recommended contents:
- FAQ.md
- shipping.md
- billing.md
- refund-policy.md
`
		if err := writeIfMissing(knowledgeReadmePath, knowledgeReadme); err != nil {
			return "", "", fmt.Errorf("write knowledge readme: %w", err)
		}
		if verbose {
			fmt.Printf("✅ Knowledge folder guide ready: %s\n", knowledgeReadmePath)
		}
	}

	setupGuidePath := filepath.Join(baseDir, "SETUP.md")
	setupGuide := fmt.Sprintf("# Setup Checklist: %s\n\n1. Fill mailbox credentials in `config.yaml` for `%s`.\n2. Set `public_base_url` to the externally reachable URL of `musu-nurikun serve`.\n3. Set a strong `unsub_secret` for signed one-click unsubscribe links.\n4. If `knowledge_source: %s`, review the referenced knowledge path before running `watch` or `campaign`.\n5. Run `musu-nurikun doctor --project %s` until it passes.\n", project, mailboxProvider, knowledgeSource, project)
	if err := writeIfMissing(setupGuidePath, setupGuide); err != nil {
		return "", "", fmt.Errorf("write setup guide: %w", err)
	}
	if verbose {
		fmt.Printf("✅ Setup guide ready: %s\n", setupGuidePath)
	}
	return baseDir, configPath, nil
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize musu-nurikun environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		project := viper.GetString("project")
		mailboxProvider, _ := cmd.Flags().GetString("mailbox-provider")
		knowledgeSource, _ := cmd.Flags().GetString("knowledge-source")
		jsonMode := viper.GetBool("json")
		if !jsonMode {
			fmt.Printf("📬 Initializing musu-nurikun for project '%s' (Version %s)...\n", project, Version)
		}

		baseDir, configPath, err := bootstrapProject(project, !jsonMode, mailboxProvider, knowledgeSource)
		if err != nil {
			return err
		}

		dbPath := filepath.Join(baseDir, "data", "nurikun.db")
		result := map[string]interface{}{
			"project":           project,
			"project_dir":       baseDir,
			"config_path":       configPath,
			"db_path":           dbPath,
			"mailbox_provider":  mailboxProvider,
			"knowledge_source":  knowledgeSource,
			"ai_url":            viper.GetString("ai_url"),
			"setup_guide_path":  filepath.Join(baseDir, "SETUP.md"),
			"next_steps": []string{
				fmt.Sprintf("fill mailbox credentials and public delivery settings in %s", configPath),
				fmt.Sprintf("run 'musu-nurikun doctor --project %s'", project),
			},
		}
		if !jsonMode {
			fmt.Printf("\n✨ Initialization complete! Next: configure your mailbox (IMAP/SMTP or Gmail) and knowledge source in %s\n", configPath)
		}
		printJSONSuccess("Project initialized", result)
		return nil
	},
}

func init() {
	initCmd.Flags().String("mailbox-provider", "imap", "Mailbox provider preset for the generated config (imap or gmail)")
	initCmd.Flags().String("knowledge-source", "none", "Knowledge source preset for the generated config (none, crawlai, folder)")
	rootCmd.AddCommand(initCmd)
}
