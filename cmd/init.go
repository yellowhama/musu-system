package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-nurikun/internal/db"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize musu-nurikun environment",
	Run: func(cmd *cobra.Command, args []string) {
		project := viper.GetString("project")
		fmt.Printf("🌐 Initializing musu-nurikun for project '%s'...\n", project)

		// 1. Create project directories
		baseDir := filepath.Join("projects", project)
		dirs := []string{
			filepath.Join(baseDir, "data"),
			filepath.Join(baseDir, "profiles"),
			filepath.Join(baseDir, "screenshots"),
		}
		
		for _, d := range dirs {
			if err := os.MkdirAll(d, 0755); err != nil {
				fmt.Printf("❌ Failed to create directory %s: %v\n", d, err)
				return
			}
			fmt.Printf("✅ Directory ready: ./%s\n", d)
		}

		// 2. Initialize Database
		dbPath := filepath.Join(baseDir, "data", "nurikun.db")
		_, err := db.NewStore(dbPath)
		if err != nil {
			fmt.Printf("❌ Failed to initialize database: %v\n", err)
			return
		}
		fmt.Printf("✅ Database ready: %s\n", dbPath)

		// 3. Create local project config
		configPath := filepath.Join(baseDir, "config.yaml")
		viper.Set("db_path", dbPath)
		viper.WriteConfigAs(configPath)
		fmt.Printf("✅ Project configuration saved: %s\n", configPath)

		fmt.Printf("\n✨ Initialization complete! Run 'musu-nurikun forge' to create your first digital identity.\n")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
