package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-marketer/internal/db"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all drafted campaigns for the current project",
	Run: func(cmd *cobra.Command, args []string) {
		project := viper.GetString("project")
		dbPath := filepath.Join("projects", project, "data", "marketer.db")

		store, err := db.NewStore(dbPath)
		if err != nil {
			fmt.Printf("❌ Failed to connect to DB: %v\n", err)
			return
		}

		campaigns, err := store.ListCampaigns()
		if err != nil {
			fmt.Printf("❌ Failed to list campaigns: %v\n", err)
			return
		}

		fmt.Printf("\n📂 --- CAMPAIGN LIST (Project: %s) --- 📂\n", project)
		fmt.Printf("%-4s | %-20s | %-10s | %-10s\n", "ID", "Topic", "Status", "Persona")
		fmt.Println("---------------------------------------------------------")
		for _, c := range campaigns {
			fmt.Printf("%-4d | %-20s | %-10s | %-10s\n", c.ID, c.Topic, c.Status, c.Persona)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
