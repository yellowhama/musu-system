package preflight

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvaluateDoctorReportsTopicAndAIFixes(t *testing.T) {
	wd, _ := os.Getwd()
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	wikiDir := filepath.Join(tmp, "wiki")
	projectDir := filepath.Join(tmp, "projects", "ready-project", "data")
	if err := os.MkdirAll(wikiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wikiDir, "note.md"), []byte("# note"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "marketer.db"), []byte("db"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := EvaluateDoctor(DoctorOptions{
		Project:    "ready-project",
		WikiDir:    wikiDir,
		AIProvider: "openai",
		AIURL:      "http://127.0.0.1:9",
		Topic:      "scheduler",
	})

	if !result.Blocking {
		t.Fatalf("expected blocking result for missing topic grounding and dead AI endpoint")
	}
	if !strings.Contains(result.ActionableFix, `add or sync wiki content for topic "scheduler"`) {
		t.Fatalf("expected actionable fix to mention missing topic grounding, got %q", result.ActionableFix)
	}
	if !strings.Contains(result.ActionableFix, "reachable --ai-url") {
		t.Fatalf("expected actionable fix to mention reachable ai-url, got %q", result.ActionableFix)
	}
}

func TestEvaluateDoctorNoActionRequiredWhenReady(t *testing.T) {
	wd, _ := os.Getwd()
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	wikiDir := filepath.Join(tmp, "wiki")
	projectDir := filepath.Join(tmp, "projects", "ready-project", "data")
	if err := os.MkdirAll(filepath.Join(wikiDir, "topics"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wikiDir, "index.json"), []byte(`{"pages":[{"slug":"scheduler-overview","title":"Scheduler Deep Dive","summary":"Topic coverage","tags":["scheduler"]}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wikiDir, "topics", "scheduler-overview.md"), []byte("# Scheduler Deep Dive\nThis note covers scheduler operations."), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "marketer.db"), []byte("db"), 0o644); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[]}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	result := EvaluateDoctor(DoctorOptions{
		Project:    "ready-project",
		WikiDir:    wikiDir,
		AIProvider: "openai",
		AIURL:      srv.URL,
		Topic:      "scheduler",
	})

	if result.Blocking {
		t.Fatalf("expected non-blocking result for ready project")
	}
	if result.ActionableFix != "No action required." {
		t.Fatalf("expected no action required, got %q", result.ActionableFix)
	}
}
