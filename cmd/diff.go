package cmd

import (
	"fmt"

	"github.com/naru-bot/upp/internal/db"
	"github.com/naru-bot/upp/internal/diff"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "diff <name|url|id>",
		Short: "Show content changes between snapshots",
		Long: `Show what changed in the monitored page content.

Compares the two most recent snapshots and displays a unified diff.

Examples:
  upp diff "My Site"
  upp diff https://example.com
  upp diff 1`,
		Args: requireArgs(1),
		Run:  runDiff,
	})
}

type diffOutput struct {
	Target     string        `json:"target"`
	URL        string        `json:"url"`
	HasChanges bool          `json:"has_changes"`
	Summary    string        `json:"summary"`
	Added      int           `json:"lines_added"`
	Removed    int           `json:"lines_removed"`
	Changes    []diff.Change `json:"changes,omitempty"`
	OldTime    string        `json:"old_snapshot_time,omitempty"`
	NewTime    string        `json:"new_snapshot_time,omitempty"`
}

func runDiff(cmd *cobra.Command, args []string) {
	t, err := db.GetTarget(args[0])
	if err != nil {
		exitError(err.Error())
	}

	snaps, err := db.GetLatestSnapshots(t.ID, 2)
	if err != nil {
		exitError(err.Error())
	}

	if len(snaps) < 2 {
		if jsonOutput {
			printJSON(diffOutput{
				Target:     t.Name,
				URL:        t.URL,
				HasChanges: false,
				Summary:    "Not enough snapshots to compare (need at least 2 checks)",
			})
		} else {
			fmt.Println("Not enough snapshots to compare. Run 'upp check' at least twice.")
		}
		return
	}

	// snaps[0] is newest, snaps[1] is older
	d := diff.Diff(snaps[1].Content, snaps[0].Content)

	if jsonOutput {
		printJSON(diffOutput{
			Target:     t.Name,
			URL:        t.URL,
			HasChanges: d.HasChanges,
			Summary:    d.Summary,
			Added:      d.Added,
			Removed:    d.Removed,
			Changes:    d.Changes,
			OldTime:    snaps[1].CreatedAt.String(),
			NewTime:    snaps[0].CreatedAt.String(),
		})
		return
	}

	fmt.Printf("Changes for: %s (%s)\n", t.Name, t.URL)
	fmt.Printf("Old: %s\nNew: %s\n\n", snaps[1].CreatedAt.Format("2006-01-02 15:04:05"), snaps[0].CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Print(diff.FormatUnified(d, "previous", "current"))
}
