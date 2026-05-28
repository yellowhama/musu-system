package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-crawl-ai/internal/preflight"
	"github.com/yellowhama/musu-crawl-ai/internal/utils"
)

var doctorCapabilitySources []string

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check wiki, index, project config, and AI connectivity",
	RunE: func(cmd *cobra.Command, args []string) error {
		out, _ := cmd.Flags().GetString("out")
		project, _ := cmd.Flags().GetString("project")
		conf, err := utils.LoadConfig(project)
		if err != nil {
			return err
		}
		if out == "" {
			out = conf.WikiDir
		}

		jsonMode := viper.GetBool("json")
		if !jsonMode {
			fmt.Println("==> musu-crawl-ai doctor")
			fmt.Printf("Wiki Dir     : %s\n", out)
			fmt.Printf("Project      : %s\n", project)
			fmt.Printf("AI Provider  : %s\n", conf.AIProvider)
			fmt.Printf("AI URL       : %s\n", conf.AIBaseURL)
		}
		autoFix, _ := cmd.Flags().GetBool("fix")
		result := preflight.EvaluateDoctor(preflight.DoctorOptions{
			Out:               out,
			Project:           project,
			AIProvider:        conf.AIProvider,
			AIURL:             conf.AIBaseURL,
			CapabilitySources: doctorCapabilitySources,
			AutoFix:           autoFix,
			FixScaffold: func() error {
				return bootstrapProjectDirs(out, projectName(project), conf.AIProvider, conf.AIBaseURL, !jsonMode)
			},
		})

		if !jsonMode {
			if result.Report.WikiExists {
				fmt.Printf("✅ Wiki directory exists (%d markdown files)\n", result.Report.WikiMarkdownCount)
			} else {
				fmt.Printf("❌ Wiki directory missing: %s\n", out)
				fmt.Println("   Fix: run 'musu-crawl init --out <dir>' first, or use doctor --fix.")
			}
			if result.Report.IndexExists {
				fmt.Println("✅ Search index exists")
			} else {
				fmt.Printf("⚠️  Search index missing: %s\n", filepath.Join(out, "musu.bleve"))
				fmt.Println("   Run: musu-crawl index --out", out)
			}
			if result.Report.ProjectExists {
				fmt.Println("✅ Project directory exists")
			}
			if result.Report.AIReachable {
				fmt.Println("✅ AI endpoint reachable")
			} else if result.Report.AIError != "" {
				fmt.Printf("❌ AI endpoint probe failed: %v\n", result.Report.AIError)
			}
		}

		if result.Blocking {
			if jsonMode {
				utils.PrintJSONError("doctor found blocking issues", result.Report, result.ActionableFix)
			}
			return fmt.Errorf("doctor found blocking issues")
		}
		if !jsonMode {
			fmt.Println("✅ Doctor passed")
		}
		utils.PrintJSON("Doctor passed", result.Report)
		return nil
	},
}

func init() {
	doctorCmd.Flags().String("out", "./wiki", "Wiki directory to inspect")
	doctorCmd.Flags().StringP("project", "p", "default", "Project to inspect")
	doctorCmd.Flags().Bool("fix", false, "Auto-create missing local wiki scaffold when safe")
	doctorCmd.Flags().StringSliceVar(&doctorCapabilitySources, "capability-source", nil, "Optional source(s) to describe capability metadata for (web, yt, gh, arxiv, reddit, hf, x)")
	rootCmd.AddCommand(doctorCmd)
}

func projectName(project string) string {
	if strings.TrimSpace(project) == "" || project == "all" {
		return "default"
	}
	return project
}
