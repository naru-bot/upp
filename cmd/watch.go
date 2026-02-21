package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
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

// padRight pads a string with spaces to the given visible width.
// It accounts for ANSI escape codes not counting as visible characters.
func padRight(s string, width int) string {
	// Strip ANSI codes to get visible length
	visible := stripAnsi(s)
	pad := width - len(visible)
	if pad <= 0 {
		return s
	}
	return s + strings.Repeat(" ", pad)
}

// stripAnsi removes ANSI escape sequences from a string
func stripAnsi(s string) string {
	var result []byte
	inEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if s[i] == 'm' {
				inEscape = false
			}
			continue
		}
		result = append(result, s[i])
	}
	return string(result)
}

func renderDashboard() {
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

	// Column widths
	const (
		wTarget = 26
		wUptime = 8
		wResp   = 10
		wChg    = 9
		wLast   = 12
	)

	// Header
	fmt.Printf("  %s%s%s%s%s%s\n",
		padRight(colorBold("TARGET"), wTarget),
		padRight(colorBold("UPTIME"), wUptime),
		padRight(colorBold("RESP"), wResp),
		padRight(colorBold("CHANGES"), wChg),
		padRight(colorBold("LAST CHECK"), wLast),
		colorBold("STATUS"))
	fmt.Printf("  %s%s%s%s%s%s\n",
		padRight(strings.Repeat("─", wTarget-2), wTarget),
		padRight(strings.Repeat("─", wUptime-2), wUptime),
		padRight(strings.Repeat("─", wResp-2), wResp),
		padRight(strings.Repeat("─", wChg-2), wChg),
		padRight(strings.Repeat("─", wLast-2), wLast),
		"──────")

	for _, t := range targets {
		lastResults, _ := db.GetCheckHistory(t.ID, 1)

		status := "—"
		lastChecked := "never"
		if len(lastResults) > 0 {
			status = lastResults[0].Status
			lastChecked = lastResults[0].CheckedAt.Format("15:04:05")
		}

		since24h := now.Add(-24 * time.Hour)
		total, up, avgMs, _ := db.GetUptimeStats(t.ID, since24h)
		uptimePct := float64(0)
		if total > 0 {
			uptimePct = float64(up) / float64(total) * 100
		}

		results, _ := db.GetCheckHistory(t.ID, 1000)
		changes := 0
		for _, r := range results {
			if r.Status == "changed" {
				changes++
			}
		}

		// Status string
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
				if len(shortErr) > 25 {
					shortErr = shortErr[:22] + "..."
				}
				statusStr += " " + colorRed("("+shortErr+")")
			}
		}

		// Uptime string
		uptimeStr := fmt.Sprintf("%.1f%%", uptimePct)
		if uptimePct >= 99 {
			uptimeStr = colorGreen(uptimeStr)
		} else if uptimePct >= 95 {
			uptimeStr = colorYellow(uptimeStr)
		} else {
			uptimeStr = colorRed(uptimeStr)
		}

		respStr := fmt.Sprintf("%.0fms", avgMs)

		fmt.Printf("  %s%s%s%s%s%s\n",
			padRight(truncate(t.Name, wTarget-2), wTarget),
			padRight(uptimeStr, wUptime),
			padRight(respStr, wResp),
			padRight(fmt.Sprintf("%d", changes), wChg),
			padRight(lastChecked, wLast),
			statusStr)
	}
	fmt.Printf("\n%s targets monitored\n", colorCyan(fmt.Sprintf("%d", len(targets))))
}
