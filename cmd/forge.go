package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-nurikun/internal/db"
	"github.com/yellowhama/musu-nurikun/internal/identity"
)

var forgeCmd = &cobra.Command{
	Use:   "forge [name]",
	Short: "Forge a new digital identity",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		project := viper.GetString("project")
		dbPath := viper.GetString("db_path")
		if dbPath == "" {
			dbPath = filepath.Join("projects", project, "data", "nurikun.db")
		}

		fmt.Printf("🎭 Forging identity '%s' for project '%s'...\n", name, project)

		// Using Mock Provider for now
		provider := &identity.MockProvider{}

		email, _ := provider.RequestEmail()
		fmt.Printf("📧 Email acquired: %s\n", email)

		phone, _, _ := provider.RequestPhone("reddit")
		fmt.Printf("📱 Phone number reserved: %s\n", phone)

		// Store in DB
		store, err := db.NewStore(dbPath)
		if err != nil {
			fmt.Printf("❌ DB Error: %v\n", err)
			return
		}

		metadata, _ := json.Marshal(map[string]string{
			"password": "TemporaryPassword123!",
			"provider": "mock",
		})

		id, err := store.SaveIdentity(name, email, phone, string(metadata))
		if err != nil {
			fmt.Printf("❌ Failed to save identity: %v\n", err)
			return
		}

		fmt.Printf("\n✅ Identity forged successfully (ID: %d)!\n", id)
		fmt.Printf("👉 Ready for 'musu-nurikun signup reddit --id %d'\n", id)
	},
}

func init() {
	rootCmd.AddCommand(forgeCmd)
}
