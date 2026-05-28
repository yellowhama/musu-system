package cmd

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-marketer/internal/db"
	"github.com/yellowhama/musu-marketer/internal/publisher"
)

var publishCmd = &cobra.Command{
	Use:   "publish [ID]",
	Short: "Publish a drafted campaign using a registered adapter",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		project := viper.GetString("project")
		id, _ := strconv.Atoi(args[0])
		dbPath := filepath.Join("projects", project, "data", "marketer.db")
		platform, _ := cmd.Flags().GetString("platform")

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

		fmt.Printf("📢 Publishing campaign %d (%s) using adapter: %s...\n", id, c.Topic, platform)

		// ⚡ Optimized: Get from Registry
		pub, err := publisher.Get(platform)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			fmt.Printf("💡 Available adapters: %v\n", publisher.List())
			return
		}

		loc, err := pub.Publish(c.Topic, c.Content)
		if err != nil {
			fmt.Printf("❌ Publication failed: %v\n", err)
			return
		}

		_, _ = store.UpdateStatus(id, "published")

		fmt.Printf("✅ Success! Location/Log: %s\n", loc)
	},
}

func init() {
	publishCmd.Flags().String("platform", "local", "Adapter to use for publishing")
	rootCmd.AddCommand(publishCmd)
}
