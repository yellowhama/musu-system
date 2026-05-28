package knowledge

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFixture(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", name, err)
	}
}

func TestFolderSourceRetrieve(t *testing.T) {
	dir := t.TempDir()

	writeFixture(t, dir, "billing.md", "# Billing FAQ\n\nHow do I get a refund?\nRefunds are processed within 7 days. Contact billing for a refund.\n\nUnrelated note about shipping.\n")
	writeFixture(t, dir, "shipping.txt", "Shipping takes 3-5 business days. We ship worldwide.\n")
	writeFixture(t, dir, "account.md", "# Account\n\nReset your password from the account settings page.\n")
	// Non-matching extension must be ignored.
	writeFixture(t, dir, "ignore.json", `{"refund":"refund refund refund"}`)

	src, err := newFolderSource(Settings{Dir: dir})
	if err != nil {
		t.Fatalf("newFolderSource: %v", err)
	}
	if src.Name() != "folder" {
		t.Fatalf("Name() = %q, want folder", src.Name())
	}

	tests := []struct {
		name      string
		query     string
		limit     int
		wantTop   string // expected basename of top result, "" means expect no results
		wantCount int
	}{
		{
			name:      "refund query ranks billing first",
			query:     "refund",
			limit:     5,
			wantTop:   "billing.md",
			wantCount: 1,
		},
		{
			name:      "shipping query ranks shipping doc",
			query:     "shipping worldwide",
			limit:     5,
			wantTop:   "shipping.txt",
			wantCount: 2, // billing.md also mentions shipping once
		},
		{
			name:      "limit caps results",
			query:     "the a",
			limit:     1,
			wantCount: 1,
		},
		{
			name:      "no overlap returns nothing",
			query:     "quantum entanglement",
			limit:     5,
			wantTop:   "",
			wantCount: 0,
		},
		{
			name:      "empty query returns nothing",
			query:     "",
			limit:     5,
			wantTop:   "",
			wantCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := src.Retrieve(tc.query, tc.limit)
			if err != nil {
				t.Fatalf("Retrieve: %v", err)
			}
			if len(got) != tc.wantCount {
				t.Fatalf("got %d snippets, want %d: %+v", len(got), tc.wantCount, got)
			}
			if tc.wantTop != "" {
				if base := filepath.Base(got[0].Source); base != tc.wantTop {
					t.Fatalf("top result = %q, want %q", base, tc.wantTop)
				}
				if got[0].Score <= 0 {
					t.Fatalf("top score = %v, want > 0", got[0].Score)
				}
				if got[0].Content == "" {
					t.Fatalf("top content is empty")
				}
			}
			// Scores must be non-increasing.
			for i := 1; i < len(got); i++ {
				if got[i].Score > got[i-1].Score {
					t.Fatalf("scores not sorted descending: %v before %v", got[i-1].Score, got[i].Score)
				}
			}
		})
	}
}

func TestFolderTitleFromHeading(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "doc.md", "# My Heading\n\nbody text refund here\n")
	writeFixture(t, dir, "plain.txt", "no heading here just text refund\n")

	src, err := newFolderSource(Settings{Dir: dir})
	if err != nil {
		t.Fatalf("newFolderSource: %v", err)
	}
	got, err := src.Retrieve("refund", 5)
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	titles := map[string]string{}
	for _, s := range got {
		titles[filepath.Base(s.Source)] = s.Title
	}
	if titles["doc.md"] != "My Heading" {
		t.Fatalf("doc.md title = %q, want %q", titles["doc.md"], "My Heading")
	}
	if titles["plain.txt"] != "plain" {
		t.Fatalf("plain.txt title = %q, want %q", titles["plain.txt"], "plain")
	}
}

func TestNewFolderSourceErrors(t *testing.T) {
	if _, err := newFolderSource(Settings{Dir: ""}); err == nil {
		t.Fatal("expected error for empty Dir")
	}
	if _, err := newFolderSource(Settings{Dir: filepath.Join(t.TempDir(), "does-not-exist")}); err == nil {
		t.Fatal("expected error for missing Dir")
	}
}

func TestNewCrawlAISourceRequiresPath(t *testing.T) {
	if _, err := newCrawlAISource(Settings{CrawlPath: ""}); err == nil {
		t.Fatal("expected error for empty CrawlPath")
	}
	src, err := newCrawlAISource(Settings{CrawlPath: "musu-crawl-ai", Project: "demo"})
	if err != nil {
		t.Fatalf("newCrawlAISource: %v", err)
	}
	if src.Name() != "crawlai" {
		t.Fatalf("Name() = %q, want crawlai", src.Name())
	}
}
