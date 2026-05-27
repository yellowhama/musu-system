package preflight

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yellowhama/musu-nurikun/internal/config"
)

func TestEvaluateDoctorBlocksOnFixFailure(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "projects", "missing", "config.yaml")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[]}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	conf := &config.Config{
		AIBaseURL:       srv.URL,
		MailboxProvider: "imap",
		IMAPHost:        "imap.example.com",
		IMAPUser:        "user",
		IMAPPass:        "pass",
		SMTPHost:        "smtp.example.com",
		SMTPFrom:        "sender@example.com",
		KnowledgeSource: "none",
	}

	result := EvaluateDoctor(DoctorOptions{
		Project:    "missing",
		ConfigPath: configPath,
		Config:     conf,
		AutoFix:    true,
		FixProject: func() (*config.Config, error) { return nil, fmt.Errorf("boom") },
	})

	if !result.Blocking {
		t.Fatalf("expected blocking result when FixProject fails")
	}
	if result.Report.ConfigFixError == "" {
		t.Fatalf("expected config_fix_error to be recorded")
	}
}

func TestEvaluateDoctorBlocksWithoutPublicBaseURLAndUnsubSecret(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[]}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	conf := &config.Config{
		AIBaseURL:       srv.URL,
		MailboxProvider: "imap",
		IMAPHost:        "imap.example.com",
		IMAPUser:        "user",
		IMAPPass:        "pass",
		SMTPHost:        "smtp.example.com",
		SMTPFrom:        "sender@example.com",
		KnowledgeSource: "none",
	}

	result := EvaluateDoctor(DoctorOptions{
		Project:    "default",
		ConfigPath: filepath.Join(t.TempDir(), "config.yaml"),
		Config:     conf,
	})

	if !result.Blocking {
		t.Fatalf("expected blocking result when public_base_url and unsub_secret are missing")
	}
	if !strings.Contains(result.ActionableFix, "public_base_url") {
		t.Fatalf("expected actionable fix to mention public_base_url, got %q", result.ActionableFix)
	}
	if !strings.Contains(result.ActionableFix, "unsub_secret") {
		t.Fatalf("expected actionable fix to mention unsub_secret, got %q", result.ActionableFix)
	}
}
