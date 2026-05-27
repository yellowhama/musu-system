package cmd

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-nurikun/internal/config"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check project config, mailbox settings, knowledge source, and AI connectivity",
	RunE: func(cmd *cobra.Command, args []string) error {
		project := viper.GetString("project")
		conf, err := config.Load(project)
		if err != nil {
			return err
		}

		jsonMode := viper.GetBool("json")
		if !jsonMode {
			fmt.Println("==> musu-nurikun doctor")
			fmt.Printf("Project          : %s\n", project)
			fmt.Printf("AI URL           : %s\n", conf.AIBaseURL)
			fmt.Printf("Mailbox Provider : %s\n", conf.MailboxProvider)
			fmt.Printf("Knowledge Source : %s\n", conf.KnowledgeSource)
		}

		hasError := false
		report := map[string]interface{}{
			"project":            project,
			"ai_url":             conf.AIBaseURL,
			"mailbox_provider":   conf.MailboxProvider,
			"knowledge_source":   conf.KnowledgeSource,
			"config_exists":      false,
			"mailbox_ok":         false,
			"knowledge_ok":       false,
			"public_base_url_ok": false,
			"unsub_secret_ok":    false,
			"ai_reachable":       false,
		}
		autoFix, _ := cmd.Flags().GetBool("fix")
		fixMailboxProvider, _ := cmd.Flags().GetString("mailbox-provider")
		fixKnowledgeSource, _ := cmd.Flags().GetString("knowledge-source")

		configPath := filepath.Join("projects", project, "config.yaml")
		if _, err := os.Stat(configPath); err != nil {
			if autoFix {
				if !jsonMode {
					fmt.Printf("🛠️  Auto-fixing missing project scaffold for %s\n", project)
				}
				if _, _, fixErr := bootstrapProject(project, !jsonMode, fixMailboxProvider, fixKnowledgeSource); fixErr == nil {
					report["config_exists"] = true
					if refreshedConf, loadErr := config.Load(project); loadErr == nil {
						conf = refreshedConf
						report["ai_url"] = conf.AIBaseURL
						report["mailbox_provider"] = conf.MailboxProvider
						report["knowledge_source"] = conf.KnowledgeSource
					}
				} else {
					report["config_fix_error"] = fixErr.Error()
					if !jsonMode {
						fmt.Printf("❌ Auto-fix failed: %v\n", fixErr)
					}
				}
			} else if !jsonMode {
				fmt.Printf("⚠️  Project config missing: %s\n", configPath)
				fmt.Printf("   Run: musu-nurikun init --project %s or use doctor --fix\n", project)
			}
		} else {
			if !jsonMode {
				fmt.Println("✅ Project config exists")
			}
			report["config_exists"] = true
		}

		if issues := mailboxIssues(conf); len(issues) > 0 {
			if !jsonMode {
				fmt.Println("❌ Mailbox configuration issues:")
				for _, issue := range issues {
					fmt.Printf("   - %s\n", issue)
				}
			}
			report["mailbox_issues"] = issues
			hasError = true
		} else {
			if !jsonMode {
				fmt.Println("✅ Mailbox configuration looks complete")
			}
			report["mailbox_ok"] = true
		}

		if issues := knowledgeIssues(conf); len(issues) > 0 {
			if !jsonMode {
				fmt.Println("❌ Knowledge source issues:")
				for _, issue := range issues {
					fmt.Printf("   - %s\n", issue)
				}
			}
			report["knowledge_issues"] = issues
			hasError = true
		} else {
			if !jsonMode {
				fmt.Println("✅ Knowledge source configuration looks complete")
			}
			report["knowledge_ok"] = true
		}

		if strings.TrimSpace(conf.PublicBaseURL) == "" {
			if !jsonMode {
				fmt.Println("⚠️  public_base_url is empty; campaign unsubscribe/confirm links will degrade.")
			}
		} else {
			if !jsonMode {
				fmt.Println("✅ public_base_url configured")
			}
			report["public_base_url_ok"] = true
		}
		if strings.TrimSpace(conf.UnsubSecret) == "" {
			if !jsonMode {
				fmt.Println("⚠️  unsub_secret is empty; one-click signed links are not protected.")
			}
		} else {
			if !jsonMode {
				fmt.Println("✅ unsub_secret configured")
			}
			report["unsub_secret_ok"] = true
		}

		if err := probeModels(conf.AIBaseURL); err != nil {
			if !jsonMode {
				fmt.Printf("❌ AI endpoint probe failed: %v\n", err)
			}
			report["ai_error"] = err.Error()
			hasError = true
		} else {
			if !jsonMode {
				fmt.Println("✅ AI endpoint reachable")
			}
			report["ai_reachable"] = true
		}

		if hasError {
			err := fmt.Errorf("doctor found blocking issues")
			printJSONError(err, report)
			return err
		}
		if !jsonMode {
			fmt.Println("✅ Doctor passed")
		}
		printJSONSuccess("Doctor passed", report)
		return nil
	},
}

func init() {
	doctorCmd.Flags().Bool("fix", false, "Auto-create missing local project scaffold when safe")
	doctorCmd.Flags().String("mailbox-provider", "imap", "Mailbox provider preset to use with doctor --fix (imap or gmail)")
	doctorCmd.Flags().String("knowledge-source", "none", "Knowledge source preset to use with doctor --fix (none, crawlai, folder)")
	rootCmd.AddCommand(doctorCmd)
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
