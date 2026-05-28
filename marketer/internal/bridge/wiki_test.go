package bridge

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindByTopicUsesIndexAndBodyContent(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.json"), []byte(`[
  {
    "id":"go-1-22",
    "title":"Go Release Notes",
    "source":"web",
    "project":"default",
    "path":"projects/default/web/go_release.md",
    "summary":"Scheduler and loop fixes",
    "tags":["golang","release"]
  }
]`), 0644); err != nil {
		t.Fatal(err)
	}

	docPath := filepath.Join(root, "projects", "default", "web")
	if err := os.MkdirAll(docPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docPath, "go_release.md"), []byte("Detailed Go 1.22 notes with scheduler improvements."), 0644); err != nil {
		t.Fatal(err)
	}

	b := NewWikiBridge(root)
	results, err := b.FindByTopic("scheduler")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Title != "Go Release Notes" {
		t.Fatalf("unexpected title: %s", results[0].Title)
	}
}

func TestFindByTopicRanksStructuredMatchesAheadOfBodyNoise(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.json"), []byte(`[
  {
    "id":"title-hit",
    "title":"Scheduler Reliability Playbook",
    "source":"web",
    "project":"default",
    "path":"projects/default/web/scheduler_playbook.md",
    "summary":"Operator trust and timing guardrails",
    "tags":["scheduler","reliability"]
  },
  {
    "id":"body-hit",
    "title":"General Operations Notes",
    "source":"web",
    "project":"default",
    "path":"projects/default/web/ops_notes.md",
    "summary":"Broad operator notes",
    "tags":["ops"]
  }
]`), 0644); err != nil {
		t.Fatal(err)
	}

	docPath := filepath.Join(root, "projects", "default", "web")
	if err := os.MkdirAll(docPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docPath, "scheduler_playbook.md"), []byte("A focused guide for scheduler reliability and incident prevention."), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docPath, "ops_notes.md"), []byte("This file mentions scheduler reliability once in passing but is mostly unrelated."), 0644); err != nil {
		t.Fatal(err)
	}

	b := NewWikiBridge(root)
	results, err := b.FindByTopic("scheduler reliability")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Title != "Scheduler Reliability Playbook" {
		t.Fatalf("expected ranked title match first, got %q", results[0].Title)
	}
	if results[1].Title != "General Operations Notes" {
		t.Fatalf("expected weaker body match second, got %q", results[1].Title)
	}
}
