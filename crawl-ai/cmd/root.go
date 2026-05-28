package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-crawl-ai/internal/utils"
)

const Version = "v0.8.0"

var rootCmd = &cobra.Command{
	Use:           "musu-crawl-ai",
	Short:         "AI-Ready Knowledge Harvester & Wiki Generator",
	Long:          `A high-performance crawler to fetch and organize content for AI.`,
	Version:       Version,
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("json", cmd.Root().PersistentFlags().Lookup("json"))
		
		// Silent background check for updates
		if cmd.Name() != "update" && cmd.Name() != "help" {
			go checkNewVersion()
		}
	},
}

func checkNewVersion() {
	latest, _, err := GetLatestRelease("yellowhama", "musu-crawl-ai")
	if err == nil && latest != Version {
		fmt.Fprintf(os.Stderr, "\n💡 New version available: %s (Current: %s)\n", latest, Version)
		fmt.Fprintf(os.Stderr, "👉 Run 'musu-crawl update' to upgrade.\n\n")
	}
}

func Execute() error {
	err := rootCmd.Execute()
	if err != nil {
		if viper.GetBool("json") && err.Error() == "doctor found blocking issues" {
			os.Exit(1)
		}
		fix := ""
		if strings.Contains(err.Error(), "arg(s)") {
			fix = "Check 'musu-crawl [command] --help' for argument requirements."
		}
		if fix == "" {
			fix = "Check 'musu-crawl [command] --help' for command usage and required flags."
		}
		utils.PrintError(err, fix)
		os.Exit(1)
	}
	return nil
}

func init() {
	rootCmd.PersistentFlags().Bool("json", false, "Output in machine-readable JSON format")
	rootCmd.PersistentFlags().String("ai-provider", "ollama", "AI provider (ollama, sglang, openai)")
	rootCmd.PersistentFlags().String("ai-url", "http://localhost:11434/v1", "AI base URL")
	
	viper.BindPFlag("json", rootCmd.PersistentFlags().Lookup("json"))
	viper.BindPFlag("ai_provider", rootCmd.PersistentFlags().Lookup("ai-provider"))
	viper.BindPFlag("ai_url", rootCmd.PersistentFlags().Lookup("ai-url"))
}
