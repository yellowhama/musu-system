package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const Version = "v2.0.4"

var rootCmd = &cobra.Command{
	Use:     "musu-marketer",
	Short:   "AI Influencer & Marketing Automation Engine",
	Long:    `Transforms verified knowledge from musu-crawl-ai into viral social media content.`,
	Version: Version,
	SilenceUsage: true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if cmd.Name() != "update" && cmd.Name() != "help" {
			go checkNewVersion()
		}
	},
}

func checkNewVersion() {
	latest, _, err := GetLatestRelease("yellowhama", "musu-marketer")
	if err == nil && latest != Version {
		fmt.Fprintf(os.Stderr, "\n💡 New version available: %s (Current: %s)\n", latest, Version)
		fmt.Fprintf(os.Stderr, "👉 Run 'musu-marketer update' to upgrade.\n\n")
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if viper.GetBool("json") {
			if err.Error() == "doctor found blocking issues" {
				os.Exit(1)
			}
			fix := ""
			if strings.Contains(err.Error(), "arg(s)") {
				fix = "Check 'musu-marketer [command] --help' for argument requirements."
			}
			if fix == "" {
				fix = "Check 'musu-marketer [command] --help' for command usage and required flags."
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
	rootCmd.PersistentFlags().String("wiki", "", "Path to musu-crawl-ai wiki (auto-discovered when omitted)")
	rootCmd.PersistentFlags().StringP("project", "p", "default", "Project name to scope the assets")
	rootCmd.PersistentFlags().String("ai-provider", "ollama", "AI provider (ollama, sglang, openai)")
	rootCmd.PersistentFlags().String("ai-url", "http://localhost:11434/v1", "AI base URL")
	rootCmd.PersistentFlags().Bool("json", false, "Output in machine-readable JSON format")
	
	viper.BindPFlag("wiki_dir", rootCmd.PersistentFlags().Lookup("wiki"))
	viper.BindPFlag("project", rootCmd.PersistentFlags().Lookup("project"))
	viper.BindPFlag("ai_provider", rootCmd.PersistentFlags().Lookup("ai-provider"))
	viper.BindPFlag("ai_url", rootCmd.PersistentFlags().Lookup("ai-url"))
	viper.BindPFlag("json", rootCmd.PersistentFlags().Lookup("json"))
}

func initConfig() {
	viper.AutomaticEnv()
	if strings.TrimSpace(viper.GetString("wiki_dir")) == "" {
		viper.Set("wiki_dir", discoverWikiDir())
	}
}

func discoverWikiDir() string {
	if env := strings.TrimSpace(os.Getenv("MUSU_MARKETER_WIKI")); env != "" {
		return env
	}
	if env := strings.TrimSpace(os.Getenv("MUSU_WIKI")); env != "" {
		return env
	}

	candidates := []string{}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, "wiki"),
			filepath.Join(wd, "..", "musu-crawl-ai", "wiki"),
		)
	}
	if exe, err := os.Executable(); err == nil {
		base := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(base, "wiki"),
			filepath.Join(base, "..", "musu-crawl-ai", "wiki"),
		)
	}
	candidates = append(candidates, `C:\Users\empty\musu-crawl-ai\wiki`)

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	if len(candidates) > 0 {
		return candidates[0]
	}
	return "./wiki"
}
