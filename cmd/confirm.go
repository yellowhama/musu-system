package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var confirmToken string

var confirmCmd = &cobra.Command{
	Use:   "confirm",
	Short: "Confirm a pending subscription via its double opt-in token",
	Long: `Complete double opt-in: promote a 'pending' subscriber to 'confirmed'.

Only confirmed subscribers are ever included in a campaign. The token is the
one issued by the subscribe command; it is single-use.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if confirmToken == "" {
			return fmt.Errorf("--token is required")
		}
		store, err := openStore()
		if err != nil {
			return err
		}
		defer store.Close()

		sub, err := store.ConfirmSubscriber(confirmToken)
		if err != nil {
			return fmt.Errorf("confirm token: %w", err)
		}
		fmt.Printf("✅ Confirmed subscriber #%d: %s", sub.ID, sub.Email)
		if sub.Name != "" {
			fmt.Printf(" (%s)", sub.Name)
		}
		fmt.Printf(" on list #%d — status: %s\n", sub.ListID, sub.Status)
		fmt.Println("   This address is now mailable for campaigns on its list.")
		return nil
	},
}

func init() {
	confirmCmd.Flags().StringVar(&confirmToken, "token", "", "Confirmation token from `subscribe` (required)")
	rootCmd.AddCommand(confirmCmd)
}
