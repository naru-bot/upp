package cmd

import (
	"fmt"
	"time"

	"github.com/naru-bot/upp/internal/db"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "data <name|url|id>",
		Short: "Show latest stored snapshot content for a target",
		Long: `Show the latest stored snapshot content for a target.

Examples:
  upp data "My Site"
  upp data https://example.com
  upp data 1`,
		Args: requireArgs(1),
		Run:  runData,
	})
}

type dataOutput struct {
	Target   db.Target   `json:"target"`
	Snapshot db.Snapshot `json:"snapshot"`
}

func runData(cmd *cobra.Command, args []string) {
	t, err := db.GetTarget(args[0])
	if err != nil {
		exitError(err.Error())
	}

	snaps, err := db.GetLatestSnapshots(t.ID, 1)
	if err != nil {
		exitError(err.Error())
	}
	if len(snaps) == 0 {
		if jsonOutput {
			printJSON(map[string]string{"error": "no snapshots found (run 'upp check')"})
			return
		}
		fmt.Println("No snapshots found. Run 'upp check' first.")
		return
	}

	snap := snaps[0]
	if jsonOutput {
		printJSON(dataOutput{Target: *t, Snapshot: snap})
		return
	}

	fmt.Printf("Target: %s (id %d)\n", t.Name, t.ID)
	fmt.Printf("URL: %s\n", t.URL)
	fmt.Printf("Snapshot: %s\n\n", snap.CreatedAt.Format(time.RFC3339))
	fmt.Print(snap.Content)
	if len(snap.Content) > 0 && snap.Content[len(snap.Content)-1] != '\n' {
		fmt.Print("\n")
	}
}
