package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestBootstrapProjectCreatesArtifacts(t *testing.T) {
	wd, _ := os.Getwd()
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	viper.Set("wiki_dir", "F:/wiki-fixture")
	viper.Set("ai_provider", "openai")
	viper.Set("ai_url", "http://127.0.0.1:9999/v1")

	baseDir, dbPath, err := bootstrapProject("alpha", false)
	if err != nil {
		t.Fatalf("bootstrapProject failed: %v", err)
	}

	assertExists := func(path string) {
		t.Helper()
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}

	assertExists(baseDir)
	assertExists(dbPath)
	assertExists(filepath.Join(baseDir, "config.yaml"))
	assertExists(filepath.Join(baseDir, "personas", "default.md"))
	assertExists(filepath.Join(baseDir, "NEXT_STEPS.md"))

	configBytes, err := os.ReadFile(filepath.Join(baseDir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	configText := string(configBytes)
	for _, want := range []string{
		"db_path: " + dbPath,
		"wiki_dir: F:/wiki-fixture",
		"ai_provider: openai",
		"ai_url: http://127.0.0.1:9999/v1",
	} {
		if !strings.Contains(configText, want) {
			t.Fatalf("expected config to contain %q, got:\n%s", want, configText)
		}
	}

	nextStepsBytes, err := os.ReadFile(filepath.Join(baseDir, "NEXT_STEPS.md"))
	if err != nil {
		t.Fatal(err)
	}
	nextSteps := string(nextStepsBytes)
	if !strings.Contains(nextSteps, "musu-marketer doctor --project alpha") {
		t.Fatalf("expected NEXT_STEPS to mention doctor command, got:\n%s", nextSteps)
	}
	if !strings.Contains(nextSteps, "musu-marketer draft <topic> --project alpha --persona default") {
		t.Fatalf("expected NEXT_STEPS to mention draft command, got:\n%s", nextSteps)
	}
}
