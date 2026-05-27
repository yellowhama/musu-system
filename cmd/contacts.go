package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yellowhama/musu-nurikun/internal/db"
)

var contactsList int

var contactsCmd = &cobra.Command{
	Use:   "contacts",
	Short: "List the subscribers currently mailable on a list",
	Long: `Show subscribers for a list.

NOTE: the store exposes only DueSubscribers (confirmed, non-suppressed, past
the cadence window) — there is no "all subscribers" query and DB schema
changes are out of scope here. This command therefore prints the DueSubscribers
view as an approximation of the active audience for the list. Pending,
suppressed, and within-cadence subscribers are intentionally not shown.`,
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

		subs, err := store.DueSubscribers(list.ID, list.CadenceDays)
		if err != nil {
			return fmt.Errorf("list subscribers for list %d: %w", list.ID, err)
		}

		fmt.Printf("Mailable (due) subscribers on list #%d %q (cadence %d days):\n", list.ID, list.Name, list.CadenceDays)
		if len(subs) == 0 {
			fmt.Println("  (none currently due — confirmed subscribers may still exist within the cadence window)")
			return nil
		}
		fmt.Printf("  %-5s %-32s %-20s %s\n", "ID", "EMAIL", "NAME", "STATUS")
		for _, s := range subs {
			printContact(s)
		}
		fmt.Printf("\n%d due subscriber(s).\n", len(subs))
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
