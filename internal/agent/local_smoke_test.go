package agent_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yellowhama/musu-nurikun/internal/agent"
	"github.com/yellowhama/musu-nurikun/internal/knowledge"
	"github.com/yellowhama/musu-nurikun/internal/policy"
	"github.com/yellowhama/musu-nurikun/internal/triage"
)

func TestLocalTriageAndRespondHappyPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "refund.md"), []byte("# Refund Policy\n\nRefunds are processed within 7 days after billing confirms the request."), 0o644); err != nil {
		t.Fatal(err)
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
		if strings.Contains(prompt, "customer-support triage assistant") {
			reply = `{"category":"refund","intent":"customer wants a refund update","language":"en","confidence":0.93,"sensitive":true}`
		} else {
			reply = "Hi, refunds are processed within 7 days after billing confirms the request. If you need more help, a human colleague can follow up."
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": reply}},
			},
		})
	}))
	defer srv.Close()

	client := agent.NewAgentClient(srv.URL, "llama3", dir, "demo")
	tr, err := triage.Classify(client, "Refund status", "Can you tell me when my refund will be processed?")
	if err != nil {
		t.Fatalf("Classify failed: %v", err)
	}
	if tr.Category != policy.CategoryRefund {
		t.Fatalf("expected refund category, got %q", tr.Category)
	}

	src, err := knowledge.New(knowledge.Settings{Kind: "folder", Dir: dir})
	if err != nil {
		t.Fatalf("knowledge.New failed: %v", err)
	}
	snippets, err := src.Retrieve(tr.Intent, 5)
	if err != nil {
		t.Fatalf("Retrieve failed: %v", err)
	}
	if len(snippets) == 0 {
		t.Fatal("expected grounding snippets")
	}

	reply, err := agent.Respond(client, "Can you tell me when my refund will be processed?", snippets, "Warm and concise")
	if err != nil {
		t.Fatalf("Respond failed: %v", err)
	}
	if !strings.Contains(strings.ToLower(reply), "7 days") {
		t.Fatalf("expected grounded reply to mention refund timing, got %q", reply)
	}
}
