package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBootstrapProjectDirsCreatesProjectArtifacts(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "wiki")

	if err := bootstrapProjectDirs(out, "alpha", "openai", "http://127.0.0.1:9999/v1", false); err != nil {
		t.Fatalf("bootstrapProjectDirs failed: %v", err)
	}

	projectDir := filepath.Join(out, "projects", "alpha")
	configPath := filepath.Join(projectDir, "config.toml")
	promptPath := filepath.Join(projectDir, "PROMPT.md")
	nextStepsPath := filepath.Join(projectDir, "NEXT_STEPS.md")

	for _, path := range []string{projectDir, configPath, promptPath, nextStepsPath} {
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
		`out = "` + out + `"`,
		`ai_provider = "openai"`,
		`ai_url = "http://127.0.0.1:9999/v1"`,
	} {
		if !strings.Contains(configText, want) {
			t.Fatalf("expected config to contain %q, got:\n%s", want, configText)
		}
	}

	promptBytes, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(promptBytes), "# Research Prompt: alpha") {
		t.Fatalf("expected PROMPT.md to mention project name, got:\n%s", string(promptBytes))
	}

	nextStepsBytes, err := os.ReadFile(nextStepsPath)
	if err != nil {
		t.Fatal(err)
	}
	nextSteps := string(nextStepsBytes)
	if !strings.Contains(nextSteps, "musu-crawl doctor --out "+out+" --project alpha") {
		t.Fatalf("expected NEXT_STEPS to mention doctor command, got:\n%s", nextSteps)
	}
	if !strings.Contains(nextSteps, "musu-crawl fetch <source> <id> --project alpha") {
		t.Fatalf("expected NEXT_STEPS to mention fetch command, got:\n%s", nextSteps)
	}
}
