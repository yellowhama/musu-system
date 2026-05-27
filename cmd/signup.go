package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-nurikun/internal/agent"
	"github.com/yellowhama/musu-nurikun/internal/browser"
	"github.com/yellowhama/musu-nurikun/internal/db"
)

var signupCmd = &cobra.Command{
	Use:   "signup [platform]",
	Short: "Autonomously sign up for a platform using a forged identity",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		platform := args[0]
		project := viper.GetString("project")
		dbPath := viper.GetString("db_path")
		if dbPath == "" {
			dbPath = filepath.Join("projects", project, "data", "nurikun.db")
		}
		idFlag, _ := cmd.Flags().GetInt("id")

		store, err := db.NewStore(dbPath)
		if err != nil {
			fmt.Printf("❌ DB Error: %v\n", err)
			return
		}

		ident, err := store.GetIdentity(idFlag)
		if err != nil {
			fmt.Printf("❌ Identity ID %d not found.\n", idFlag)
			return
		}

		fmt.Printf("🐣 Starting autonomous signup for %s using identity: %s\n", platform, ident.Name)

		// 1. Initialize Stealth Walker
		walker, err := browser.NewWalker(project)
		if err != nil {
			fmt.Printf("❌ Walker Error: %v\n", err)
			return
		}
		defer walker.Close()

		// 2. Initial Navigation
		targetURL := ""
		switch strings.ToLower(platform) {
		case "reddit":
			targetURL = "https://www.reddit.com/register/"
		case "x", "twitter":
			targetURL = "https://twitter.com/i/flow/signup"
		default:
			targetURL = platform
		}

		page, err := walker.Navigate(targetURL)
		if err != nil {
			fmt.Printf("❌ Navigation failed: %v\n", err)
			return
		}
		defer page.Close() // Ensure page is closed

		// 3. Cognitive Loop
		navigator := agent.NewNavigator("llama3")
		goal := fmt.Sprintf("Register a new account on %s for %s (%s).", platform, ident.Name, ident.Email)

		for i := 0; i < 15; i++ {
			fmt.Printf("\n🧠 Step %d: Consulting AI for next action...\n", i+1)
			action, err := navigator.DetermineNextAction(goal, page)
			if err != nil {
				fmt.Printf("❌ AI Error: %v\n", err)
				break
			}

			fmt.Printf("🎯 Action: %s on %s | Reason: %s\n", action.Action, action.Selector, action.Reason)

			if action.Action == "done" {
				fmt.Println("🎉 AI claims registration is complete!")
				break
			}

			// Execute Action via Stealth Walker
			switch action.Action {
			case "click":
				err = walker.HumanClick(page, action.Selector)
			case "fill":
				val := action.Value
				lowerSel := strings.ToLower(action.Selector)
				if strings.Contains(lowerSel, "email") { val = ident.Email }
				if strings.Contains(lowerSel, "user") || strings.Contains(lowerSel, "name") { val = ident.Name }
				err = page.Fill(action.Selector, val)
			case "wait":
				time.Sleep(5 * time.Second)
			}

			if err != nil {
				fmt.Printf("   ⚠️  Action failed: %v\n", err)
			}

			time.Sleep(2 * time.Second)
		}

		fmt.Printf("\n🏁 Autonomous signup session ended.\n")
		shotPath := filepath.Join("projects", project, "screenshots", "signup_final.png")
		os.MkdirAll(filepath.Dir(shotPath), 0755)
		page.Screenshot(playwright.PageScreenshotOptions{
			Path: playwright.String(shotPath),
		})
	},
}

func init() {
	signupCmd.Flags().Int("id", 1, "ID of the forged identity to use")
	rootCmd.AddCommand(signupCmd)
}
