package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFetchCommandWebHappyPath(t *testing.T) {
	wd, _ := os.Getwd()
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html><head><title>Scheduler Test Page</title></head><body><main><h1>Scheduler Reliability</h1><p>Schedulers need reliability, timing, and operator trust.</p></main></body></html>`))
	}))
	defer page.Close()

	ai := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/embeddings":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"embedding": []float64{0.1, 0.2, 0.3}},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer ai.Close()

	wikiDir := "./wiki"
	if err := bootstrapProjectDirs(wikiDir, "demo", "openai", ai.URL+"/v1", false); err != nil {
		t.Fatalf("bootstrapProjectDirs failed: %v", err)
	}
	if err := os.Setenv("MUSU_OUT", wikiDir); err != nil {
		t.Fatal(err)
	}
	defer os.Unsetenv("MUSU_OUT")

	if err := fetchCmd.Flags().Set("project", "demo"); err != nil {
		t.Fatal(err)
	}
	if err := fetchCmd.Flags().Set("out", wikiDir); err != nil {
		t.Fatal(err)
	}
	if err := fetchCmd.Flags().Set("compile", "false"); err != nil {
		t.Fatal(err)
	}
	if err := fetchCmd.Flags().Set("model", "llama3"); err != nil {
		t.Fatal(err)
	}

	fetchCmd.Run(fetchCmd, []string{"web", page.URL})

	indexPath := filepath.Join(wikiDir, "index.json")
	indexBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("expected index.json to exist: %v", err)
	}
	indexText := string(indexBytes)
	for _, want := range []string{
		`"title": "Scheduler Test Page"`,
		`"project": "demo"`,
		`Scheduler Reliability`,
	} {
		if !strings.Contains(indexText, want) {
			t.Fatalf("expected index to contain %q, got:\n%s", want, indexText)
		}
	}

	entries, err := filepath.Glob(filepath.Join(wikiDir, "projects", "demo", "web", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one fetched markdown file, got %d", len(entries))
	}

	docBytes, err := os.ReadFile(entries[0])
	if err != nil {
		t.Fatal(err)
	}
	docText := string(docBytes)
	if !strings.Contains(docText, "Scheduler Reliability") {
		t.Fatalf("expected fetched markdown to contain extracted content, got:\n%s", docText)
	}
}
