package cmd

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-marketer/internal/db"
)

var viewCmd = &cobra.Command{
	Use:   "view [ID]",
	Short: "View the content and strategy of a specific campaign",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		project := viper.GetString("project")
		id, _ := strconv.Atoi(args[0])
		dbPath := filepath.Join("projects", project, "data", "marketer.db")

		store, err := db.NewStore(dbPath)
		if err != nil {
			fmt.Printf("❌ Failed to connect to DB: %v\n", err)
			return
		}

		c, err := store.GetCampaign(id)
		if err != nil {
			fmt.Printf("❌ Campaign not found: %v\n", err)
			return
		}

		fmt.Printf("\n📑 --- VIEWING CAMPAIGN (Project: %s | ID: %d) --- 📑\n", project, c.ID)
		fmt.Printf("Topic: %s\n", c.Topic)
		fmt.Printf("Persona: %s | Status: %s | Created: %s\n", c.Persona, c.Status, c.CreatedAt)
		
		if c.Brief != "" {
			fmt.Println("\n🧠 STRATEGIC BRIEF:")
			fmt.Println(c.Brief)
		}
		
		fmt.Println("\n📝 CAMPAIGN CONTENT:")
		fmt.Println("---------------------------------------------------------")
		fmt.Println(c.Content)
	},
}

func init() {
	rootCmd.AddCommand(viewCmd)
}
