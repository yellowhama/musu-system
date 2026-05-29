package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-system/marketer/internal/referral"
)

var referralCmd = &cobra.Command{
	Use:   "referral",
	Short: "Match consented high-intent leads to partner lawyers (draft, human connects)",
	Long: `Deterministically match exported leads to partner lawyers by region,
specialty, and urgency, producing a referral DRAFT for human review.

Operates ONLY on exported JSON files — never live subscriber data. Include only
consenting leads. Nothing is connected or sent automatically; the operator
reviews and connects. Writes projects/<project>/referral/referral-draft.md.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		project := viper.GetString("project")
		leadsPath, _ := cmd.Flags().GetString("leads")
		lawyersPath, _ := cmd.Flags().GetString("lawyers")
		outDir, _ := cmd.Flags().GetString("out")

		var leads []referral.Lead
		if err := readJSON(leadsPath, &leads); err != nil {
			return fmt.Errorf("read leads: %w", err)
		}
		var lawyers []referral.Lawyer
		if err := readJSON(lawyersPath, &lawyers); err != nil {
			return fmt.Errorf("read lawyers: %w", err)
		}
		if len(leads) == 0 {
			return fmt.Errorf("no leads in %s", leadsPath)
		}
		if len(lawyers) == 0 {
			return fmt.Errorf("no lawyers in %s", lawyersPath)
		}

		matches := referral.MatchLeads(leads, lawyers)
		matched := 0
		for _, m := range matches {
			if m.Lawyer != nil {
				matched++
			}
		}
		fmt.Printf("🤝 %d leads, %d lawyers → %d matched, %d need manual review\n",
			len(leads), len(lawyers), matched, len(leads)-matched)

		dir := filepath.Join("projects", project, outDir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create out dir %s: %w", dir, err)
		}
		path := filepath.Join(dir, "referral-draft.md")
		if err := os.WriteFile(path, []byte(referral.Render(matches)), 0o644); err != nil {
			return fmt.Errorf("write referral draft: %w", err)
		}
		fmt.Printf("📝 Saved: %s (review before connecting)\n", path)
		return nil
	},
}

func readJSON(path string, v any) error {
	if path == "" {
		return fmt.Errorf("path is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func init() {
	referralCmd.Flags().String("leads", "", "path to exported consenting-leads JSON (required)")
	referralCmd.Flags().String("lawyers", "", "path to partner-lawyers JSON (required)")
	referralCmd.Flags().String("out", "referral", "output subdirectory under projects/<project>/")
	_ = referralCmd.MarkFlagRequired("leads")
	_ = referralCmd.MarkFlagRequired("lawyers")
	rootCmd.AddCommand(referralCmd)
}
