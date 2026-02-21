package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/naru-bot/upp/internal/db"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all monitored targets",
		Aliases: []string{"ls"},
		Run: func(cmd *cobra.Command, args []string) {
			targets, err := db.ListTargets()
			if err != nil {
				exitError(err.Error())
			}

			if jsonOutput {
				printJSON(targets)
				return
			}

			if len(targets) == 0 {
				fmt.Println("No targets configured. Use 'upp add <url>' to start monitoring.")
				return
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "ID\tNAME\tURL\tTYPE\tINTERVAL\tSTATUS\n")
			fmt.Fprintf(w, "──\t────\t───\t────\t────────\t──────\n")

			for _, t := range targets {
				status := "active"
				if t.Paused {
					status = "paused"
				}
				// Get last check result
				results, err := db.GetCheckHistory(t.ID, 1)
				if err == nil && len(results) > 0 {
					last := results[0]
					age := time.Since(last.CheckedAt).Round(time.Second)
					status = fmt.Sprintf("%s (%s ago)", last.Status, age)
				}

				fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%ds\t%s\n",
					t.ID, t.Name, truncate(t.URL, 40), t.Type, t.Interval, status)
			}
			w.Flush()
		},
	})
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
