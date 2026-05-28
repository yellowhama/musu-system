package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	subscribeList  int
	subscribeEmail string
	subscribeName  string
)

var subscribeCmd = &cobra.Command{
	Use:   "subscribe",
	Short: "Add a self-subscribing recipient (pending — double opt-in required)",
	Long: `Record a subscription request for an opt-in list.

This is the FIRST half of double opt-in: the subscriber is stored as 'pending'
and is NOT mailable. A confirmation token is issued; the recipient must confirm
(via the confirm command / URL) before any campaign will reach them. Only use
this for people who asked to subscribe — never for scraped or purchased lists.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if subscribeEmail == "" || subscribeList == 0 {
			return fmt.Errorf("--list and --email are required")
		}
		store, err := openStore()
		if err != nil {
			return err
		}
		defer store.Close()

		token, err := store.AddSubscriber(subscribeList, subscribeEmail, subscribeName, "self-subscribe")
		if err != nil {
			return fmt.Errorf("add subscriber %s to list %d: %w", subscribeEmail, subscribeList, err)
		}

		fmt.Printf("📨 Pending subscription recorded for %s (list #%d).\n", subscribeEmail, subscribeList)
		fmt.Println("   Status: pending — NOT yet mailable (awaiting double opt-in confirmation).")
		fmt.Printf("   Confirmation token: %s\n", token)
		fmt.Printf("\n   Confirm with:\n     musu-nurikun confirm --token %s\n", token)
		fmt.Printf("   (or send the recipient a confirm URL embedding this token)\n")
		return nil
	},
}

func init() {
	subscribeCmd.Flags().IntVar(&subscribeList, "list", 0, "List ID to subscribe to (required)")
	subscribeCmd.Flags().StringVar(&subscribeEmail, "email", "", "Subscriber email address (required)")
	subscribeCmd.Flags().StringVar(&subscribeName, "name", "", "Subscriber display name (optional)")
	rootCmd.AddCommand(subscribeCmd)
}
