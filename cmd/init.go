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
	return baseDir, configPath, nil
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize musu-nurikun environment",
	Run: func(cmd *cobra.Command, args []string) {
		project := viper.GetString("project")
		mailboxProvider, _ := cmd.Flags().GetString("mailbox-provider")
		knowledgeSource, _ := cmd.Flags().GetString("knowledge-source")
		fmt.Printf("📬 Initializing musu-nurikun for project '%s' (Version %s)...\n", project, Version)

		if _, configPath, err := bootstrapProject(project, true, mailboxProvider, knowledgeSource); err != nil {
			fmt.Printf("❌ Failed to initialize project: %v\n", err)
			return
		} else {
			fmt.Printf("\n✨ Initialization complete! Next: configure your mailbox (IMAP/SMTP or Gmail) and knowledge source in %s\n", configPath)
		}
	},
}

func init() {
	initCmd.Flags().String("mailbox-provider", "imap", "Mailbox provider preset for the generated config (imap or gmail)")
	initCmd.Flags().String("knowledge-source", "none", "Knowledge source preset for the generated config (none, crawlai, folder)")
	rootCmd.AddCommand(initCmd)
}
