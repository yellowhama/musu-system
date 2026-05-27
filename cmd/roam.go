package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-nurikun/internal/browser"
)

var roamCmd = &cobra.Command{
	Use:   "roam [url]",
	Short: "Start a stealth browsing session (Warm-up)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		project := viper.GetString("project")
		target := args[0]

		fmt.Printf("🚶 Citizen 'nurikun' is starting to roam: %s\n", target)

		walker, err := browser.NewWalker(project)
		if err != nil {
			fmt.Printf("❌ Failed to start walker: %v\n", err)
			return
		}
		defer walker.Close()

		_, err = walker.Navigate(target)
		if err != nil {
			fmt.Printf("❌ Navigation failed: %v\n", err)
			return
		}

		fmt.Printf("✅ Arrived at %s. Press Enter to end roaming.\n", target)
		fmt.Scanln() 
	},
}

func init() {
	rootCmd.AddCommand(roamCmd)
}
