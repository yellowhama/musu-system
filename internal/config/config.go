package config

import (
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/yellowhama/musu-nurikun/internal/policy"
)

// Config is the runtime configuration for a project's email agent. Secrets
// (passwords, OAuth tokens) come from env (NURIKUN_*) or the project
// config.yaml and are never committed to git.
type Config struct {
	Project    string
	AIModel    string
	AIBaseURL  string // OpenAI-compatible /v1 base URL (ai_url)
	AIProvider string

	MailboxProvider string // "imap" | "gmail"

	IMAPHost string
	IMAPPort int
	IMAPUser string
	IMAPPass string

	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string
	SMTPFrom string

	GmailCredentials string
	GmailToken       string

	KnowledgeSource string // "crawlai" | "folder" | "none"
	CrawlPath       string
	KnowledgeDir    string

	SenderName     string
	SenderAddress  string
	SenderPhysical string // postal address required in the compliance footer

	PublicBaseURL string // base URL the `serve` endpoints are reachable at (unsubscribe/confirm links)
	UnsubSecret   string // HMAC secret for signed one-click unsubscribe links

	Policy policy.Config
}

// Load reads configuration with precedence: env (NURIKUN_*) > project
// config.yaml > defaults. The project config file is optional.
func Load(project string) (*Config, error) {
	v := viper.New()
	v.SetDefault("ai_model", "llama3")
	v.SetDefault("ai_url", "http://localhost:11434/v1")
	v.SetDefault("ai_provider", "ollama")
	v.SetDefault("mailbox_provider", "imap")
	v.SetDefault("imap_port", 993)
	v.SetDefault("smtp_port", 587)
	v.SetDefault("knowledge_source", "none")

	v.SetEnvPrefix("NURIKUN")
	v.AutomaticEnv()

	cfgPath := filepath.Join("projects", project, "config.yaml")
	v.SetConfigFile(cfgPath)
	_ = v.ReadInConfig() // optional

	return &Config{
		Project:    project,
		AIModel:    v.GetString("ai_model"),
		AIBaseURL:  v.GetString("ai_url"),
		AIProvider: v.GetString("ai_provider"),

		MailboxProvider:  v.GetString("mailbox_provider"),
		IMAPHost:         v.GetString("imap_host"),
		IMAPPort:         v.GetInt("imap_port"),
		IMAPUser:         v.GetString("imap_user"),
		IMAPPass:         v.GetString("imap_pass"),
		SMTPHost:         v.GetString("smtp_host"),
		SMTPPort:         v.GetInt("smtp_port"),
		SMTPUser:         v.GetString("smtp_user"),
		SMTPPass:         v.GetString("smtp_pass"),
		SMTPFrom:         v.GetString("smtp_from"),
		GmailCredentials: v.GetString("gmail_credentials"),
		GmailToken:       v.GetString("gmail_token"),

		KnowledgeSource: v.GetString("knowledge_source"),
		CrawlPath:       v.GetString("crawl_path"),
		KnowledgeDir:    v.GetString("knowledge_dir"),

		SenderName:     v.GetString("sender_name"),
		SenderAddress:  v.GetString("sender_address"),
		SenderPhysical: v.GetString("sender_physical"),

		PublicBaseURL: v.GetString("public_base_url"),
		UnsubSecret:   v.GetString("unsub_secret"),

		Policy: policy.DefaultConfig(),
	}, nil
}
