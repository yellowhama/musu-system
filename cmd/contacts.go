package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yellowhama/musu-nurikun/internal/db"
)

var contactsList int

var contactsCmd = &cobra.Command{
	Use:   "contacts",
	Short: "List the subscribers of a list with their consent status",
	Long:  `Show every subscriber of a list and its status (pending / confirmed / unsubscribed).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if contactsList == 0 {
			return fmt.Errorf("--list is required")
		}
		store, err := openStore()
		if err != nil {
			return err
		}
		defer store.Close()

		list, err := store.GetList(contactsList)
		if err != nil {
			return fmt.Errorf("get list %d: %w", contactsList, err)
		}

		subs, err := store.ListSubscribers(list.ID)
		if err != nil {
			return fmt.Errorf("list subscribers for list %d: %w", list.ID, err)
		}

		fmt.Printf("Subscribers on list #%d %q (cadence %d days):\n", list.ID, list.Name, list.CadenceDays)
		if len(subs) == 0 {
			fmt.Println("  (no subscribers yet)")
			return nil
		}
		fmt.Printf("  %-5s %-32s %-20s %s\n", "ID", "EMAIL", "NAME", "STATUS")
		var confirmed int
		for _, s := range subs {
			printContact(s)
			if s.Status == "confirmed" {
				confirmed++
			}
		}
		fmt.Printf("\n%d subscriber(s): %d confirmed (mailable, subject to suppression + cadence).\n", len(subs), confirmed)
		return nil
	},
}

func printContact(s db.Subscriber) {
	fmt.Printf("  %-5d %-32s %-20s %s\n", s.ID, s.Email, s.Name, s.Status)
}

func init() {
	contactsCmd.Flags().IntVar(&contactsList, "list", 0, "List ID whose subscribers to show (required)")
	rootCmd.AddCommand(contactsCmd)
}
