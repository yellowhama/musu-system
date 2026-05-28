package agent

import (
	"path/filepath"
	"testing"

	"github.com/yellowhama/musu-crawl-ai/internal/processor"
)

func TestSearchActionHappyPath(t *testing.T) {
	tmp := t.TempDir()
	proc := processor.NewWikiProcessor(tmp, "demo")
	if _, err := proc.SaveToWiki("web", "https://example.com/scheduler", "Scheduler Deep Dive", "Scheduler operations require reliability and timing.", []string{"scheduler", "ops"}, "Grounded scheduler notes", 0.9); err != nil {
		t.Fatalf("SaveToWiki failed: %v", err)
	}
	if err := proc.UpdateIndex(); err != nil {
		t.Fatalf("UpdateIndex failed: %v", err)
	}

	orchestrator := NewOrchestrator(tmp)
	results, err := orchestrator.SearchAction("scheduler", "demo", false, "")
	if err != nil {
		t.Fatalf("SearchAction failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one search result")
	}
	if results[0].Project != "demo" {
		t.Fatalf("expected project demo, got %q", results[0].Project)
	}
	if results[0].Source != "web" {
		t.Fatalf("expected source web, got %q", results[0].Source)
	}
	if filepath.Base(results[0].ID) == "" {
		t.Fatalf("expected non-empty result id, got %+v", results[0])
	}
}
