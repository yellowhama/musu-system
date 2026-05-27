package cmd

import (
	"bufio"
	"encoding/json"
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

		var meta db.IdentityMetadata
		json.Unmarshal([]byte(ident.Metadata), &meta)

		fmt.Printf("🐣 Starting autonomous signup for %s using identity: %s\n", platform, ident.Name)

		walker, err := browser.NewWalker(project)
		if err != nil {
			fmt.Printf("❌ Walker Error: %v\n", err)
			return
		}
		defer walker.Close()

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
		defer page.Close() 

		navigator := agent.NewNavigator("llama3")
		goal := fmt.Sprintf(`Register a new account on %s for %s (%s).`, platform, ident.Name, ident.Email)

		reader := bufio.NewReader(os.Stdin)

		for i := 0; i < 20; i++ {
			fmt.Printf("\n🧠 Step %d: Consulting AI for next action...\n", i+1)
			
			action, err := navigator.DetermineNextAction(goal, page)
			
			// ⚡ Agentic Handover Logic (Refined)
			if err != nil {
				if strings.Contains(err.Error(), "AGENT_REQUIRED") {
					fmt.Println("\n⚠️  AGENT INTERVENTION REQUIRED")
					fmt.Println(err.Error()) // Prints DOM
					
					shotPath := filepath.Join("projects", project, "screenshots", fmt.Sprintf("step_%d.png", i+1))
					os.MkdirAll(filepath.Dir(shotPath), 0755)
					page.Screenshot(playwright.PageScreenshotOptions{Path: playwright.String(shotPath)})
					fmt.Printf("📸 Screenshot saved to: %s\n", shotPath)
					
					fmt.Println("\n👉 Agent, provide next action JSON (or type 'done'):")
					fmt.Print("INSTRUCTION > ")
					
					input, _ := reader.ReadString('\n')
					input = strings.TrimSpace(input)
					if input == "done" {
						break
					}
					
					action = &agent.BrowserAction{}
					if uErr := json.Unmarshal([]byte(input), &action); uErr != nil {
						fmt.Printf("❌ Invalid JSON: %v. Retrying...\n", uErr)
						continue
					}
				} else {
					fmt.Printf("❌ Fatal Error: %v\n", err)
					break
				}
			}

			fmt.Printf("🎯 Executing Action: %s on %s | Reason: %s\n", action.Action, action.Selector, action.Reason)

			if action.Action == "done" {
				fmt.Println("🎉 AI claims registration is complete!")
				break
			}

			switch action.Action {
			case "click":
				err = walker.HumanClick(page, action.Selector)
			case "fill":
				val := action.Value
				lowSel := strings.ToLower(action.Selector)
				if strings.Contains(lowSel, "email") { val = ident.Email }
				if strings.Contains(lowSel, "user") || strings.Contains(lowSel, "name") { val = ident.Name }
				if strings.Contains(lowSel, "pass") { val = meta.EmailPassword }
				err = page.Fill(action.Selector, val)
			case "wait":
				time.Sleep(5 * time.Second)
			}

			if err != nil {
				fmt.Printf("   ⚠️  Action execution failed: %v\n", err)
			}
			time.Sleep(2 * time.Second)
		}

		fmt.Printf("\n🏁 Autonomous signup session ended.\n")
	},
}

func init() {
	signupCmd.Flags().Int("id", 1, "ID of the forged identity to use")
	rootCmd.AddCommand(signupCmd)
}
