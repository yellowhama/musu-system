package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "musu-nurikun",
	Short: "Autonomous Digital Citizen Agent",
	Long:  `Engineered to acquire digital identities, register on platforms, and interact autonomously.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringP("project", "p", "default", "Project name to scope the identities")
	viper.BindPFlag("project", rootCmd.PersistentFlags().Lookup("project"))
}

func initConfig() {
	viper.AutomaticEnv()
}
