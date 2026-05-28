package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-marketer/internal/preflight"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check wiki, project, and AI connectivity before drafting campaigns",
	RunE: func(cmd *cobra.Command, args []string) error {
		project := viper.GetString("project")
		wikiDir := viper.GetString("wiki_dir")
		aiProvider := viper.GetString("ai_provider")
		aiURL := viper.GetString("ai_url")
		topic, _ := cmd.Flags().GetString("topic")

		jsonMode := viper.GetBool("json")
		if !jsonMode {
			fmt.Println("==> musu-marketer doctor")
			fmt.Printf("Project      : %s\n", project)
			fmt.Printf("Wiki         : %s\n", wikiDir)
			fmt.Printf("AI Provider  : %s\n", aiProvider)
			fmt.Printf("AI URL       : %s\n", aiURL)
		}
		autoFix, _ := cmd.Flags().GetBool("fix")
		result := preflight.EvaluateDoctor(preflight.DoctorOptions{
			Project:    project,
			WikiDir:    wikiDir,
			AIProvider: aiProvider,
			AIURL:      aiURL,
			Topic:      topic,
			AutoFix:    autoFix,
			FixProject: func() error {
				_, _, err := bootstrapProject(project, !jsonMode)
				return err
			},
		})

		if !jsonMode {
			renderDoctorResult(result, wikiDir, topic)
		}

		if result.Blocking {
			err := fmt.Errorf("doctor found blocking issues")
			printJSONError(err, result.Report, result.ActionableFix)
			return err
		}
		if !jsonMode {
			fmt.Println("✅ Doctor passed")
		}
		printJSONSuccess("Doctor passed", result.Report)
		return nil
	},
}

func init() {
	doctorCmd.Flags().Bool("fix", false, "Auto-create missing local project scaffold when safe")
	doctorCmd.Flags().String("topic", "", "Optional topic to validate draft-readiness against the wiki")
	rootCmd.AddCommand(doctorCmd)
}
