package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadUsesProjectDotEnvOverrides(t *testing.T) {
	wd, _ := os.Getwd()
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	projectDir := filepath.Join("projects", "alpha")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	configText := `ai_url: http://config.example/v1
mailbox_provider: imap
imap_host: config-imap.example.com
knowledge_source: none
public_base_url: https://config.example
unsub_secret: config-secret
`
	if err := os.WriteFile(filepath.Join(projectDir, "config.yaml"), []byte(configText), 0o644); err != nil {
		t.Fatal(err)
	}

	envText := `NURIKUN_AI_URL=http://env.example/v1
NURIKUN_IMAP_HOST=env-imap.example.com
NURIKUN_PUBLIC_BASE_URL=https://env.example
`
	if err := os.WriteFile(filepath.Join(projectDir, ".env"), []byte(envText), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load("alpha")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.AIBaseURL != "http://env.example/v1" {
		t.Fatalf("expected AIBaseURL from .env, got %q", cfg.AIBaseURL)
	}
	if cfg.IMAPHost != "env-imap.example.com" {
		t.Fatalf("expected IMAPHost from .env, got %q", cfg.IMAPHost)
	}
	if cfg.PublicBaseURL != "https://env.example" {
		t.Fatalf("expected PublicBaseURL from .env, got %q", cfg.PublicBaseURL)
	}
	if cfg.UnsubSecret != "config-secret" {
		t.Fatalf("expected unsub_secret to fall back to config.yaml, got %q", cfg.UnsubSecret)
	}
}
