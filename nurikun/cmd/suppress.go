package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var suppressEmail string

var suppressCmd = &cobra.Command{
	Use:     "suppress",
	Aliases: []string{"unsubscribe"},
	Short:   "Unsubscribe an address and add it to the suppression list",
	Long: `Honor an opt-out request.

Marks every list membership for the address as 'unsubscribed' and adds it to
the suppression list — the hard send-time gate that no campaign flag can
bypass. Once suppressed, the address is never sent to again unless it is
removed from suppression out-of-band.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if suppressEmail == "" {
			return fmt.Errorf("--email is required")
		}
		store, err := openStore()
		if err != nil {
			return err
		}
		defer store.Close()

		if err := store.Unsubscribe(suppressEmail, "manual"); err != nil {
			return fmt.Errorf("unsubscribe %s: %w", suppressEmail, err)
		}
		fmt.Printf("🚫 %s unsubscribed and suppressed (reason: manual). It will not be mailed again.\n", suppressEmail)
		return nil
	},
}

func init() {
	suppressCmd.Flags().StringVar(&suppressEmail, "email", "", "Email address to unsubscribe/suppress (required)")
	rootCmd.AddCommand(suppressCmd)
}
