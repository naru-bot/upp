package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/naru-bot/upp/internal/db"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "status [name|url|id]",
		Short: "Show uptime statistics and status summary",
		Long: `Display uptime percentage, average response time, and recent status.

Without arguments, shows summary for all targets.

Examples:
  upp status
  upp status "My Site"
  upp status --period 7d`,
		Run: runStatus,
	}
	cmd.Flags().StringP("period", "p", "24h", "Stats period: 1h, 24h, 7d, 30d")
	rootCmd.AddCommand(cmd)
}

type statusOutput struct {
	Target        string  `json:"target"`
	URL           string  `json:"url"`
	Type          string  `json:"type"`
	UptimePercent float64 `json:"uptime_percent"`
	AvgResponseMs float64 `json:"avg_response_ms"`
	MinResponseMs int64   `json:"min_response_ms"`
	MaxResponseMs int64   `json:"max_response_ms"`
	TotalChecks   int     `json:"total_checks"`
	LastStatus    string  `json:"last_status"`
	LastError     string  `json:"last_error,omitempty"`
	LastChecked   string  `json:"last_checked,omitempty"`
	Changes       int     `json:"content_changes"`
	Sparkline     string  `json:"sparkline,omitempty"`
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

		// Count changes + build sparkline
		results, _ := db.GetCheckHistory(t.ID, 1000)
		changes := 0
		lastStatus := "unknown"
		lastError := ""
		lastChecked := ""
		var minMs, maxMs int64
		var responseTimes []int64
		for i, r := range results {
			if i == 0 {
				lastStatus = r.Status
				lastError = r.Error
				lastChecked = r.CheckedAt.Format(time.RFC3339)
				minMs = r.ResponseTime
				maxMs = r.ResponseTime
			}
			if r.Status == "changed" {
				changes++
			}
			if r.ResponseTime < minMs {
				minMs = r.ResponseTime
			}
			if r.ResponseTime > maxMs {
				maxMs = r.ResponseTime
			}
			responseTimes = append(responseTimes, r.ResponseTime)
		}

		// Build sparkline from last 20 checks (reversed to show oldest→newest)
		spark := buildSparkline(responseTimes, 20)

		out := statusOutput{
			Target:        t.Name,
			URL:           t.URL,
			Type:          t.Type,
			UptimePercent: uptimePct,
			AvgResponseMs: avgMs,
			MinResponseMs: minMs,
			MaxResponseMs: maxMs,
			TotalChecks:   total,
			LastStatus:    lastStatus,
			LastError:     lastError,
			LastChecked:   lastChecked,
			Changes:       changes,
			Sparkline:     spark,
		}
		outputs = append(outputs, out)
	}

	if jsonOutput {
		printJSON(outputs)
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "TARGET\tTYPE\tUPTIME\tAVG RESP\tCHECKS\tCHANGES\tRESPONSE\tSTATUS\n")
	fmt.Fprintf(w, "──────\t────\t──────\t────────\t──────\t───────\t────────\t──────\n")

	for _, o := range outputs {
		// Color uptime
		uptimeStr := fmt.Sprintf("%.1f%%", o.UptimePercent)
		if !noColor && !jsonOutput {
			if o.UptimePercent >= 99.9 {
				uptimeStr = colorGreen(uptimeStr)
			} else if o.UptimePercent >= 95 {
				uptimeStr = colorYellow(uptimeStr)
			} else if o.TotalChecks > 0 {
				uptimeStr = colorRed(uptimeStr)
			}
		}

		// Color status
		statusStr := o.LastStatus
		if !noColor && !jsonOutput {
			switch o.LastStatus {
			case "up", "unchanged":
				statusStr = colorGreen("● " + o.LastStatus)
			case "changed":
				statusStr = colorYellow("△ " + o.LastStatus)
			case "down", "error":
				statusStr = colorRed("✗ " + o.LastStatus)
			}
		}

		// Append short error to status if down
		if o.LastError != "" && (o.LastStatus == "down" || o.LastStatus == "error") {
			shortErr := shortenError(o.LastError)
			if !noColor && !jsonOutput {
				shortErr = colorRed(shortErr)
			}
			statusStr += " " + shortErr
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%.0fms\t%d\t%d\t%s\t%s\n",
			truncate(o.Target, 25), o.Type, uptimeStr, o.AvgResponseMs, o.TotalChecks, o.Changes, o.Sparkline, statusStr)
	}
	w.Flush()
}

func shortenError(err string) string {
	// Extract meaningful part from verbose Go errors
	if idx := strings.LastIndex(err, ": "); idx != -1 {
		err = err[idx+2:]
	}
	if len(err) > 40 {
		err = err[:37] + "..."
	}
	return "(" + err + ")"
}

func buildSparkline(values []int64, maxLen int) string {
	if len(values) == 0 {
		return ""
	}

	// Reverse (history is newest-first, we want oldest→newest)
	reversed := make([]int64, len(values))
	for i, v := range values {
		reversed[len(values)-1-i] = v
	}

	// Trim to maxLen
	if len(reversed) > maxLen {
		reversed = reversed[len(reversed)-maxLen:]
	}

	// Find min/max
	min, max := reversed[0], reversed[0]
	for _, v := range reversed {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	blocks := []rune("▁▂▃▄▅▆▇█")
	spread := max - min
	if spread == 0 {
		spread = 1
	}

	var result []rune
	for _, v := range reversed {
		idx := int(float64(v-min) / float64(spread) * float64(len(blocks)-1))
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}
		result = append(result, blocks[idx])
	}
	return string(result)
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
