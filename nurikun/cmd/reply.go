package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-nurikun/internal/config"
	"github.com/yellowhama/musu-nurikun/internal/db"
	"github.com/yellowhama/musu-nurikun/internal/mailbox"
)

var (
	replyID   int
	replySend bool
)

var replyCmd = &cobra.Command{
	Use:   "reply",
	Short: "Print a stored drafted/escalated reply, or send it with --send",
	Run: func(cmd *cobra.Command, args []string) {
		if replyID <= 0 {
			fmt.Println("❌ --id is required (the stored message id)")
			return
		}

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

		msg, err := findOutboundMessage(store, replyID)
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			return
		}

		fmt.Printf("✉️  Message #%d [status=%s category=%s conf=%.2f]\n", msg.ID, msg.Status, msg.Category, msg.Confidence)
		fmt.Printf("To:      %s\n", msg.ToAddr)
		fmt.Printf("Subject: %s\n", msg.Subject)
		fmt.Printf("--- DRAFT ---\n%s\n--- END DRAFT ---\n", msg.Body)

		if !replySend {
			fmt.Println("\n(dry run — pass --send to deliver this reply)")
			return
		}

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

		if err := mb.Send(mailbox.OutMessage{
			To:        msg.ToAddr,
			Subject:   msg.Subject,
			Body:      msg.Body,
			InReplyTo: msg.MessageID,
		}); err != nil {
			fmt.Printf("❌ Failed to send message #%d: %v\n", msg.ID, err)
			return
		}

		if err := store.UpdateMessageStatus(msg.ID, "sent"); err != nil {
			fmt.Printf("⚠️  Sent, but failed to update status for #%d: %v\n", msg.ID, err)
			return
		}

		fmt.Printf("✅ Sent message #%d to %s\n", msg.ID, msg.ToAddr)
	},
}

// findOutboundMessage looks up a drafted or escalated outbound message by id.
func findOutboundMessage(store *db.Store, id int) (db.Message, error) {
	for _, status := range []string{"drafted", "escalated"} {
		msgs, err := store.MessagesByStatus(status)
		if err != nil {
			return db.Message{}, fmt.Errorf("failed to load %s messages: %w", status, err)
		}
		for _, m := range msgs {
			if m.ID == id {
				return m, nil
			}
		}
	}
	return db.Message{}, fmt.Errorf("no drafted or escalated message found with id %d", id)
}

func init() {
	replyCmd.Flags().IntVar(&replyID, "id", 0, "Stored message id to reply with")
	replyCmd.Flags().BoolVar(&replySend, "send", false, "Actually send the reply (otherwise just print it)")
	rootCmd.AddCommand(replyCmd)
}
