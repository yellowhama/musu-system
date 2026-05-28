package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/yellowhama/musu-marketer/internal/db"
)

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
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	_ = r.Close()
	return buf.String()
}

func setupDraftFixture(t *testing.T, project string) (string, string, *httptest.Server) {
	t.Helper()

	wikiDir := filepath.Join(t.TempDir(), "wiki")
	if err := os.MkdirAll(filepath.Join(wikiDir, "topics"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wikiDir, "index.json"), []byte(`{"pages":[{"id":"scheduler-overview","title":"Scheduler Deep Dive","source":"web","project":"`+project+`","path":"topics/scheduler-overview.md","tags":["scheduler"],"summary":"Grounded scheduler notes"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wikiDir, "topics", "scheduler-overview.md"), []byte("# Scheduler Deep Dive\n\nScheduler operations require reliability and timing."), 0o644); err != nil {
		t.Fatal(err)
	}

	viper.Set("wiki_dir", wikiDir)
	viper.Set("ai_provider", "openai")

	baseDir, dbPath, err := bootstrapProject(project, false)
	if err != nil {
		t.Fatalf("bootstrapProject failed: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		switch {
		case strings.Contains(prompt, "Senior Marketing Strategist"):
			reply = `{"value_proposition":"Scheduler reliability","target_segment":"Ops teams","triggers":["urgency","authority"],"primary_goal":"Drive trust","selected_framework":"AIDA"}`
		case strings.Contains(prompt, "MARKETING MASTERY AUDITOR"):
			reply = `{"approved":true,"feedback":"looks good"}`
		default:
			reply = "Twitter Thread:\n1. Scheduler reliability matters.\n\nBlog Post:\nOperational confidence starts with strong scheduling."
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": reply}},
			},
		})
	}))

	viper.Set("ai_url", srv.URL)
	if !strings.Contains(baseDir, filepath.Join("projects", project)) {
		t.Fatalf("expected project baseDir to be under projects/%s, got %s", project, baseDir)
	}
	return wikiDir, dbPath, srv
}

func TestExecuteDraftHappyPath(t *testing.T) {
	wd, _ := os.Getwd()
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	wikiDir, dbPath, srv := setupDraftFixture(t, "demo")
	defer srv.Close()

	if err := ExecuteDraft("scheduler", wikiDir, dbPath, "llama3", "default", "demo"); err != nil {
		t.Fatalf("ExecuteDraft failed: %v", err)
	}

	store, err := db.NewStore(dbPath)
	if err != nil {
		t.Fatalf("reopen db: %v", err)
	}
	defer store.Close()

	campaigns, err := store.ListCampaigns()
	if err != nil {
		t.Fatalf("ListCampaigns failed: %v", err)
	}
	if len(campaigns) != 1 {
		t.Fatalf("expected 1 campaign, got %d", len(campaigns))
	}
	if campaigns[0].Topic != "scheduler" {
		t.Fatalf("expected topic scheduler, got %q", campaigns[0].Topic)
	}
}

func TestDraftCommandHappyPath(t *testing.T) {
	wd, _ := os.Getwd()
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	_, dbPath, srv := setupDraftFixture(t, "demo")
	defer srv.Close()

	viper.Set("project", "demo")
	viper.Set("json", false)
	if err := draftCmd.Flags().Set("model", "llama3"); err != nil {
		t.Fatal(err)
	}
	if err := draftCmd.Flags().Set("persona", "default"); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		draftCmd.Run(draftCmd, []string{"scheduler"})
	})
	if !strings.Contains(out, "FINAL STRATEGIC CAMPAIGN") {
		t.Fatalf("expected final campaign output, got:\n%s", out)
	}
	if !strings.Contains(out, "Scheduler reliability matters") {
		t.Fatalf("expected generated campaign text, got:\n%s", out)
	}

	store, err := db.NewStore(dbPath)
	if err != nil {
		t.Fatalf("reopen db: %v", err)
	}
	defer store.Close()
	campaigns, err := store.ListCampaigns()
	if err != nil {
		t.Fatalf("ListCampaigns failed: %v", err)
	}
	if len(campaigns) != 1 {
		t.Fatalf("expected 1 campaign after draft command, got %d", len(campaigns))
	}
}
