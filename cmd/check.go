package cmd

import (
	"fmt"
	"time"

	"github.com/cheryeong/watchdog/internal/checker"
	"github.com/cheryeong/watchdog/internal/db"
	"github.com/cheryeong/watchdog/internal/notify"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "check [name|url|id]",
		Short: "Run checks now (all targets or specific one)",
		Long: `Run uptime and change detection checks immediately.

Without arguments, checks all targets. With an argument, checks only the specified target.

Examples:
  watchdog check
  watchdog check "My Site"
  watchdog check https://example.com`,
		Run: runCheck,
	})
}

type checkOutput struct {
	Target       string `json:"target"`
	URL          string `json:"url"`
	Status       string `json:"status"`
	StatusCode   int    `json:"status_code,omitempty"`
	ResponseMs   int64  `json:"response_time_ms"`
	ContentHash  string `json:"content_hash,omitempty"`
	Changed      bool   `json:"changed"`
	Error        string `json:"error,omitempty"`
	SSLDaysLeft  *int   `json:"ssl_days_left,omitempty"`
}

func runCheck(cmd *cobra.Command, args []string) {
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
			fmt.Println("No targets to check. Use 'watchdog add <url>' first.")
		}
		return
	}

	var outputs []checkOutput

	for _, t := range targets {
		if t.Paused {
			continue
		}

		result := checker.Check(&t)

		// Save check result
		cr := &db.CheckResult{
			TargetID:     t.ID,
			Status:       result.Status,
			StatusCode:   result.StatusCode,
			ResponseTime: result.ResponseTime.Milliseconds(),
			ContentHash:  result.ContentHash,
			Error:        result.Error,
		}
		db.SaveCheckResult(cr)

		// Save snapshot if content available
		if result.Content != "" && result.ContentHash != "" {
			snaps, _ := db.GetLatestSnapshots(t.ID, 1)
			if len(snaps) == 0 || snaps[0].Hash != result.ContentHash {
				db.SaveSnapshot(t.ID, result.Content, result.ContentHash)
			}
		}

		out := checkOutput{
			Target:      t.Name,
			URL:         t.URL,
			Status:      result.Status,
			StatusCode:  result.StatusCode,
			ResponseMs:  result.ResponseTime.Milliseconds(),
			ContentHash: result.ContentHash,
			Changed:     result.Status == "changed",
			Error:       result.Error,
		}

		if result.SSLExpiry != nil {
			days := int(time.Until(*result.SSLExpiry).Hours() / 24)
			out.SSLDaysLeft = &days
		}

		outputs = append(outputs, out)

		// Send notifications if down or changed
		if result.Status == "down" || result.Status == "changed" || result.Status == "error" {
			sendNotifications(t.Name, t.URL, result.Status, result.Error)
		}

		if !jsonOutput {
			icon := statusIcon(result.Status)
			fmt.Printf("%s %s (%s) — %s [%dms]",
				icon, t.Name, t.URL, result.Status, result.ResponseTime.Milliseconds())
			if result.Error != "" {
				fmt.Printf(" (%s)", result.Error)
			}
			if result.SSLExpiry != nil {
				days := int(time.Until(*result.SSLExpiry).Hours() / 24)
				fmt.Printf(" [SSL: %dd]", days)
			}
			fmt.Println()
		}
	}

	if jsonOutput {
		printJSON(outputs)
	}
}

func statusIcon(status string) string {
	switch status {
	case "up", "unchanged":
		return "✓"
	case "changed":
		return "△"
	case "down":
		return "✗"
	default:
		return "?"
	}
}

func sendNotifications(target, url, status, errMsg string) {
	configs, err := db.ListNotifyConfigs()
	if err != nil || len(configs) == 0 {
		return
	}

	msg := fmt.Sprintf("[watchdog] %s (%s) is %s", target, url, status)
	if errMsg != "" {
		msg += ": " + errMsg
	}

	event := notify.Event{
		Target:  target,
		URL:     url,
		Status:  status,
		Error:   errMsg,
		Time:    time.Now().UTC().Format(time.RFC3339),
		Message: msg,
	}

	for _, c := range configs {
		if c.Enabled {
			notify.Send(c.Type, c.Config, event)
		}
	}
}
