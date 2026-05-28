package config

import (
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/yellowhama/musu-system/core/env"
	"github.com/yellowhama/musu-system/nurikun/internal/policy"
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
// `.env` file > project `config.yaml` > defaults. Layer merging is delegated
// to the shared `musu-core/env` package so the precedence is identical across
// every CLI in the ecosystem.
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

	projectEnv := env.LoadProjectEnv(filepath.Join("projects", project, ".env"))

	return &Config{
		Project:    project,
		AIModel:    env.String(v, projectEnv, "ai_model", "NURIKUN_AI_MODEL"),
		AIBaseURL:  env.String(v, projectEnv, "ai_url", "NURIKUN_AI_URL"),
		AIProvider: env.String(v, projectEnv, "ai_provider", "NURIKUN_AI_PROVIDER"),

		MailboxProvider:  env.String(v, projectEnv, "mailbox_provider", "NURIKUN_MAILBOX_PROVIDER"),
		IMAPHost:         env.String(v, projectEnv, "imap_host", "NURIKUN_IMAP_HOST"),
		IMAPPort:         env.Int(v, projectEnv, "imap_port", "NURIKUN_IMAP_PORT"),
		IMAPUser:         env.String(v, projectEnv, "imap_user", "NURIKUN_IMAP_USER"),
		IMAPPass:         env.String(v, projectEnv, "imap_pass", "NURIKUN_IMAP_PASS"),
		SMTPHost:         env.String(v, projectEnv, "smtp_host", "NURIKUN_SMTP_HOST"),
		SMTPPort:         env.Int(v, projectEnv, "smtp_port", "NURIKUN_SMTP_PORT"),
		SMTPUser:         env.String(v, projectEnv, "smtp_user", "NURIKUN_SMTP_USER"),
		SMTPPass:         env.String(v, projectEnv, "smtp_pass", "NURIKUN_SMTP_PASS"),
		SMTPFrom:         env.String(v, projectEnv, "smtp_from", "NURIKUN_SMTP_FROM"),
		GmailCredentials: env.String(v, projectEnv, "gmail_credentials", "NURIKUN_GMAIL_CREDENTIALS"),
		GmailToken:       env.String(v, projectEnv, "gmail_token", "NURIKUN_GMAIL_TOKEN"),

		KnowledgeSource: env.String(v, projectEnv, "knowledge_source", "NURIKUN_KNOWLEDGE_SOURCE"),
		CrawlPath:       env.String(v, projectEnv, "crawl_path", "NURIKUN_CRAWL_PATH"),
		KnowledgeDir:    env.String(v, projectEnv, "knowledge_dir", "NURIKUN_KNOWLEDGE_DIR"),

		SenderName:     env.String(v, projectEnv, "sender_name", "NURIKUN_SENDER_NAME"),
		SenderAddress:  env.String(v, projectEnv, "sender_address", "NURIKUN_SENDER_ADDRESS"),
		SenderPhysical: env.String(v, projectEnv, "sender_physical", "NURIKUN_SENDER_PHYSICAL"),

		PublicBaseURL: env.String(v, projectEnv, "public_base_url", "NURIKUN_PUBLIC_BASE_URL"),
		UnsubSecret:   env.String(v, projectEnv, "unsub_secret", "NURIKUN_UNSUB_SECRET"),

		Policy: policy.DefaultConfig(),
	}, nil
}
