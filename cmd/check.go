package cmd

import (
	"fmt"
	"time"

	"github.com/naru-bot/upp/internal/checker"
	"github.com/naru-bot/upp/internal/config"
	"github.com/naru-bot/upp/internal/db"
	"github.com/naru-bot/upp/internal/notify"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "check [name|url|id]",
		Short: "Run checks now (all targets or specific one)",
		Long: `Run uptime and change detection checks immediately.

Without arguments, checks all targets. With an argument, checks only the specified target.

Examples:
  upp check
  upp check "My Site"
  upp check https://example.com`,
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
			fmt.Println("No targets to check. Use 'upp add <url>' first.")
		}
		return
	}

	var outputs []checkOutput

	for _, t := range targets {
		if t.Paused {
			continue
		}

		if !jsonOutput && !quiet {
			fmt.Printf("  ⟳ Checking %s...\r", t.Name)
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
			// Clear the "checking" line
			fmt.Printf("\r\033[K")

			icon := statusIcon(result.Status)
			statusText := result.Status
			nameText := t.Name
			urlText := fmt.Sprintf("(%s)", t.URL)
			respText := fmt.Sprintf("[%dms]", result.ResponseTime.Milliseconds())

			if !noColor {
				switch result.Status {
				case "up", "unchanged":
					icon = colorGreen(icon)
					statusText = colorGreen(statusText)
				case "changed":
					icon = colorYellow(icon)
					statusText = colorYellow(statusText)
				case "down", "error":
					icon = colorRed(icon)
					statusText = colorRed(statusText)
				}
				nameText = colorBold(t.Name)
				urlText = colorCyan(fmt.Sprintf("(%s)", t.URL))
			}

			fmt.Printf("%s %s %s — %s %s",
				icon, nameText, urlText, statusText, respText)
			if result.Error != "" {
				errText := result.Error
				if !noColor {
					errText = colorRed(errText)
				}
				fmt.Printf(" (%s)", errText)
			}
			if result.SSLExpiry != nil {
				days := int(time.Until(*result.SSLExpiry).Hours() / 24)
				warnDays := config.Get().SSLWarnDays()
				if days < warnDays {
					sslText := fmt.Sprintf("[SSL: %dd]", days)
					if !noColor {
						if days < warnDays/2 {
							sslText = colorRed(sslText)
						} else {
							sslText = colorYellow(sslText)
						}
					}
					fmt.Printf(" %s", sslText)
				}
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

	msg := fmt.Sprintf("[upp] %s (%s) is %s", target, url, status)
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
