package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/cheryeong/watchdog/internal/db"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "status [name|url|id]",
		Short: "Show uptime statistics and status summary",
		Long: `Display uptime percentage, average response time, and recent status.

Without arguments, shows summary for all targets.

Examples:
  watchdog status
  watchdog status "My Site"
  watchdog status --period 7d`,
		Run: runStatus,
	}
	cmd.Flags().StringP("period", "p", "24h", "Stats period: 1h, 24h, 7d, 30d")
	rootCmd.AddCommand(cmd)
}

type statusOutput struct {
	Target        string  `json:"target"`
	URL           string  `json:"url"`
	UptimePercent float64 `json:"uptime_percent"`
	AvgResponseMs float64 `json:"avg_response_ms"`
	TotalChecks   int     `json:"total_checks"`
	LastStatus    string  `json:"last_status"`
	LastChecked   string  `json:"last_checked,omitempty"`
	Changes       int     `json:"content_changes"`
}

func runStatus(cmd *cobra.Command, args []string) {
	period, _ := cmd.Flags().GetString("period")
	since := parsePeriod(period)

	var targets []db.Target
	if len(args) > 0 {
		t, err := db.GetTarget(args[0])
		if err != nil {
			exitError(err.Error())
		}
		targets = []db.Target{*t}
	} else {
		var err error
		targets, err = db.ListTargets()
		if err != nil {
			exitError(err.Error())
		}
	}

	if len(targets) == 0 {
		if jsonOutput {
			printJSON([]interface{}{})
		} else {
			fmt.Println("No targets configured.")
		}
		return
	}

	var outputs []statusOutput

	for _, t := range targets {
		total, up, avgMs, err := db.GetUptimeStats(t.ID, since)
		if err != nil {
			continue
		}

		var uptimePct float64
		if total > 0 {
			uptimePct = float64(up) / float64(total) * 100
		}

		// Count changes
		results, _ := db.GetCheckHistory(t.ID, 1000)
		changes := 0
		lastStatus := "unknown"
		lastChecked := ""
		for i, r := range results {
			if i == 0 {
				lastStatus = r.Status
				lastChecked = r.CheckedAt.Format(time.RFC3339)
			}
			if r.Status == "changed" {
				changes++
			}
		}

		out := statusOutput{
			Target:        t.Name,
			URL:           t.URL,
			UptimePercent: uptimePct,
			AvgResponseMs: avgMs,
			TotalChecks:   total,
			LastStatus:    lastStatus,
			LastChecked:   lastChecked,
			Changes:       changes,
		}
		outputs = append(outputs, out)
	}

	if jsonOutput {
		printJSON(outputs)
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "TARGET\tUPTIME\tAVG RESP\tCHECKS\tCHANGES\tLAST STATUS\n")
	fmt.Fprintf(w, "──────\t──────\t────────\t──────\t───────\t───────────\n")

	for _, o := range outputs {
		fmt.Fprintf(w, "%s\t%.1f%%\t%.0fms\t%d\t%d\t%s\n",
			truncate(o.Target, 25), o.UptimePercent, o.AvgResponseMs, o.TotalChecks, o.Changes, o.LastStatus)
	}
	w.Flush()
}

func parsePeriod(p string) time.Time {
	now := time.Now()
	switch p {
	case "1h":
		return now.Add(-1 * time.Hour)
	case "7d":
		return now.Add(-7 * 24 * time.Hour)
	case "30d":
		return now.Add(-30 * 24 * time.Hour)
	default:
		return now.Add(-24 * time.Hour)
	}
}
