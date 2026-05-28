//go:build integration

package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/yellowhama/musu-marketer/internal/db"
)

func TestDraftCommandRealAIIntegration(t *testing.T) {
	aiURL := strings.TrimSpace(os.Getenv("MUSU_MARKETER_INTEGRATION_AI_URL"))
	if aiURL == "" {
		t.Skip("set MUSU_MARKETER_INTEGRATION_AI_URL to run real integration")
	}
	model := strings.TrimSpace(os.Getenv("MUSU_MARKETER_INTEGRATION_MODEL"))
	if model == "" {
		model = "llama3"
	}

	wd, _ := os.Getwd()
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	_, dbPath, srv := setupDraftFixture(t, "integration-demo")
	srv.Close()
	viper.Set("ai_url", aiURL)
	viper.Set("project", "integration-demo")
	viper.Set("json", false)

	if err := draftCmd.Flags().Set("model", model); err != nil {
		t.Fatal(err)
	}
	if err := draftCmd.Flags().Set("persona", "default"); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		draftCmd.Run(draftCmd, []string{"scheduler"})
	})
	if !strings.Contains(out, "FINAL STRATEGIC CAMPAIGN") {
		t.Fatalf("expected final campaign output, got:\n%s", out)
	}

	store, err := db.NewStore(dbPath)
	if err != nil {
		t.Fatalf("reopen db: %v", err)
	}
	defer store.Close()

	campaigns, err := store.ListCampaigns()
	if err != nil {
		t.Fatalf("ListCampaigns failed: %v", err)
	}
	if len(campaigns) != 1 {
		t.Fatalf("expected 1 campaign after real integration draft, got %d", len(campaigns))
	}
	if campaigns[0].Topic != "scheduler" {
		t.Fatalf("expected topic scheduler, got %q", campaigns[0].Topic)
	}
}
