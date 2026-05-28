package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-system/nurikun/internal/db"
)

// dbPathForProject resolves the sqlite path for the active project, matching
// the layout created by `init` (projects/<project>/data/nurikun.db). An
// explicit db_path (env/config) overrides the default.
func dbPathForProject(project string) string {
	if p := viper.GetString("db_path"); p != "" {
		return p
	}
	return filepath.Join("projects", project, "data", "nurikun.db")
}

// openStore opens the project store, returning a helpful error if init hasn't run.
func openStore() (*db.Store, error) {
	project := viper.GetString("project")
	path := dbPathForProject(project)
	store, err := db.NewStore(path)
	if err != nil {
		return nil, fmt.Errorf("open store at %s (did you run `init`?): %w", path, err)
	}
	return store, nil
}

var (
	listsName    string
	listsCadence int
)

var listsCmd = &cobra.Command{
	Use:   "lists",
	Short: "Create an opt-in mailing list, or show existing lists",
	Long: `Manage opt-in mailing lists.

With --name, creates a new list (subscribers must self-subscribe and confirm
via double opt-in before they can be mailed). Without --name, prints the
existing lists for the active project.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := openStore()
		if err != nil {
			return err
		}
		defer store.Close()

		if listsName != "" {
			id, err := store.CreateList(listsName, listsCadence)
			if err != nil {
				return fmt.Errorf("create list %q: %w", listsName, err)
			}
			fmt.Printf("✅ Created list #%d %q (cadence: %d days)\n", id, listsName, listsCadence)
			return nil
		}

		lists, err := store.ListLists()
		if err != nil {
			return fmt.Errorf("list lists: %w", err)
		}
		if len(lists) == 0 {
			fmt.Println("No lists yet. Create one: musu-nurikun lists --name \"My Newsletter\"")
			return nil
		}
		fmt.Printf("%-4s %-30s %s\n", "ID", "NAME", "CADENCE(days)")
		for _, l := range lists {
			fmt.Printf("%-4d %-30s %d\n", l.ID, l.Name, l.CadenceDays)
		}
		return nil
	},
}

func init() {
	listsCmd.Flags().StringVar(&listsName, "name", "", "Name of the list to create (omit to list existing)")
	listsCmd.Flags().IntVar(&listsCadence, "cadence-days", 4, "Minimum days between sends to the same subscriber")
	rootCmd.AddCommand(listsCmd)
}
