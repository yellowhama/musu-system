package preflight

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/yellowhama/musu-nurikun/internal/config"
)

type DoctorOptions struct {
	Project    string
	ConfigPath string
	Config     *config.Config
	AutoFix    bool
	FixProject func() (*config.Config, error)
}

type DoctorReport struct {
	Project         string   `json:"project"`
	AIURL           string   `json:"ai_url"`
	MailboxProvider string   `json:"mailbox_provider"`
	KnowledgeSource string   `json:"knowledge_source"`
	ConfigExists    bool     `json:"config_exists"`
	MailboxOK       bool     `json:"mailbox_ok"`
	KnowledgeOK     bool     `json:"knowledge_ok"`
	PublicBaseURLOK bool     `json:"public_base_url_ok"`
	UnsubSecretOK   bool     `json:"unsub_secret_ok"`
	AIReachable     bool     `json:"ai_reachable"`

	MailboxIssues   []string `json:"mailbox_issues,omitempty"`
	KnowledgeIssues []string `json:"knowledge_issues,omitempty"`
	AIError         string   `json:"ai_error,omitempty"`
	ConfigFixError  string   `json:"config_fix_error,omitempty"`
}

type DoctorResult struct {
	Report        DoctorReport
	Blocking      bool
	ActionableFix string
}

func EvaluateDoctor(opts DoctorOptions) DoctorResult {
	conf := opts.Config
	report := DoctorReport{
		Project:         opts.Project,
		AIURL:           conf.AIBaseURL,
		MailboxProvider: conf.MailboxProvider,
		KnowledgeSource: conf.KnowledgeSource,
		ConfigExists:    false,
		MailboxOK:       false,
		KnowledgeOK:     false,
		PublicBaseURLOK: false,
		UnsubSecretOK:   false,
		AIReachable:     false,
	}
	result := DoctorResult{
		Report:        report,
		ActionableFix: "Run init or doctor --fix to recreate the scaffold, then fill mailbox credentials, public_base_url, unsub_secret, and start the configured AI endpoint.",
	}

	if _, err := os.Stat(opts.ConfigPath); err != nil {
		if opts.AutoFix && opts.FixProject != nil {
			if refreshedConf, fixErr := opts.FixProject(); fixErr == nil {
				conf = refreshedConf
				result.Report.ConfigExists = true
				result.Report.AIURL = conf.AIBaseURL
				result.Report.MailboxProvider = conf.MailboxProvider
				result.Report.KnowledgeSource = conf.KnowledgeSource
			} else {
				result.Report.ConfigFixError = fixErr.Error()
				result.Blocking = true
			}
		} else {
			result.Blocking = true
		}
	} else {
		result.Report.ConfigExists = true
	}

	if issues := mailboxIssues(conf); len(issues) > 0 {
		result.Report.MailboxIssues = issues
		result.Blocking = true
	} else {
		result.Report.MailboxOK = true
	}

	if issues := knowledgeIssues(conf); len(issues) > 0 {
		result.Report.KnowledgeIssues = issues
		result.Blocking = true
	} else {
		result.Report.KnowledgeOK = true
	}

	if strings.TrimSpace(conf.PublicBaseURL) != "" {
		result.Report.PublicBaseURLOK = true
	} else {
		result.Blocking = true
	}
	if strings.TrimSpace(conf.UnsubSecret) != "" {
		result.Report.UnsubSecretOK = true
	} else {
		result.Blocking = true
	}

	if err := probeModels(conf.AIBaseURL); err != nil {
		result.Report.AIError = err.Error()
		result.Blocking = true
	} else {
		result.Report.AIReachable = true
	}

	result.ActionableFix = buildActionableFix(result.Report)

	return result
}

func buildActionableFix(report DoctorReport) string {
	var fixes []string
	if !report.ConfigExists {
		fixes = append(fixes, "run init or doctor --fix to recreate the project scaffold")
	}
	if len(report.MailboxIssues) > 0 {
		fixes = append(fixes, "fill the required mailbox credentials for the selected mailbox_provider")
	}
	if len(report.KnowledgeIssues) > 0 {
		fixes = append(fixes, "fix knowledge_source configuration so the referenced crawl or folder path exists")
	}
	if !report.PublicBaseURLOK {
		fixes = append(fixes, "set public_base_url so confirmation and unsubscribe links can be generated")
	}
	if !report.UnsubSecretOK {
		fixes = append(fixes, "set unsub_secret for signed one-click unsubscribe links")
	}
	if !report.AIReachable {
		fixes = append(fixes, "start the configured AI endpoint or pass a reachable --ai-url")
	}
	if len(fixes) == 0 {
		return "No action required."
	}
	return strings.Join(fixes, "; ") + "."
}

func mailboxIssues(conf *config.Config) []string {
	var issues []string
	switch strings.ToLower(strings.TrimSpace(conf.MailboxProvider)) {
	case "imap":
		if strings.TrimSpace(conf.IMAPHost) == "" {
			issues = append(issues, "imap_host is required for IMAP mode")
		}
		if strings.TrimSpace(conf.IMAPUser) == "" {
			issues = append(issues, "imap_user is required for IMAP mode")
		}
		if strings.TrimSpace(conf.IMAPPass) == "" {
			issues = append(issues, "imap_pass is required for IMAP mode")
		}
		if strings.TrimSpace(conf.SMTPHost) == "" {
			issues = append(issues, "smtp_host is required for IMAP mode")
		}
		if strings.TrimSpace(conf.SMTPFrom) == "" {
			issues = append(issues, "smtp_from is required for IMAP mode")
		}
	case "gmail":
		if strings.TrimSpace(conf.GmailCredentials) == "" {
			issues = append(issues, "gmail_credentials is required for Gmail mode")
		}
		if strings.TrimSpace(conf.GmailToken) == "" {
			issues = append(issues, "gmail_token is required for Gmail mode")
		}
	default:
		issues = append(issues, "mailbox_provider must be 'imap' or 'gmail'")
	}
	return issues
}

func knowledgeIssues(conf *config.Config) []string {
	var issues []string
	switch strings.ToLower(strings.TrimSpace(conf.KnowledgeSource)) {
	case "", "none":
		return issues
	case "crawlai":
		if strings.TrimSpace(conf.CrawlPath) == "" {
			issues = append(issues, "crawl_path is required when knowledge_source=crawlai")
			return issues
		}
		if info, err := os.Stat(conf.CrawlPath); err != nil || !info.IsDir() {
			issues = append(issues, fmt.Sprintf("crawl_path does not exist: %s", conf.CrawlPath))
		}
	case "folder":
		if strings.TrimSpace(conf.KnowledgeDir) == "" {
			issues = append(issues, "knowledge_dir is required when knowledge_source=folder")
			return issues
		}
		if info, err := os.Stat(conf.KnowledgeDir); err != nil || !info.IsDir() {
			issues = append(issues, fmt.Sprintf("knowledge_dir does not exist: %s", conf.KnowledgeDir))
		}
	default:
		issues = append(issues, "knowledge_source must be one of: none, crawlai, folder")
	}
	return issues
}

func probeModels(baseURL string) error {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return fmt.Errorf("empty ai-url")
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(baseURL + "/models")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return fmt.Errorf("unexpected status %s from %s/models", resp.Status, baseURL)
}
