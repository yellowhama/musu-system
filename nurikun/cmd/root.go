package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const Version = "v0.3.1"

var rootCmd = &cobra.Command{
	Use:   "musu-nurikun",
	Short: "Autonomous email agent — opt-in mailing lists & inbound support",
	Long: `musu-nurikun is the "Hand" of the Musu ecosystem: an autonomous email agent.

It handles inbound customer support (triage, grounded replies, escalation) and
compliant outbound campaigns to opt-in subscribers (self-subscribe + double
opt-in, controlled cadence, one-click unsubscribe). It emails only people who
subscribed themselves — no cold outreach, no fake identities, no anti-detection.`,
	Version: Version,
	SilenceUsage: true,
	SilenceErrors: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if viper.GetBool("json") {
			if err.Error() == "doctor found blocking issues" {
				os.Exit(1)
			}
			fix := ""
			if strings.Contains(err.Error(), "arg(s)") {
				fix = "Check 'musu-nurikun [command] --help' for argument requirements."
			}
			if fix == "" {
				fix = "Check 'musu-nurikun [command] --help' for command usage and required flags."
			}
			printJSONError(err, nil, fix)
			os.Exit(1)
		}
		if !viper.GetBool("json") {
			fmt.Println(err)
		}
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringP("project", "p", "default", "Project name to scope mailboxes, lists, and contacts")
	rootCmd.PersistentFlags().String("ai-url", "http://localhost:11434/v1", "OpenAI-compatible AI base URL")
	rootCmd.PersistentFlags().Bool("json", false, "Output in machine-readable JSON format")
	viper.BindPFlag("project", rootCmd.PersistentFlags().Lookup("project"))
	viper.BindPFlag("ai_url", rootCmd.PersistentFlags().Lookup("ai-url"))
	viper.BindPFlag("json", rootCmd.PersistentFlags().Lookup("json"))
}

func initConfig() {
	viper.AutomaticEnv()
}
