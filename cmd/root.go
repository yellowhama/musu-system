package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const Version = "v0.3.0"

var rootCmd = &cobra.Command{
	Use:   "musu-nurikun",
	Short: "Autonomous email agent — opt-in mailing lists & inbound support",
	Long: `musu-nurikun is the "Hand" of the Musu ecosystem: an autonomous email agent.

It handles inbound customer support (triage, grounded replies, escalation) and
compliant outbound campaigns to opt-in subscribers (self-subscribe + double
opt-in, controlled cadence, one-click unsubscribe). It emails only people who
subscribed themselves — no cold outreach, no fake identities, no anti-detection.`,
	Version: Version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringP("project", "p", "default", "Project name to scope mailboxes, lists, and contacts")
	rootCmd.PersistentFlags().String("ai-url", "http://localhost:11434/v1", "OpenAI-compatible AI base URL")
	viper.BindPFlag("project", rootCmd.PersistentFlags().Lookup("project"))
	viper.BindPFlag("ai_url", rootCmd.PersistentFlags().Lookup("ai-url"))
}

func initConfig() {
	viper.AutomaticEnv()
}
