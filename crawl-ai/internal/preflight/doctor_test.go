package preflight

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvaluateDoctorReportsProjectAndAIFixes(t *testing.T) {
	tmp := t.TempDir()
	wikiDir := filepath.Join(tmp, "wiki")
	if err := os.MkdirAll(wikiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wikiDir, "note.md"), []byte("# note"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(wikiDir, "musu.bleve"), 0o755); err != nil {
		t.Fatal(err)
	}

	result := EvaluateDoctor(DoctorOptions{
		Out:               wikiDir,
		Project:           "missing-project",
		AIProvider:        "openai",
		AIURL:             "http://127.0.0.1:9",
		CapabilitySources: []string{"web"},
	})

	if !result.Blocking {
		t.Fatalf("expected blocking result for missing project and dead AI endpoint")
	}
	if !strings.Contains(result.ActionableFix, "wiki/projects/missing-project") {
		t.Fatalf("expected actionable fix to mention missing project path, got %q", result.ActionableFix)
	}
	if !strings.Contains(result.ActionableFix, "reachable --ai-url") {
		t.Fatalf("expected actionable fix to mention reachable ai-url, got %q", result.ActionableFix)
	}
	if !strings.Contains(result.ActionableFix, "static setup metadata") {
		t.Fatalf("expected actionable fix to mention static capability metadata, got %q", result.ActionableFix)
	}
}

func TestEvaluateDoctorNoActionRequiredWhenReady(t *testing.T) {
	tmp := t.TempDir()
	wikiDir := filepath.Join(tmp, "wiki")
	projectDir := filepath.Join(wikiDir, "projects", "ready-project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wikiDir, "note.md"), []byte("# note"), 0o644); err != nil {
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
		Out:        wikiDir,
		Project:    "ready-project",
		AIProvider: "openai",
		AIURL:      srv.URL,
	})

	if result.Blocking {
		t.Fatalf("expected non-blocking result for ready project")
	}
	if result.ActionableFix != "No action required." {
		t.Fatalf("expected no action required, got %q", result.ActionableFix)
	}
}
