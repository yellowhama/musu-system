package cmd

import (
	"fmt"
	"math/rand"
	"time"

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
		active, _ := cmd.Flags().GetBool("active")

		fmt.Printf("🚶 Citizen 'nurikun' is starting to roam: %s (Active: %v)\n", target, active)

		walker, err := browser.NewWalker(project)
		if err != nil {
			fmt.Printf("❌ Failed to start walker: %v\n", err)
			return
		}
		defer walker.Close()

		page, err := walker.Navigate(target)
		if err != nil {
			fmt.Printf("❌ Navigation failed: %v\n", err)
			return
		}
		defer page.Close()

		if active {
			fmt.Println("🧠 Starting active lurking session (Human-like behavior)...")
			for i := 0; i < 5; i++ { // Perform 5 cycles of lurking
				fmt.Printf("   [Cycle %d] Scrolling and reading...\n", i+1)
				walker.HumanScroll(page)
				
				if rand.Float64() > 0.5 {
					fmt.Println("   [Cycle %d] Investigating interactive elements...")
					walker.HumanHover(page)
				}
				
				time.Sleep(time.Duration(rand.Intn(5)+2) * time.Second)
			}
			fmt.Println("✅ Lurking session completed.")
		}

		fmt.Printf("✅ Arrived at %s. Press Enter to end roaming.\n", target)
		fmt.Scanln() 
	},
}

func init() {
	roamCmd.Flags().Bool("active", false, "Simulate active human lurking and reading")
	rootCmd.AddCommand(roamCmd)
}
