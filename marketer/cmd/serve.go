package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/yellowhama/musu-marketer/internal/api"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the musu-marketer REST API server",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		srv := api.NewServer(port)
		if err := srv.Start(); err != nil {
			fmt.Printf("❌ Server failed: %v\n", err)
		}
	},
}

func init() {
	// Fixed: Removed 'p' shorthand for port because it's already used for 'project' globally.
	serveCmd.Flags().Int("port", 8081, "Port to run the API server")
	rootCmd.AddCommand(serveCmd)
}
