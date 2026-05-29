package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-system/marketer/internal/kakao"
)

var notifyCmd = &cobra.Command{
	Use:   "notify",
	Short: "Draft Kakao 알림톡 (informational) templates for registration (does NOT send)",
	Long: `Generate registration-ready Kakao AlimTalk 정보성 template drafts and run a
deterministic compliance check (정보성 only, ≤1000 chars, declared variables,
no promotional markers).

This is the content layer of the Kakao channel. It does NOT send anything:
templates must be pre-approved by Kakao, and delivery to opt-in subscribers is
handled by nurikun (v0.4.0) human-in-the-loop. Promotional (친구톡) messaging
needs separate marketing consent and is out of scope.

Writes projects/<project>/kakao/alimtalk-templates.md.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		project := viper.GetString("project")
		outDir, _ := cmd.Flags().GetString("out")

		templates := kakao.DefaultTemplates
		issues := 0
		for _, t := range templates {
			issues += len(kakao.Validate(t))
		}
		fmt.Printf("📨 %d 정보성 알림톡 템플릿, 컴플라이언스 이슈 %d건\n", len(templates), issues)

		dir := filepath.Join("projects", project, outDir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create out dir %s: %w", dir, err)
		}
		path := filepath.Join(dir, "alimtalk-templates.md")
		if err := os.WriteFile(path, []byte(kakao.Render(templates)), 0o644); err != nil {
			return fmt.Errorf("write templates: %w", err)
		}
		fmt.Printf("📝 Saved: %s (카카오 비즈메시지 등록·심사 후 사용)\n", path)
		if issues > 0 {
			return fmt.Errorf("%d compliance issue(s) — fix before registering", issues)
		}
		return nil
	},
}

func init() {
	notifyCmd.Flags().String("out", "kakao", "output subdirectory under projects/<project>/")
	rootCmd.AddCommand(notifyCmd)
}
