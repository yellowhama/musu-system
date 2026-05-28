//go:build integration

package cmd

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func captureCommandStdout(t *testing.T, fn func()) string {
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

func TestFetchAndSemanticSearchRealAIIntegration(t *testing.T) {
	aiURL := strings.TrimSpace(os.Getenv("MUSU_CRAWL_INTEGRATION_AI_URL"))
	if aiURL == "" {
		t.Skip("set MUSU_CRAWL_INTEGRATION_AI_URL to run real integration")
	}
	embedModel := strings.TrimSpace(os.Getenv("MUSU_CRAWL_INTEGRATION_EMBED_MODEL"))
	if embedModel == "" {
		embedModel = "nomic-embed-text"
	}

	wd, _ := os.Getwd()
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html><head><title>Scheduler Integration Page</title></head><body><main><h1>Scheduler Reliability</h1><p>Schedulers need reliability, timing, and operator trust.</p></main></body></html>`))
	}))
	defer page.Close()

	wikiDir := "./wiki"
	if err := bootstrapProjectDirs(wikiDir, "integration-demo", "openai", aiURL, false); err != nil {
		t.Fatalf("bootstrapProjectDirs failed: %v", err)
	}
	if err := os.Setenv("MUSU_OUT", wikiDir); err != nil {
		t.Fatal(err)
	}
	defer os.Unsetenv("MUSU_OUT")

	if err := fetchCmd.Flags().Set("project", "integration-demo"); err != nil {
		t.Fatal(err)
	}
	if err := fetchCmd.Flags().Set("out", wikiDir); err != nil {
		t.Fatal(err)
	}
	if err := fetchCmd.Flags().Set("compile", "false"); err != nil {
		t.Fatal(err)
	}
	if err := fetchCmd.Flags().Set("model", embedModel); err != nil {
		t.Fatal(err)
	}
	fetchCmd.Run(fetchCmd, []string{"web", page.URL})

	if _, err := os.Stat(filepath.Join(wikiDir, "musu.vectors.json")); err != nil {
		t.Fatalf("expected vector store after real fetch integration: %v", err)
	}

	if err := searchCmd.Flags().Set("out", wikiDir); err != nil {
		t.Fatal(err)
	}
	if err := searchCmd.Flags().Set("project", "integration-demo"); err != nil {
		t.Fatal(err)
	}
	if err := searchCmd.Flags().Set("semantic", "true"); err != nil {
		t.Fatal(err)
	}
	if err := searchCmd.Flags().Set("model", embedModel); err != nil {
		t.Fatal(err)
	}
	if err := searchCmd.Flags().Set("json", "true"); err != nil {
		t.Fatal(err)
	}
	viper.Set("json", true)

	out := captureCommandStdout(t, func() {
		searchCmd.Run(searchCmd, []string{"scheduler"})
	})
	if !strings.Contains(out, `"status": "success"`) {
		t.Fatalf("expected successful semantic search JSON, got:\n%s", out)
	}
	if !strings.Contains(out, `Scheduler Integration Page`) {
		t.Fatalf("expected semantic search result to mention fetched page, got:\n%s", out)
	}
}

func TestResearchCommandRealAIIntegration(t *testing.T) {
	aiURL := strings.TrimSpace(os.Getenv("MUSU_CRAWL_INTEGRATION_AI_URL"))
	if aiURL == "" {
		t.Skip("set MUSU_CRAWL_INTEGRATION_AI_URL to run real integration")
	}
	chatModel := strings.TrimSpace(os.Getenv("MUSU_CRAWL_INTEGRATION_CHAT_MODEL"))
	if chatModel == "" {
		chatModel = "llama3"
	}

	wd, _ := os.Getwd()
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	contentPages := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.Path {
		case "/p1":
			_, _ = w.Write([]byte(`<!doctype html><html><head><title>Scheduler Reliability Basics</title></head><body><main><h1>Scheduler Reliability</h1><p>Reliable schedulers need time guarantees, retries, and operator trust.</p></main></body></html>`))
		case "/p2":
			_, _ = w.Write([]byte(`<!doctype html><html><head><title>Scheduler Incident Review</title></head><body><main><h1>Incident Lessons</h1><p>Backpressure, clear alerting, and queue visibility reduce scheduler outages.</p></main></body></html>`))
		case "/p3":
			_, _ = w.Write([]byte(`<!doctype html><html><head><title>Operator Runbook</title></head><body><main><h1>Runbook</h1><p>Runbooks and ownership improve scheduler recovery time.</p></main></body></html>`))
		case "/p4":
			_, _ = w.Write([]byte(`<!doctype html><html><head><title>Capacity Guide</title></head><body><main><h1>Capacity</h1><p>Capacity planning and queue telemetry keep scheduler latency stable.</p></main></body></html>`))
		default:
			_, _ = w.Write([]byte(`<!doctype html><html><head><title>Fallback</title></head><body><main><p>Scheduler fallback reference.</p></main></body></html>`))
		}
	}))
	defer contentPages.Close()

	searchPage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		base := contentPages.URL
		_, _ = w.Write([]byte(`
<!doctype html><html><body>
<a class="result__url" href="` + base + `/p1">scheduler 1</a>
<a class="result__url" href="` + base + `/p2">scheduler 2</a>
<a class="result__url" href="` + base + `/p3">scheduler 3</a>
<a class="result__url" href="` + base + `/p4">scheduler 4</a>
<a class="result__url" href="` + base + `/p1">scheduler duplicate</a>
</body></html>`))
	}))
	defer searchPage.Close()

	wikiDir := "./wiki"
	if err := bootstrapProjectDirs(wikiDir, "integration-research", "openai", aiURL, false); err != nil {
		t.Fatalf("bootstrapProjectDirs failed: %v", err)
	}

	if err := os.Setenv("MUSU_SEARCH_BASE_URL", searchPage.URL); err != nil {
		t.Fatal(err)
	}
	defer os.Unsetenv("MUSU_SEARCH_BASE_URL")

	if err := researchCmd.Flags().Set("project", "integration-research"); err != nil {
		t.Fatal(err)
	}
	if err := researchCmd.Flags().Set("out", wikiDir); err != nil {
		t.Fatal(err)
	}
	if err := researchCmd.Flags().Set("depth", "1"); err != nil {
		t.Fatal(err)
	}
	if err := researchCmd.Flags().Set("model", chatModel); err != nil {
		t.Fatal(err)
	}
	if err := researchCmd.Flags().Set("json", "true"); err != nil {
		t.Fatal(err)
	}
	viper.Set("json", true)

	out := captureCommandStdout(t, func() {
		researchCmd.Run(researchCmd, []string{"What makes scheduler reliability strong?"})
	})
	if !strings.Contains(out, `"status": "success"`) {
		t.Fatalf("expected successful research JSON, got:\n%s", out)
	}
	if !strings.Contains(out, `"message": "Research completed"`) {
		t.Fatalf("expected research completion message, got:\n%s", out)
	}
	if !strings.Contains(out, `"question": "What makes scheduler reliability strong?"`) {
		t.Fatalf("expected echoed question in JSON output, got:\n%s", out)
	}

	if _, err := os.Stat(filepath.Join(wikiDir, "index.json")); err != nil {
		t.Fatalf("expected index.json after research integration: %v", err)
	}
}
