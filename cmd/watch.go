package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-nurikun/internal/agent"
	"github.com/yellowhama/musu-nurikun/internal/config"
	"github.com/yellowhama/musu-nurikun/internal/db"
	"github.com/yellowhama/musu-nurikun/internal/knowledge"
	"github.com/yellowhama/musu-nurikun/internal/mailbox"
	"github.com/yellowhama/musu-nurikun/internal/triage"
)

var watchLimit int

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Fetch inbound support email, triage it, draft grounded replies, and auto-send or escalate per policy",
	Run: func(cmd *cobra.Command, args []string) {
		project := viper.GetString("project")

		cfg, err := config.Load(project)
		if err != nil {
			fmt.Printf("❌ Failed to load config for project %q: %v\n", project, err)
			return
		}

		dbPath := filepath.Join("projects", project, "data", "nurikun.db")
		store, err := db.NewStore(dbPath)
		if err != nil {
			fmt.Printf("❌ Failed to open database %s: %v\n", dbPath, err)
			return
		}
		defer store.Close()

		mb, err := mailbox.New(cfg.MailboxProvider, mailbox.Settings{
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
		})
		if err != nil {
			fmt.Printf("❌ Failed to open mailbox (%s): %v\n", cfg.MailboxProvider, err)
			return
		}
		defer mb.Close()

		ks, err := knowledge.New(knowledge.Settings{
			Kind:      cfg.KnowledgeSource,
			CrawlPath: cfg.CrawlPath,
			Project:   cfg.Project,
			Dir:       cfg.KnowledgeDir,
			AIBaseURL: cfg.AIBaseURL,
			AIModel:   cfg.AIModel,
		})
		if err != nil {
			fmt.Printf("❌ Failed to build knowledge source (%s): %v\n", cfg.KnowledgeSource, err)
			return
		}

		client := agent.NewAgentClient(cfg.AIBaseURL, cfg.AIModel, cfg.KnowledgeDir, cfg.Project)

		msgs, err := mb.Fetch(watchLimit)
		if err != nil {
			fmt.Printf("❌ Failed to fetch inbound messages: %v\n", err)
			return
		}

		if len(msgs) == 0 {
			fmt.Println("📭 No new inbound messages.")
			return
		}

		fmt.Printf("📬 Processing %d inbound message(s) for project %q...\n\n", len(msgs), project)

		for _, msg := range msgs {
			processMessage(store, mb, ks, client, cfg, msg)
		}
	},
}

// processMessage runs the full inbound pipeline for a single message. All
// errors are reported and the message is skipped so the loop continues.
func processMessage(
	store *db.Store,
	mb mailbox.Mailbox,
	ks knowledge.Source,
	client *agent.AgentClient,
	cfg *config.Config,
	msg mailbox.Message,
) {
	to := ""
	if len(msg.To) > 0 {
		to = msg.To[0]
	}

	// 1. Persist the inbound message.
	inID, err := store.SaveMessage(db.Message{
		ThreadID:  msg.ThreadID,
		Direction: "in",
		FromAddr:  msg.From,
		ToAddr:    to,
		Subject:   msg.Subject,
		Body:      msg.Body,
		Status:    "received",
		MessageID: msg.MessageID,
	})
	if err != nil {
		fmt.Printf("  ⚠️  [%s] failed to save inbound message: %v\n", msg.From, err)
		return
	}

	// 2. Triage.
	tr, err := triage.Classify(client, msg.Subject, msg.Body)
	if err != nil {
		fmt.Printf("  ⚠️  [#%d %s] triage failed: %v\n", inID, msg.From, err)
		return
	}

	// 3. Retrieve grounding snippets (prefer the intent summary, fall back to subject).
	query := tr.Intent
	if query == "" {
		query = msg.Subject
	}
	snippets, err := ks.Retrieve(query, 5)
	if err != nil {
		// Non-fatal: continue ungrounded.
		fmt.Printf("  ⚠️  [#%d %s] knowledge retrieval failed (continuing ungrounded): %v\n", inID, msg.From, err)
		snippets = nil
	}

	// 4. Draft a grounded reply.
	draft, err := agent.Respond(client, msg.Body, snippets, cfg.SenderName)
	if err != nil {
		fmt.Printf("  ⚠️  [#%d %s] reply drafting failed: %v\n", inID, msg.From, err)
		return
	}

	// 5. Apply policy.
	d := cfg.Policy.Decide(tr)

	replySubject := msg.Subject
	if replySubject != "" {
		replySubject = "Re: " + msg.Subject
	}

	out := db.Message{
		ThreadID:   msg.ThreadID,
		Direction:  "out",
		FromAddr:   cfg.SenderAddress,
		ToAddr:     msg.From,
		Subject:    replySubject,
		Body:       draft,
		Category:   string(tr.Category),
		Confidence: tr.Confidence,
		MessageID:  msg.MessageID, // thread anchor (InReplyTo target)
	}

	action := "drafted"
	switch {
	case d.AutoSend:
		out.Status = "sent"
		if _, err := store.SaveMessage(out); err != nil {
			fmt.Printf("  ⚠️  [#%d %s] failed to save outbound: %v\n", inID, msg.From, err)
			return
		}
		if err := mb.Send(mailbox.OutMessage{
			To:        msg.From,
			Subject:   replySubject,
			Body:      draft,
			InReplyTo: msg.MessageID,
		}); err != nil {
			fmt.Printf("  ⚠️  [#%d %s] auto-send failed (left as 'sent' record): %v\n", inID, msg.From, err)
			action = "send-failed"
			break
		}
		if err := mb.Mark(msg.UID); err != nil {
			fmt.Printf("  ⚠️  [#%d %s] failed to mark processed: %v\n", inID, msg.From, err)
		}
		action = "auto-sent"

	case d.Escalate:
		out.Status = "escalated"
		if _, err := store.SaveMessage(out); err != nil {
			fmt.Printf("  ⚠️  [#%d %s] failed to save escalated draft: %v\n", inID, msg.From, err)
			return
		}
		action = "escalated"

	default:
		out.Status = "drafted"
		if _, err := store.SaveMessage(out); err != nil {
			fmt.Printf("  ⚠️  [#%d %s] failed to save draft: %v\n", inID, msg.From, err)
			return
		}
		action = "drafted"
	}

	fmt.Printf("  ✉️  [#%d] from=%s | category=%s conf=%.2f lang=%s | %s — %s\n",
		inID, msg.From, tr.Category, tr.Confidence, tr.Language, action, d.Reason)
}

func init() {
	watchCmd.Flags().IntVar(&watchLimit, "limit", 10, "Maximum number of inbound messages to process")
	rootCmd.AddCommand(watchCmd)
}
