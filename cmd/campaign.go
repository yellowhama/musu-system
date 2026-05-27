package cmd

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-nurikun/internal/compliance"
	"github.com/yellowhama/musu-nurikun/internal/config"
	"github.com/yellowhama/musu-nurikun/internal/db"
	"github.com/yellowhama/musu-nurikun/internal/mailbox"
)

var (
	campaignList    int
	campaignName    string
	campaignSubject string
	campaignBody    string
	campaignPersona string
	campaignDryRun  bool
)

// Conservative defaults so a campaign cannot flood a single inbox provider or
// the whole run. These protect domain reputation and are not user-tunable here.
const (
	campaignMinInterval = 2 * time.Second
	campaignPerRunCap   = 500
)

var campaignCmd = &cobra.Command{
	Use:   "campaign",
	Short: "Send a compliant campaign to confirmed, due subscribers of a list",
	Long: `Send an opt-in campaign.

The send pipeline is gated, in order, by:
  1. confirmed + due subscribers only (DueSubscribers: confirmed, past cadence),
  2. the suppression list (IsSuppressed — hard opt-out gate),
  3. a per-domain rate limiter + per-run cap,
  4. mandatory compliance decoration: "(광고)" subject label, sender/postal
     footer, and a List-Unsubscribe header.

None of these guards can be disabled by a flag. --dry-run only suppresses the
actual SMTP/Gmail delivery; every other guard and transform still runs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if campaignList == 0 || campaignName == "" || campaignSubject == "" || campaignBody == "" {
			return fmt.Errorf("--list, --name, --subject and --body are required")
		}

		project := viper.GetString("project")
		cfg, err := config.Load(project)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		store, err := openStore()
		if err != nil {
			return err
		}
		defer store.Close()

		// 1. Persist the campaign as a draft.
		campaignID, err := store.CreateCampaign(campaignList, campaignName, campaignPersona, campaignSubject, campaignBody)
		if err != nil {
			return fmt.Errorf("create campaign: %w", err)
		}

		// 2. Resolve cadence and the due audience (confirmed, not suppressed, past cadence).
		list, err := store.GetList(campaignList)
		if err != nil {
			return fmt.Errorf("get list %d: %w", campaignList, err)
		}
		subs, err := store.DueSubscribers(list.ID, list.CadenceDays)
		if err != nil {
			return fmt.Errorf("due subscribers: %w", err)
		}

		// 3. Build the mailbox (skipped on dry-run so it can run without creds).
		var mb mailbox.Mailbox
		if !campaignDryRun {
			mb, err = mailbox.New(cfg.MailboxProvider, mailboxSettings(cfg))
			if err != nil {
				return fmt.Errorf("open mailbox: %w", err)
			}
			defer mb.Close()
		}

		limiter := compliance.NewLimiter(campaignMinInterval, campaignPerRunCap)

		fmt.Printf("📣 Campaign #%d %q → list #%d %q: %d due subscriber(s)%s\n",
			campaignID, campaignName, list.ID, list.Name, len(subs), dryRunSuffix(campaignDryRun))

		sent, skippedSuppressed, skippedLimited, failed := 0, 0, 0, 0
		for _, sub := range subs {
			// Hard opt-out gate — never bypassable.
			suppressed, err := store.IsSuppressed(sub.Email)
			if err != nil {
				return fmt.Errorf("check suppression for %s: %w", sub.Email, err)
			}
			if suppressed {
				skippedSuppressed++
				continue
			}

			// Rate / reputation gate.
			if !limiter.Allow(sub.Email) {
				skippedLimited++
				continue
			}

			out := mailbox.OutMessage{
				To:      sub.Email,
				Subject: campaignSubject,
				Body:    campaignBody,
			}
			// Mandatory compliance: ad-label + footer + List-Unsubscribe header.
			compliance.Decorate(&out, cfg.SenderName, cfg.SenderPhysical, unsubscribeURLFor(cfg, sub))

			if campaignDryRun {
				fmt.Printf("  [dry-run] would send to %s | subj=%q\n", out.To, out.Subject)
				sent++
				continue
			}

			if err := mb.Send(out); err != nil {
				fmt.Printf("  ❌ send to %s failed: %v\n", sub.Email, err)
				failed++
				continue
			}
			if err := store.MarkSent(sub.ID); err != nil {
				fmt.Printf("  ⚠️  sent to %s but failed to record cadence: %v\n", sub.Email, err)
			}
			sent++
		}

		// 4. Finalize campaign status with the delivered count.
		status := "sent"
		if campaignDryRun {
			status = "draft"
		}
		if err := store.UpdateCampaignStatus(int(campaignID), status, sent); err != nil {
			return fmt.Errorf("update campaign status: %w", err)
		}

		fmt.Printf("\nSummary: %d %s, %d suppressed-skip, %d rate-limited-skip, %d failed (campaign #%d → %s)\n",
			sent, sentVerb(campaignDryRun), skippedSuppressed, skippedLimited, failed, campaignID, status)
		return nil
	},
}

// mailboxSettings maps config to the mailbox factory's Settings.
func mailboxSettings(cfg *config.Config) mailbox.Settings {
	return mailbox.Settings{
		IMAPHost:         cfg.IMAPHost,
		IMAPPort:         cfg.IMAPPort,
		IMAPUser:         cfg.IMAPUser,
		IMAPPass:         cfg.IMAPPass,
		SMTPHost:         cfg.SMTPHost,
		SMTPPort:         cfg.SMTPPort,
		SMTPUser:         cfg.SMTPUser,
		SMTPPass:         cfg.SMTPPass,
		From:             cfg.SMTPFrom,
		GmailCredentials: cfg.GmailCredentials,
		GmailToken:       cfg.GmailToken,
	}
}

// unsubscribeURLFor produces a per-subscriber one-click opt-out target. With no
// configured web unsubscribe endpoint, we fall back to a mailto: to the sender
// address carrying an unsubscribe subject scoped to the recipient — always a
// valid, honored opt-out path. Prefer the SenderAddress, then SMTPFrom.
func unsubscribeURLFor(cfg *config.Config, sub db.Subscriber) string {
	// Prefer a signed one-click web unsubscribe when a public endpoint + secret
	// are configured (served by `musu-nurikun serve`).
	if cfg.PublicBaseURL != "" && cfg.UnsubSecret != "" {
		return fmt.Sprintf("%s/unsubscribe?email=%s&sig=%s",
			strings.TrimRight(cfg.PublicBaseURL, "/"),
			url.QueryEscape(sub.Email),
			compliance.SignUnsub(sub.Email, cfg.UnsubSecret))
	}
	addr := cfg.SenderAddress
	if addr == "" {
		addr = cfg.SMTPFrom
	}
	if addr == "" {
		return ""
	}
	subject := fmt.Sprintf("unsubscribe %s", sub.Email)
	return fmt.Sprintf("mailto:%s?subject=%s", addr, url.QueryEscape(subject))
}

func dryRunSuffix(dry bool) string {
	if dry {
		return " [DRY RUN]"
	}
	return ""
}

func sentVerb(dry bool) string {
	if dry {
		return "would-send"
	}
	return "sent"
}

func init() {
	campaignCmd.Flags().IntVar(&campaignList, "list", 0, "List ID to send the campaign to (required)")
	campaignCmd.Flags().StringVar(&campaignName, "name", "", "Campaign name (required)")
	campaignCmd.Flags().StringVar(&campaignSubject, "subject", "", "Email subject (required; auto-labeled with (광고))")
	campaignCmd.Flags().StringVar(&campaignBody, "body", "", "Email body (required; compliance footer appended)")
	campaignCmd.Flags().StringVar(&campaignPersona, "persona", "", "Optional persona tag stored with the campaign")
	campaignCmd.Flags().BoolVar(&campaignDryRun, "dry-run", false, "Preview recipients without sending (compliance still applied)")
	rootCmd.AddCommand(campaignCmd)
}
