package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

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

	projectEnv := loadProjectEnv(filepath.Join("projects", project, ".env"))

	return &Config{
		Project:    project,
		AIModel:    valueString(v, projectEnv, "ai_model", "NURIKUN_AI_MODEL"),
		AIBaseURL:  valueString(v, projectEnv, "ai_url", "NURIKUN_AI_URL"),
		AIProvider: valueString(v, projectEnv, "ai_provider", "NURIKUN_AI_PROVIDER"),

		MailboxProvider:  valueString(v, projectEnv, "mailbox_provider", "NURIKUN_MAILBOX_PROVIDER"),
		IMAPHost:         valueString(v, projectEnv, "imap_host", "NURIKUN_IMAP_HOST"),
		IMAPPort:         valueInt(v, projectEnv, "imap_port", "NURIKUN_IMAP_PORT"),
		IMAPUser:         valueString(v, projectEnv, "imap_user", "NURIKUN_IMAP_USER"),
		IMAPPass:         valueString(v, projectEnv, "imap_pass", "NURIKUN_IMAP_PASS"),
		SMTPHost:         valueString(v, projectEnv, "smtp_host", "NURIKUN_SMTP_HOST"),
		SMTPPort:         valueInt(v, projectEnv, "smtp_port", "NURIKUN_SMTP_PORT"),
		SMTPUser:         valueString(v, projectEnv, "smtp_user", "NURIKUN_SMTP_USER"),
		SMTPPass:         valueString(v, projectEnv, "smtp_pass", "NURIKUN_SMTP_PASS"),
		SMTPFrom:         valueString(v, projectEnv, "smtp_from", "NURIKUN_SMTP_FROM"),
		GmailCredentials: valueString(v, projectEnv, "gmail_credentials", "NURIKUN_GMAIL_CREDENTIALS"),
		GmailToken:       valueString(v, projectEnv, "gmail_token", "NURIKUN_GMAIL_TOKEN"),

		KnowledgeSource: valueString(v, projectEnv, "knowledge_source", "NURIKUN_KNOWLEDGE_SOURCE"),
		CrawlPath:       valueString(v, projectEnv, "crawl_path", "NURIKUN_CRAWL_PATH"),
		KnowledgeDir:    valueString(v, projectEnv, "knowledge_dir", "NURIKUN_KNOWLEDGE_DIR"),

		SenderName:     valueString(v, projectEnv, "sender_name", "NURIKUN_SENDER_NAME"),
		SenderAddress:  valueString(v, projectEnv, "sender_address", "NURIKUN_SENDER_ADDRESS"),
		SenderPhysical: valueString(v, projectEnv, "sender_physical", "NURIKUN_SENDER_PHYSICAL"),

		PublicBaseURL: valueString(v, projectEnv, "public_base_url", "NURIKUN_PUBLIC_BASE_URL"),
		UnsubSecret:   valueString(v, projectEnv, "unsub_secret", "NURIKUN_UNSUB_SECRET"),

		Policy: policy.DefaultConfig(),
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func valueString(v *viper.Viper, projectEnv map[string]string, key string, envKey string) string {
	if value, ok := os.LookupEnv(envKey); ok && strings.TrimSpace(value) != "" {
		return value
	}
	if value, ok := projectEnv[envKey]; ok && strings.TrimSpace(value) != "" {
		return value
	}
	return v.GetString(key)
}

func valueInt(v *viper.Viper, projectEnv map[string]string, key string, envKey string) int {
	if value, ok := os.LookupEnv(envKey); ok && strings.TrimSpace(value) != "" {
		if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
			return parsed
		}
	}
	if value, ok := projectEnv[envKey]; ok && strings.TrimSpace(value) != "" {
		if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
			return parsed
		}
	}
	return v.GetInt(key)
}

func loadProjectEnv(path string) map[string]string {
	values := map[string]string{}
	data, err := os.ReadFile(path)
	if err != nil {
		return values
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"'`)
		if key != "" {
			values[key] = value
		}
	}
	return values
}
