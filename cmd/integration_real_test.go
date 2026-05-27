//go:build integration

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/yellowhama/musu-nurikun/internal/db"
	"github.com/yellowhama/musu-nurikun/internal/mailbox"
)

func TestWatchCommandRealAIIntegration(t *testing.T) {
	aiURL := strings.TrimSpace(os.Getenv("MUSU_NURIKUN_INTEGRATION_AI_URL"))
	if aiURL == "" {
		t.Skip("set MUSU_NURIKUN_INTEGRATION_AI_URL to run real integration")
	}
	model := strings.TrimSpace(os.Getenv("MUSU_NURIKUN_INTEGRATION_MODEL"))
	if model == "" {
		model = "llama3"
	}

	wd, _ := os.Getwd()
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	viper.Reset()
	project := "integration-watch"
	baseDir, _, err := bootstrapProject(project, false, "imap", "folder")
	if err != nil {
		t.Fatalf("bootstrapProject failed: %v", err)
	}

	knowledgeDir := filepath.Join(baseDir, "knowledge")
	if err := os.WriteFile(filepath.Join(knowledgeDir, "faq.md"), []byte("# Shipping FAQ\n\nWe usually respond within one business day and can help with order status updates."), 0o644); err != nil {
		t.Fatal(err)
	}

	writeTestConfig(t, project, "db_path: \""+filepath.ToSlash(filepath.Join(baseDir, "data", "nurikun.db"))+"\"\n"+
		"ai_provider: openai\n"+
		"ai_model: "+model+"\n"+
		"ai_url: \""+aiURL+"\"\n"+
		"mailbox_provider: fake-real\n"+
		"knowledge_source: folder\n"+
		"knowledge_dir: \""+filepath.ToSlash(knowledgeDir)+"\"\n"+
		"sender_name: \"Support Team\"\n"+
		"sender_address: \"support@example.com\"\n"+
		"sender_physical: \"Seoul HQ\"\n"+
		"public_base_url: \"https://example.com\"\n"+
		"unsub_secret: \"integration-secret\"\n")

	fake := &fakeMailbox{
		fetches: [][]mailbox.Message{
			{
				{
					UID:       "uid-integration",
					ThreadID:  "thread-integration",
					From:      "customer@example.com",
					To:        []string{"support@example.com"},
					Subject:   "Shipping help",
					Body:      "When do you usually respond to shipping questions?",
					MessageID: "<msg-int@example.com>",
					Date:      time.Now(),
				},
			},
		},
	}
	restore := mailbox.RegisterFactory("fake-real", func(s mailbox.Settings) (mailbox.Mailbox, error) {
		return fake, nil
	})
	defer restore()

	viper.Set("project", project)
	watchLimit = 10
	_ = captureStdout(t, func() {
		watchCmd.Run(watchCmd, nil)
	})

	store, err := db.NewStore(filepath.Join(baseDir, "data", "nurikun.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	received, err := store.MessagesByStatus("received")
	if err != nil {
		t.Fatal(err)
	}
	if len(received) != 1 {
		t.Fatalf("expected 1 received message, got %d", len(received))
	}

	outboundStatuses := []string{"drafted", "sent", "escalated"}
	totalOutbound := 0
	for _, status := range outboundStatuses {
		msgs, err := store.MessagesByStatus(status)
		if err != nil {
			t.Fatal(err)
		}
		totalOutbound += len(msgs)
	}
	if totalOutbound == 0 {
		t.Fatal("expected at least one outbound record after real-AI watch integration")
	}
}
