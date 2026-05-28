package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-system/nurikun/internal/config"
	"github.com/yellowhama/musu-system/nurikun/internal/preflight"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check project config, mailbox settings, knowledge source, and AI connectivity",
	RunE: func(cmd *cobra.Command, args []string) error {
		project := viper.GetString("project")
		conf, err := config.Load(project)
		if err != nil {
			return err
		}

		jsonMode := viper.GetBool("json")
		if !jsonMode {
			fmt.Println("==> musu-nurikun doctor")
			fmt.Printf("Project          : %s\n", project)
			fmt.Printf("AI URL           : %s\n", conf.AIBaseURL)
			fmt.Printf("Mailbox Provider : %s\n", conf.MailboxProvider)
			fmt.Printf("Knowledge Source : %s\n", conf.KnowledgeSource)
		}
		autoFix, _ := cmd.Flags().GetBool("fix")
		fixMailboxProvider, _ := cmd.Flags().GetString("mailbox-provider")
		fixKnowledgeSource, _ := cmd.Flags().GetString("knowledge-source")

		configPath := filepath.Join("projects", project, "config.yaml")
		result := preflight.EvaluateDoctor(preflight.DoctorOptions{
			Project:    project,
			ConfigPath: configPath,
			Config:     conf,
			AutoFix:    autoFix,
			FixProject: func() (*config.Config, error) {
				_, _, err := bootstrapProject(project, !jsonMode, fixMailboxProvider, fixKnowledgeSource)
				if err != nil {
					return nil, err
				}
				return config.Load(project)
			},
		})

		if !jsonMode {
			renderDoctorResult(result, project, configPath)
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
	doctorCmd.Flags().String("mailbox-provider", "imap", "Mailbox provider preset to use with doctor --fix (imap or gmail)")
	doctorCmd.Flags().String("knowledge-source", "none", "Knowledge source preset to use with doctor --fix (none, crawlai, folder)")
	rootCmd.AddCommand(doctorCmd)
}
