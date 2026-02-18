package cmd

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"

	"github.com/cheryeong/watchdog/internal/db"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export monitoring data as JSON or CSV",
		Long: `Export all targets and their check history.

Examples:
  watchdog export --json > data.json
  watchdog export --format csv > data.csv`,
		Run: runExport,
	}
	cmd.Flags().String("format", "json", "Export format: json, csv")
	rootCmd.AddCommand(cmd)
}

type exportData struct {
	Targets []db.Target       `json:"targets"`
	Results []db.CheckResult  `json:"check_results"`
}

func runExport(cmd *cobra.Command, args []string) {
	format, _ := cmd.Flags().GetString("format")

	targets, err := db.ListTargets()
	if err != nil {
		exitError(err.Error())
	}

	if format == "csv" {
		w := csv.NewWriter(os.Stdout)
		w.Write([]string{"target_id", "target_name", "url", "type", "status", "status_code", "response_ms", "error", "checked_at"})

		for _, t := range targets {
			results, _ := db.GetCheckHistory(t.ID, 10000)
			for _, r := range results {
				w.Write([]string{
					strconv.FormatInt(r.TargetID, 10),
					t.Name,
					t.URL,
					t.Type,
					r.Status,
					strconv.Itoa(r.StatusCode),
					strconv.FormatInt(r.ResponseTime, 10),
					r.Error,
					r.CheckedAt.String(),
				})
			}
		}
		w.Flush()
		return
	}

	// Default: JSON
	var allResults []db.CheckResult
	for _, t := range targets {
		results, _ := db.GetCheckHistory(t.ID, 10000)
		allResults = append(allResults, results...)
	}

	data := exportData{Targets: targets, Results: allResults}
	if targets == nil {
		data.Targets = []db.Target{}
	}
	if allResults == nil {
		data.Results = []db.CheckResult{}
	}

	printJSON(data)
	if !jsonOutput {
		fmt.Fprintf(os.Stderr, "\nTip: Use 'watchdog export --json' to suppress this message\n")
	}
}
