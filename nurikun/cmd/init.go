package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-system/nurikun/internal/db"
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

func writeProjectArtifacts(project string, mailboxProvider string, knowledgeSource string) error {
	baseDir := filepath.Join("projects", project)
	envExamplePath := filepath.Join(baseDir, ".env.example")
	bootstrapScriptPath := filepath.Join(baseDir, "bootstrap.ps1")
	oauthDir := filepath.Join(baseDir, "oauth")

	if err := writeIfMissing(envExamplePath, buildEnvExample(project, mailboxProvider, knowledgeSource)); err != nil {
		return fmt.Errorf("write env example: %w", err)
	}
	if err := writeIfMissing(bootstrapScriptPath, buildBootstrapScript(project)); err != nil {
		return fmt.Errorf("write bootstrap script: %w", err)
	}

	if strings.EqualFold(mailboxProvider, "gmail") {
		if err := os.MkdirAll(oauthDir, 0755); err != nil {
			return fmt.Errorf("create oauth dir: %w", err)
		}
		oauthReadmePath := filepath.Join(oauthDir, "README.md")
		oauthGuide := "# Gmail OAuth Bootstrap\n\n1. Put your Google OAuth desktop credentials at `credentials.json`.\n2. Run the Gmail token bootstrap flow to generate `token.json`.\n3. Keep both files local and out of git.\n"
		if err := writeIfMissing(oauthReadmePath, oauthGuide); err != nil {
			return fmt.Errorf("write oauth guide: %w", err)
		}
	}

	return nil
}

func buildEnvExample(project string, mailboxProvider string, knowledgeSource string) string {
	lines := []string{
		"# Copy this file to .env and fill the values before running doctor/watch/campaign.",
		fmt.Sprintf("NURIKUN_AI_URL=http://127.0.0.1:11434/v1"),
		"NURIKUN_AI_MODEL=llama3",
		"",
		"NURIKUN_PUBLIC_BASE_URL=https://mail.example.com",
		"NURIKUN_UNSUB_SECRET=<generate-or-run-bootstrap-script>",
		"NURIKUN_SENDER_NAME=Example Support",
		"NURIKUN_SENDER_ADDRESS=support@example.com",
		"NURIKUN_SENDER_PHYSICAL=123 Example-ro, Seoul",
	}

	switch strings.ToLower(mailboxProvider) {
	case "gmail":
		lines = append(lines,
			"",
			"NURIKUN_MAILBOX_PROVIDER=gmail",
			fmt.Sprintf("NURIKUN_GMAIL_CREDENTIALS=./projects/%s/oauth/credentials.json", project),
			fmt.Sprintf("NURIKUN_GMAIL_TOKEN=./projects/%s/oauth/token.json", project),
			"NURIKUN_SMTP_FROM=support@example.com",
		)
	default:
		lines = append(lines,
			"",
			"NURIKUN_MAILBOX_PROVIDER=imap",
			"NURIKUN_IMAP_HOST=imap.example.com",
			"NURIKUN_IMAP_PORT=993",
			"NURIKUN_IMAP_USER=<imap-user>",
			"NURIKUN_IMAP_PASS=<imap-password>",
			"NURIKUN_SMTP_HOST=smtp.example.com",
			"NURIKUN_SMTP_PORT=587",
			"NURIKUN_SMTP_USER=<smtp-user>",
			"NURIKUN_SMTP_PASS=<smtp-password>",
			"NURIKUN_SMTP_FROM=support@example.com",
		)
	}

	switch strings.ToLower(knowledgeSource) {
	case "crawlai":
		lines = append(lines,
			"",
			"NURIKUN_KNOWLEDGE_SOURCE=crawlai",
			fmt.Sprintf("NURIKUN_CRAWL_PATH=%s", filepath.ToSlash(discoverCrawlWiki())),
		)
	case "folder":
		lines = append(lines,
			"",
			"NURIKUN_KNOWLEDGE_SOURCE=folder",
			fmt.Sprintf("NURIKUN_KNOWLEDGE_DIR=./projects/%s/knowledge", project),
		)
	default:
		lines = append(lines,
			"",
			"NURIKUN_KNOWLEDGE_SOURCE=none",
		)
	}

	return strings.Join(lines, "\n") + "\n"
}

func buildBootstrapScript(project string) string {
	return fmt.Sprintf(`$ErrorActionPreference = 'Stop'
$base = Split-Path -Parent $MyInvocation.MyCommand.Path
$envExample = Join-Path $base '.env.example'
$envPath = Join-Path $base '.env'

if (-not (Test-Path $envExample)) {
  throw ".env.example is missing at $envExample"
}

if (-not (Test-Path $envPath)) {
  Copy-Item $envExample $envPath
  Write-Host "Copied .env.example to .env"
}

$content = Get-Content $envPath -Raw
if ($content -match 'NURIKUN_UNSUB_SECRET=<generate-or-run-bootstrap-script>') {
  $bytes = New-Object byte[] 32
  [System.Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
  $secret = [Convert]::ToBase64String($bytes).TrimEnd('=').Replace('+','-').Replace('/','_')
  $content = $content -replace 'NURIKUN_UNSUB_SECRET=<generate-or-run-bootstrap-script>', ('NURIKUN_UNSUB_SECRET=' + $secret)
  Set-Content -Path $envPath -Value $content
  Write-Host "Generated NURIKUN_UNSUB_SECRET in .env"
}

Write-Host "Next:"
Write-Host "  1. Fill mailbox credentials in $envPath"
Write-Host "  2. Review projects/%s/config.yaml"
Write-Host "  3. Run: musu-nurikun doctor --project %s"
`, project, project)
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

	gmailCredentialsPath := ""
	gmailTokenPath := ""
	if strings.EqualFold(mailboxProvider, "gmail") {
		gmailCredentialsPath = fmt.Sprintf("./projects/%s/oauth/credentials.json", project)
		gmailTokenPath = fmt.Sprintf("./projects/%s/oauth/token.json", project)
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
gmail_credentials: "%s"
gmail_token: "%s"

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
`, dbPath, viper.GetString("ai_url"), mailboxProvider, gmailCredentialsPath, gmailTokenPath, knowledgeSource, crawlPath, knowledgeDir)
	if err := os.WriteFile(configPath, []byte(configTemplate), 0644); err != nil {
		return "", "", fmt.Errorf("write config: %w", err)
	}
	if verbose {
		fmt.Printf("✅ Project configuration saved: %s\n", configPath)
	}

	if err := writeProjectArtifacts(project, mailboxProvider, knowledgeSource); err != nil {
		return "", "", err
	}
	if verbose {
		fmt.Printf("✅ Bootstrap artifacts ready: ./%s/.env.example and bootstrap.ps1\n", baseDir)
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
	setupGuide := fmt.Sprintf("# Setup Checklist: %s\n\n1. Run `./projects/%s/bootstrap.ps1` to copy `.env.example` to `.env` and generate `NURIKUN_UNSUB_SECRET` / `unsub_secret`.\n2. Fill mailbox credentials in `.env` or `config.yaml` for `%s`.\n3. Set `public_base_url` to the externally reachable URL of `musu-nurikun serve`.\n4. If `knowledge_source: %s`, review the referenced knowledge path before running `watch` or `campaign`.\n5. Run `musu-nurikun doctor --project %s` until it passes.\n", project, project, mailboxProvider, knowledgeSource, project)
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
			"env_example_path":  filepath.Join(baseDir, ".env.example"),
			"bootstrap_script":  filepath.Join(baseDir, "bootstrap.ps1"),
			"mailbox_provider":  mailboxProvider,
			"knowledge_source":  knowledgeSource,
			"ai_url":            viper.GetString("ai_url"),
			"setup_guide_path":  filepath.Join(baseDir, "SETUP.md"),
			"next_steps": []string{
				fmt.Sprintf("run ./projects/%s/bootstrap.ps1", project),
				fmt.Sprintf("fill mailbox credentials and public delivery settings in %s or .env", configPath),
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
