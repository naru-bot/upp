package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/naru-bot/upp/internal/db"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "history <name|url|id>",
		Short: "Show check history for a target",
		Args:  requireArgs(1),
		Run:   runHistory,
	}
	cmd.Flags().IntP("limit", "l", 20, "Number of results to show")
	rootCmd.AddCommand(cmd)
}

func runHistory(cmd *cobra.Command, args []string) {
	limit, _ := cmd.Flags().GetInt("limit")

	t, err := db.GetTarget(args[0])
	if err != nil {
		exitError(err.Error())
	}

	results, err := db.GetCheckHistory(t.ID, limit)
	if err != nil {
		exitError(err.Error())
	}

	if jsonOutput {
		printJSON(results)
		return
	}

	if len(results) == 0 {
		fmt.Println("No check history. Run 'upp check' first.")
		return
	}

	fmt.Printf("History for: %s (%s)\n\n", t.Name, t.URL)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "TIME\tSTATUS\tCODE\tRESPONSE\tERROR\n")
	fmt.Fprintf(w, "────\t──────\t────\t────────\t─────\n")

	for _, r := range results {
		fmt.Fprintf(w, "%s\t%s\t%d\t%dms\t%s\n",
			r.CheckedAt.Format("2006-01-02 15:04:05"), r.Status, r.StatusCode, r.ResponseTime, r.Error)
	}
	w.Flush()
}
