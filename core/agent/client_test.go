package agent

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// stubChat returns a server that answers /chat/completions with a fixed body
// and /embeddings with a one-element vector.
func stubChat(t *testing.T, replyContent string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		// Echo the request body's "response_format" presence by switching reply if needed.
		var req struct {
			ResponseFormat *responseFormat `json:"response_format,omitempty"`
		}
		_ = json.Unmarshal(body, &req)
		out := chatResponse{}
		out.Choices = append(out.Choices, struct {
			Message chatMessage `json:"message"`
		}{Message: chatMessage{Role: "assistant", Content: replyContent}})
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})
	mux.HandleFunc("/embeddings", func(w http.ResponseWriter, r *http.Request) {
		out := embedResponse{}
		out.Data = append(out.Data, struct {
			Embedding []float64 `json:"embedding"`
		}{Embedding: []float64{0.1, 0.2, 0.3}})
		_ = json.NewEncoder(w).Encode(out)
	})
	return httptest.NewServer(mux)
}

func TestNew_Defaults(t *testing.T) {
	c := New("", "")
	if c.BaseURL != "http://localhost:11434/v1" {
		t.Errorf("BaseURL default: got %q", c.BaseURL)
	}
	if c.Model != "llama3" {
		t.Errorf("Model default: got %q", c.Model)
	}
	if c.VisionModel != "llava" {
		t.Errorf("VisionModel default: got %q", c.VisionModel)
	}
	if c.Role != "assistant" {
		t.Errorf("Role default: got %q", c.Role)
	}
	if c.telemetryEnabled() {
		t.Errorf("telemetry should be OFF without WithTelemetry")
	}
}

func TestNew_OptionsApply(t *testing.T) {
	c := New("http://example/v1", "qwen",
		WithVisionModel("llava-13b"),
		WithTelemetry("/tmp/wiki", "proj-a"),
		WithRole("nurikun_agent"),
	)
	if c.VisionModel != "llava-13b" || c.Role != "nurikun_agent" || c.WikiDir != "/tmp/wiki" || c.Project != "proj-a" {
		t.Errorf("options not applied: %+v", c)
	}
	if !c.telemetryEnabled() {
		t.Error("telemetry should be ON with WithTelemetry")
	}
}

func TestAsk_TelemetryOff_NoFilesWritten(t *testing.T) {
	srv := stubChat(t, "hello world")
	defer srv.Close()
	c := New(srv.URL, "m", WithHTTPClient(srv.Client()))
	got, err := c.Ask("hi", false)
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if got != "hello world" {
		t.Errorf("Ask result: got %q", got)
	}
}

func TestAsk_TelemetryOn_WritesTraceFile(t *testing.T) {
	srv := stubChat(t, "telemetry-on result")
	defer srv.Close()

	wiki := t.TempDir()
	c := New(srv.URL, "m",
		WithHTTPClient(srv.Client()),
		WithTelemetry(wiki, "proj"),
		WithRole("nurikun_agent"),
	)
	if _, err := c.Ask("hi", true); err != nil {
		t.Fatalf("Ask: %v", err)
	}

	// Expect exactly one trace file under wiki/telemetry/<date>/*.json
	tdir := filepath.Join(wiki, "telemetry")
	dates, err := os.ReadDir(tdir)
	if err != nil {
		t.Fatalf("expected telemetry dir, got: %v", err)
	}
	if len(dates) != 1 {
		t.Fatalf("expected one date dir, got %d", len(dates))
	}
	entries, _ := os.ReadDir(filepath.Join(tdir, dates[0].Name()))
	if len(entries) != 1 {
		t.Fatalf("expected one trace file, got %d", len(entries))
	}
	name := entries[0].Name()
	if !strings.HasPrefix(name, "nurikun_agent_proj_") {
		t.Errorf("trace filename should embed role+project: got %q", name)
	}

	// Trace JSON should record the role + status.
	data, _ := os.ReadFile(filepath.Join(tdir, dates[0].Name(), name))
	var tr ExecutionTrace
	if err := json.Unmarshal(data, &tr); err != nil {
		t.Fatalf("trace not JSON: %v", err)
	}
	if tr.Role != "nurikun_agent" || tr.Status != "success" || tr.Response != "telemetry-on result" {
		t.Errorf("trace contents off: %+v", tr)
	}
}

func TestAsk_NonOKStatus_ReturnsError(t *testing.T) {
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "model not found", http.StatusNotFound)
	}))
	defer bad.Close()
	c := New(bad.URL, "m", WithHTTPClient(bad.Client()))
	if _, err := c.Ask("hi", false); err == nil {
		t.Fatal("expected error on 404")
	}
}

func TestEmbed_ReturnsVector(t *testing.T) {
	srv := stubChat(t, "ignored")
	defer srv.Close()
	c := New(srv.URL, "embed-model", WithHTTPClient(srv.Client()))
	vec, err := c.Embed("text")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vec) != 3 || vec[0] != 0.1 {
		t.Errorf("unexpected vector: %v", vec)
	}
}
