package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/yellowhama/musu-crawl-ai/internal/processor"
)

func TestSearchCommandJSONHappyPath(t *testing.T) {
	tmp := t.TempDir()
	proc := processor.NewWikiProcessor(tmp, "demo")
	if _, err := proc.SaveToWiki("web", "https://example.com/scheduler", "Scheduler Deep Dive", "Scheduler operations require reliability and timing.", []string{"scheduler", "ops"}, "Grounded scheduler notes", 0.9); err != nil {
		t.Fatalf("SaveToWiki failed: %v", err)
	}
	if err := proc.UpdateIndex(); err != nil {
		t.Fatalf("UpdateIndex failed: %v", err)
	}

	if err := searchCmd.Flags().Set("out", tmp); err != nil {
		t.Fatal(err)
	}
	if err := searchCmd.Flags().Set("project", "demo"); err != nil {
		t.Fatal(err)
	}
	if err := searchCmd.Flags().Set("semantic", "false"); err != nil {
		t.Fatal(err)
	}
	if err := searchCmd.Flags().Set("json", "true"); err != nil {
		t.Fatal(err)
	}
	viper.Set("json", true)

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	searchCmd.Run(searchCmd, []string{"scheduler"})

	_ = w.Close()
	os.Stdout = oldStdout
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}

	text := string(out)
	for _, want := range []string{
		`"status": "success"`,
		`"message": "Search completed"`,
		`"title": "Scheduler Deep Dive"`,
		`"project": "demo"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, text)
		}
	}

	blevePath := filepath.Join(tmp, "musu.bleve")
	if _, err := os.Stat(blevePath); err != nil {
		t.Fatalf("expected bleve index to exist at %s: %v", blevePath, err)
	}
}
