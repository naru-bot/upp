package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/naru-bot/watchdog/internal/checker"
	"github.com/naru-bot/watchdog/internal/db"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Live dashboard — continuously monitor all targets",
		Long: `Display a live-updating terminal dashboard of all monitored targets.

Refreshes at the configured interval. Press Ctrl+C to stop.

Examples:
  watchdog watch
  watchdog watch --refresh 10`,
		Run: runWatch,
	}
	cmd.Flags().IntP("refresh", "r", 30, "Refresh interval in seconds")
	rootCmd.AddCommand(cmd)
}

func runWatch(cmd *cobra.Command, args []string) {
	refresh, _ := cmd.Flags().GetInt("refresh")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	fmt.Printf("🐕 Watchdog live dashboard (refresh: %ds) — Ctrl+C to stop\n\n", refresh)

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
	fmt.Printf("%s  %s\n\n", colorBold("🐕 Watchdog"), colorCyan(now.Format("2006-01-02 15:04:05")))

	if len(targets) == 0 {
		fmt.Println("No targets configured. Use 'watchdog add <url>' to start.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\t%s\n",
		colorBold("TARGET"), colorBold("STATUS"), colorBold("UPTIME"), colorBold("RESP"), colorBold("CHANGES"), colorBold("LAST CHECK"))
	fmt.Fprintf(w, "  ──────\t──────\t──────\t────\t───────\t──────────\n")

	for _, t := range targets {
		// Run check
		result := checker.Check(&t)
		cr := &db.CheckResult{
			TargetID:     t.ID,
			Status:       result.Status,
			StatusCode:   result.StatusCode,
			ResponseTime: result.ResponseTime.Milliseconds(),
			ContentHash:  result.ContentHash,
			Error:        result.Error,
		}
		db.SaveCheckResult(cr)

		if result.Content != "" && result.ContentHash != "" {
			snaps, _ := db.GetLatestSnapshots(t.ID, 1)
			if len(snaps) == 0 || snaps[0].Hash != result.ContentHash {
				db.SaveSnapshot(t.ID, result.Content, result.ContentHash)
			}
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
		statusStr := result.Status
		switch result.Status {
		case "up", "unchanged":
			statusStr = colorGreen("● " + result.Status)
		case "changed":
			statusStr = colorYellow("△ changed")
		case "down", "error":
			statusStr = colorRed("✗ " + result.Status)
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

		fmt.Fprintf(w, "  %s\t%s\t%s\t%.0fms\t%d\t%s\n",
			truncate(t.Name, 25), statusStr, uptimeStr, avgMs, changes, now.Format("15:04:05"))
	}
	w.Flush()
	fmt.Printf("\n%s targets monitored\n", colorCyan(fmt.Sprintf("%d", len(targets))))
}
