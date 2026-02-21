package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/naru-bot/upp/internal/db"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Live dashboard — continuously monitor all targets",
		Long: `Display a live-updating terminal dashboard of all monitored targets.

Refreshes at the configured interval. Press Ctrl+C to stop.

Examples:
  upp watch
  upp watch --refresh 10`,
		Run: runWatch,
	}
	cmd.Flags().IntP("refresh", "r", 30, "Refresh interval in seconds")
	rootCmd.AddCommand(cmd)
}

func runWatch(cmd *cobra.Command, args []string) {
	refresh, _ := cmd.Flags().GetInt("refresh")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	fmt.Printf("🐕 Upp live dashboard (refresh: %ds) — Ctrl+C to stop\n\n", refresh)

	ticker := time.NewTicker(time.Duration(refresh) * time.Second)
	defer ticker.Stop()

	// Run immediately, then on tick
	renderDashboard()

	for {
		select {
		case <-sig:
			fmt.Println("\n👋 Stopped.")
			return
		case <-ticker.C:
			renderDashboard()
		}
	}
}

func renderDashboard() {
	// Clear screen
	if !noColor {
		fmt.Print("\033[2J\033[H")
	}

	targets, err := db.ListTargets()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return
	}

	now := time.Now()
	fmt.Printf("%s  %s\n\n", colorBold("🐕 Upp"), colorCyan(now.Format("2006-01-02 15:04:05")))

	if len(targets) == 0 {
		fmt.Println("No targets configured. Use 'upp add <url>' to start.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\t%s\n",
		colorBold("TARGET"), colorBold("STATUS"), colorBold("UPTIME"), colorBold("RESP"), colorBold("CHANGES"), colorBold("LAST CHECK"))
	fmt.Fprintf(w, "  ──────\t──────\t──────\t────\t───────\t──────────\n")

	for _, t := range targets {
		// Read latest result from DB (daemon handles the actual checking)
		lastResults, _ := db.GetCheckHistory(t.ID, 1)

		status := "—"
		var respMs int64
		lastChecked := "never"
		if len(lastResults) > 0 {
			status = lastResults[0].Status
			respMs = lastResults[0].ResponseTime
			lastChecked = lastResults[0].CheckedAt.Format("15:04:05")
		}

		// Stats
		since24h := now.Add(-24 * time.Hour)
		total, up, avgMs, _ := db.GetUptimeStats(t.ID, since24h)
		uptimePct := float64(0)
		if total > 0 {
			uptimePct = float64(up) / float64(total) * 100
		}

		// Count changes
		results, _ := db.GetCheckHistory(t.ID, 1000)
		changes := 0
		for _, r := range results {
			if r.Status == "changed" {
				changes++
			}
		}

		// Color status
		statusStr := status
		switch status {
		case "up", "unchanged":
			statusStr = colorGreen("● " + status)
		case "changed":
			statusStr = colorYellow("△ changed")
		case "down", "error":
			statusStr = colorRed("✗ " + status)
			if len(lastResults) > 0 && lastResults[0].Error != "" {
				shortErr := lastResults[0].Error
				if len(shortErr) > 30 {
					shortErr = shortErr[:27] + "..."
				}
				statusStr += " " + colorRed("("+shortErr+")")
			}
		}

		// Color uptime
		uptimeStr := fmt.Sprintf("%.1f%%", uptimePct)
		if uptimePct >= 99 {
			uptimeStr = colorGreen(uptimeStr)
		} else if uptimePct >= 95 {
			uptimeStr = colorYellow(uptimeStr)
		} else {
			uptimeStr = colorRed(uptimeStr)
		}

		_ = respMs // use avgMs for display consistency
		fmt.Fprintf(w, "  %s\t%s\t%s\t%.0fms\t%d\t%s\n",
			truncate(t.Name, 25), statusStr, uptimeStr, avgMs, changes, lastChecked)
	}
	w.Flush()
	fmt.Printf("\n%s targets monitored\n", colorCyan(fmt.Sprintf("%d", len(targets))))
}
