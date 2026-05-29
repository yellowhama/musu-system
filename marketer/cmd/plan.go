package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-system/marketer/internal/funnel"
)

var planCmd = &cobra.Command{
	Use:   "plan [keyword...]",
	Short: "Build a deterministic acquisition-funnel plan toward a member target",
	Long: `Turn a member target (e.g. 10,000) into transparent funnel math + a
coordinated campaign calendar over the seo/pr/opt-in/nurture tools.

Pure and deterministic (no LLM): the numbers are assumptions printed as
assumptions, to be re-computed against real GSC/GA analytics. Writes
projects/<project>/plan/campaign-plan.md.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project := viper.GetString("project")
		cfg := funnel.DefaultConfig()
		if t, _ := cmd.Flags().GetInt("target"); t > 0 {
			cfg.TargetMembers = t
		}
		if r, _ := cmd.Flags().GetFloat64("optin-rate"); r > 0 {
			cfg.OptInRate = r
		}
		if r, _ := cmd.Flags().GetFloat64("member-rate"); r > 0 {
			cfg.MemberRate = r
		}
		if v, _ := cmd.Flags().GetInt("visits-per-article"); v > 0 {
			cfg.VisitsPerArticleMo = v
		}
		if h, _ := cmd.Flags().GetInt("horizon-months"); h > 0 {
			cfg.HorizonMonths = h
		}

		plan := funnel.BuildPlan(project, args, cfg)
		m := plan.Math
		fmt.Printf("🎯 target=%d members → %s opt-ins → %s visitors (%s/mo) → ~%d articles\n",
			cfg.TargetMembers, commaThousands(m.OptInsNeeded), commaThousands(m.VisitorsNeeded),
			commaThousands(m.MonthlyVisits), m.ArticlesNeeded)

		outDir, _ := cmd.Flags().GetString("out")
		dir := filepath.Join("projects", project, outDir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create out dir %s: %w", dir, err)
		}
		path := filepath.Join(dir, "campaign-plan.md")
		if err := os.WriteFile(path, []byte(plan.Render()), 0o644); err != nil {
			return fmt.Errorf("write plan: %w", err)
		}
		fmt.Printf("📝 Saved: %s (%d scheduled actions)\n", path, len(plan.Actions))
		return nil
	},
}

// commaThousands is a tiny formatter for the CLI summary line.
func commaThousands(n int) string {
	s := fmt.Sprintf("%d", n)
	var out []byte
	for i := 0; i < len(s); i++ {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, s[i])
	}
	return string(out)
}

func init() {
	planCmd.Flags().Int("target", 10000, "member acquisition target")
	planCmd.Flags().Float64("optin-rate", 0, "visitor→opt-in rate (default 0.03)")
	planCmd.Flags().Float64("member-rate", 0, "opt-in→member rate (default 0.5)")
	planCmd.Flags().Int("visits-per-article", 0, "sustained monthly visits per article (default 150)")
	planCmd.Flags().Int("horizon-months", 0, "campaign horizon in months (default 12)")
	planCmd.Flags().String("out", "plan", "output subdirectory under projects/<project>/")
	rootCmd.AddCommand(planCmd)
}
