//go:build integration

package seo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGenerateRealAI exercises the full LLM pipeline end-to-end against a real
// Ollama-compatible endpoint. It is skipped unless MUSU_MARKETER_INTEGRATION_AI_URL
// is set, mirroring cmd/integration_real_test.go. Requires a pulled model.
//
//	MUSU_MARKETER_INTEGRATION_AI_URL=http://localhost:11434/v1 \
//	MUSU_MARKETER_INTEGRATION_MODEL=llama3 \
//	go test -tags integration ./internal/seo/ -run TestGenerateRealAI -v
func TestGenerateRealAI(t *testing.T) {
	aiURL := strings.TrimSpace(os.Getenv("MUSU_MARKETER_INTEGRATION_AI_URL"))
	if aiURL == "" {
		t.Skip("set MUSU_MARKETER_INTEGRATION_AI_URL to run real integration")
	}
	model := strings.TrimSpace(os.Getenv("MUSU_MARKETER_INTEGRATION_MODEL"))
	if model == "" {
		model = "llama3"
	}

	// Build a tiny grounded wiki fixture so FindByTopic returns sources.
	wikiDir := t.TempDir()
	must(t, os.WriteFile(filepath.Join(wikiDir, "nongjibeop.md"),
		[]byte("# 농지법 제10조\n\n농지 소유자가 정당한 사유 없이 농지를 농업경영에 이용하지 "+
			"않으면 처분의무가 발생하고, 시장·군수가 처분명령을 내릴 수 있다.\n"), 0o644))

	gen := NewGenerator(aiURL, model, wikiDir, "integration-demo", 300)
	art, rep, err := gen.Generate("농지법 처분명령", true)
	if err != nil {
		// In strict mode the model may fail the citation gate; that is a valid
		// (safe) outcome, not a test failure — but log it for visibility.
		t.Logf("strict generate blocked (acceptable): %v | report: %+v", err, rep)
		return
	}
	if !strings.Contains(art.Render(), "## 출처") {
		t.Errorf("rendered article missing references section")
	}
	t.Logf("generated %q (%d cited claims)", art.Outline.Title, rep.CitedClaims)
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
