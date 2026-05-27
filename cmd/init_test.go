package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestBootstrapProjectCreatesSetupArtifactsForFolderKnowledge(t *testing.T) {
	wd, _ := os.Getwd()
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	viper.Set("ai_url", "http://127.0.0.1:9999/v1")

	baseDir, configPath, err := bootstrapProject("alpha", false, "gmail", "folder")
	if err != nil {
		t.Fatalf("bootstrapProject failed: %v", err)
	}

	dbPath := filepath.Join(baseDir, "data", "nurikun.db")
	setupPath := filepath.Join(baseDir, "SETUP.md")
	knowledgeReadmePath := filepath.Join(baseDir, "knowledge", "README.md")

	for _, path := range []string{baseDir, configPath, dbPath, setupPath, knowledgeReadmePath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	configText := string(configBytes)
	for _, want := range []string{
		"mailbox_provider: gmail",
		"knowledge_source: folder",
		`knowledge_dir: "./projects/alpha/knowledge"`,
		"ai_url: http://127.0.0.1:9999/v1",
	} {
		if !strings.Contains(configText, want) {
			t.Fatalf("expected config to contain %q, got:\n%s", want, configText)
		}
	}

	setupBytes, err := os.ReadFile(setupPath)
	if err != nil {
		t.Fatal(err)
	}
	setupText := string(setupBytes)
	if !strings.Contains(setupText, "public_base_url") || !strings.Contains(setupText, "unsub_secret") {
		t.Fatalf("expected SETUP.md to mention public delivery settings, got:\n%s", setupText)
	}

	knowledgeBytes, err := os.ReadFile(knowledgeReadmePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(knowledgeBytes), "FAQ.md") {
		t.Fatalf("expected knowledge README to suggest seed documents, got:\n%s", string(knowledgeBytes))
	}
}
