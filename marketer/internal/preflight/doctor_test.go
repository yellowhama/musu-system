package preflight

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestEvaluateDoctorBlocksOnFixFailure(t *testing.T) {
	wd, _ := os.Getwd()
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	wikiDir := filepath.Join(tmp, "wiki")
	if err := os.MkdirAll(wikiDir, 0o755); err != nil {
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
		Project:    "missing-project",
		WikiDir:    wikiDir,
		AIProvider: "openai",
		AIURL:      srv.URL,
		AutoFix:    true,
		FixProject: func() error { return fmt.Errorf("boom") },
	})

	if !result.Blocking {
		t.Fatalf("expected blocking result when FixProject fails")
	}
	if result.Report.ProjectFixError == "" {
		t.Fatalf("expected project_fix_error to be recorded")
	}
}
