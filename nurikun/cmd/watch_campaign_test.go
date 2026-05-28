package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/yellowhama/musu-system/nurikun/internal/db"
	"github.com/yellowhama/musu-system/nurikun/internal/mailbox"
)

type fakeMailbox struct {
	mu      sync.Mutex
	fetches [][]mailbox.Message
	sent    []mailbox.OutMessage
	marked  []string
}

func (f *fakeMailbox) Fetch(limit int) ([]mailbox.Message, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.fetches) == 0 {
		return nil, nil
	}
	batch := f.fetches[0]
	f.fetches = f.fetches[1:]
	if len(batch) > limit {
		return append([]mailbox.Message(nil), batch[:limit]...), nil
	}
	return append([]mailbox.Message(nil), batch...), nil
}

func (f *fakeMailbox) Send(msg mailbox.OutMessage) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, msg)
	return nil
}

func (f *fakeMailbox) Mark(uid string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.marked = append(f.marked, uid)
	return nil
}

func (f *fakeMailbox) Close() error { return nil }

func writeTestConfig(t *testing.T, project string, content string) {
	t.Helper()
	path := filepath.Join("projects", project, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	fn()

	_ = w.Close()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	_ = r.Close()
	return buf.String()
}

func TestWatchCommandWithFakeMailbox(t *testing.T) {
	wd, _ := os.Getwd()
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	viper.Reset()
	project := "watch-smoke"
	baseDir, _, err := bootstrapProject(project, false, "imap", "folder")
	if err != nil {
		t.Fatalf("bootstrapProject failed: %v", err)
	}

	knowledgeDir := filepath.Join(baseDir, "knowledge")
	if err := os.WriteFile(filepath.Join(knowledgeDir, "faq.md"), []byte("# Shipping FAQ\n\nWe usually respond within one business day and can help with order status updates."), 0o644); err != nil {
		t.Fatal(err)
	}

	ai := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		prompt := payload.Messages[0].Content

		reply := ""
		if strings.Contains(prompt, "customer-support triage assistant") {
			reply = `{"category":"faq","intent":"customer asks about shipping response time","language":"en","confidence":0.96,"sensitive":false}`
		} else {
			reply = "Hi, we usually respond within one business day and can help with your order status."
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": reply}},
			},
		})
	}))
	defer ai.Close()

	writeTestConfig(t, project, "db_path: \""+filepath.ToSlash(filepath.Join(baseDir, "data", "nurikun.db"))+"\"\n"+
		"ai_provider: openai\n"+
		"ai_model: llama3\n"+
		"ai_url: \""+ai.URL+"\"\n"+
		"mailbox_provider: fake\n"+
		"knowledge_source: folder\n"+
		"knowledge_dir: \""+filepath.ToSlash(knowledgeDir)+"\"\n"+
		"sender_name: \"Support Team\"\n"+
		"sender_address: \"support@example.com\"\n"+
		"sender_physical: \"Seoul HQ\"\n"+
		"public_base_url: \"https://example.com\"\n"+
		"unsub_secret: \"super-secret\"\n")

	fake := &fakeMailbox{
		fetches: [][]mailbox.Message{
			{
				{
					UID:       "uid-1",
					ThreadID:  "thread-1",
					From:      "customer@example.com",
					To:        []string{"support@example.com"},
					Subject:   "Shipping help",
					Body:      "When do you usually respond to shipping questions?",
					MessageID: "<msg-1@example.com>",
					Date:      time.Now(),
				},
			},
		},
	}
	restore := mailbox.RegisterFactory("fake", func(s mailbox.Settings) (mailbox.Mailbox, error) {
		return fake, nil
	})
	defer restore()

	viper.Set("project", project)
	watchLimit = 10
	out := captureStdout(t, func() {
		watchCmd.Run(watchCmd, nil)
	})
	if !strings.Contains(out, "auto-sent") {
		t.Fatalf("expected watch output to mention auto-sent, got:\n%s", out)
	}

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

	sent, err := store.MessagesByStatus("sent")
	if err != nil {
		t.Fatal(err)
	}
	if len(sent) != 1 {
		t.Fatalf("expected 1 sent message, got %d", len(sent))
	}

	if len(fake.sent) != 1 {
		t.Fatalf("expected fake mailbox to send 1 message, got %d", len(fake.sent))
	}
	if len(fake.marked) != 1 || fake.marked[0] != "uid-1" {
		t.Fatalf("expected uid-1 to be marked, got %+v", fake.marked)
	}
	if !strings.Contains(strings.ToLower(fake.sent[0].Body), "one business day") {
		t.Fatalf("expected grounded reply body, got %q", fake.sent[0].Body)
	}
}

func TestCampaignCommandWithFakeMailbox(t *testing.T) {
	wd, _ := os.Getwd()
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	viper.Reset()
	project := "campaign-smoke"
	baseDir, _, err := bootstrapProject(project, false, "imap", "none")
	if err != nil {
		t.Fatalf("bootstrapProject failed: %v", err)
	}

	writeTestConfig(t, project, "db_path: \""+filepath.ToSlash(filepath.Join(baseDir, "data", "nurikun.db"))+"\"\n"+
		"mailbox_provider: fake\n"+
		"sender_name: \"Nongjida Team\"\n"+
		"sender_address: \"news@example.com\"\n"+
		"sender_physical: \"Seoul HQ\"\n"+
		"public_base_url: \"https://example.com\"\n"+
		"unsub_secret: \"campaign-secret\"\n")

	fake := &fakeMailbox{}
	restore := mailbox.RegisterFactory("fake", func(s mailbox.Settings) (mailbox.Mailbox, error) {
		return fake, nil
	})
	defer restore()

	store, err := db.NewStore(filepath.Join(baseDir, "data", "nurikun.db"))
	if err != nil {
		t.Fatal(err)
	}
	listID, err := store.CreateList("Weekly Digest", 4)
	if err != nil {
		t.Fatal(err)
	}
	token, err := store.AddSubscriber(int(listID), "subscriber@example.com", "Subscriber", "self-subscribe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ConfirmSubscriber(token); err != nil {
		t.Fatal(err)
	}
	_ = store.Close()

	viper.Set("project", project)
	campaignList = int(listID)
	campaignName = "weekly-digest"
	campaignSubject = "Fresh land insights"
	campaignBody = "Here are this week's verified land insights."
	campaignPersona = "calm-expert"
	campaignDryRun = false

	out := captureStdout(t, func() {
		if err := campaignCmd.RunE(campaignCmd, nil); err != nil {
			t.Fatalf("campaign run failed: %v", err)
		}
	})
	if !strings.Contains(out, "Summary: 1 sent") {
		t.Fatalf("expected send summary, got:\n%s", out)
	}

	if len(fake.sent) != 1 {
		t.Fatalf("expected one outbound campaign email, got %d", len(fake.sent))
	}
	msg := fake.sent[0]
	if !strings.HasPrefix(msg.Subject, "(광고) ") {
		t.Fatalf("expected ad prefix on subject, got %q", msg.Subject)
	}
	if !strings.Contains(msg.Body, "보낸 사람: Nongjida Team") {
		t.Fatalf("expected compliance footer in body, got %q", msg.Body)
	}
	if got := msg.Headers["List-Unsubscribe"]; !strings.Contains(got, "https://example.com/unsubscribe?email=subscriber%40example.com") {
		t.Fatalf("expected unsubscribe header, got %q", got)
	}
	if msg.Headers["List-Unsubscribe-Post"] != "List-Unsubscribe=One-Click" {
		t.Fatalf("expected one-click header, got %+v", msg.Headers)
	}

	store, err = db.NewStore(filepath.Join(baseDir, "data", "nurikun.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	due, err := store.DueSubscribers(int(listID), 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 0 {
		t.Fatalf("expected cadence anchor to remove subscriber from due set, got %d due", len(due))
	}
}
