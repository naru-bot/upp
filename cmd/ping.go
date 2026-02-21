package cmd

import (
	"fmt"
	"time"

	"github.com/naru-bot/upp/internal/checker"
	"github.com/naru-bot/upp/internal/config"
	"github.com/naru-bot/upp/internal/db"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "ping <url>",
		Short: "Quick one-off check without saving to database",
		Long: `Run a quick health check on a URL without adding it to the monitor list.

Useful for one-off diagnostics, scripting, and AI agent spot-checks.

Examples:
  upp ping https://example.com
  upp ping https://example.com --selector "h1"
  upp ping 192.168.1.1:3306 --type tcp
  upp ping example.com --type dns
  upp ping https://api.example.com --expect "ok"
  upp ping https://example.com --count 5`,
		Args: requireArgs(1),
		Run:  runPing,
	}
	cmd.Flags().StringP("type", "t", "http", "Check type: http, tcp, ping, dns")
	cmd.Flags().StringP("selector", "s", "", "CSS selector to extract")
	cmd.Flags().String("expect", "", "Expected keyword in response body")
	cmd.Flags().IntP("count", "c", 1, "Number of checks to run")
	cmd.Flags().Int("timeout", 30, "Timeout in seconds")
	rootCmd.AddCommand(cmd)
}

type pingOutput struct {
	URL          string `json:"url"`
	Status       string `json:"status"`
	StatusCode   int    `json:"status_code,omitempty"`
	ResponseMs   int64  `json:"response_time_ms"`
	ContentHash  string `json:"content_hash,omitempty"`
	SSLDaysLeft  *int   `json:"ssl_days_left,omitempty"`
	BodyMatch    *bool  `json:"body_match,omitempty"`
	Error        string `json:"error,omitempty"`
}

func runPing(cmd *cobra.Command, args []string) {
	url := args[0]
	typ, _ := cmd.Flags().GetString("type")
	selector, _ := cmd.Flags().GetString("selector")
	expect, _ := cmd.Flags().GetString("expect")
	count, _ := cmd.Flags().GetInt("count")

	// Create a temporary target (not saved to DB)
	target := &db.Target{
		URL:      url,
		Name:     url,
		Type:     typ,
		Selector: selector,
	}

	var outputs []pingOutput
	var totalMs int64

	for i := 0; i < count; i++ {
		result := checker.Check(target)
		
		out := pingOutput{
			URL:         url,
			Status:      result.Status,
			StatusCode:  result.StatusCode,
			ResponseMs:  result.ResponseTime.Milliseconds(),
			ContentHash: result.ContentHash,
			Error:       result.Error,
		}

		if result.SSLExpiry != nil {
			days := int(time.Until(*result.SSLExpiry).Hours() / 24)
			out.SSLDaysLeft = &days
		}

		// Check expected keyword
		if expect != "" && result.Content != "" {
			matched := containsString(result.Content, expect)
			out.BodyMatch = &matched
			if !matched && out.Status == "up" {
				out.Status = "keyword_missing"
			}
		}

		totalMs += out.ResponseMs
		outputs = append(outputs, out)

		if !jsonOutput {
			icon := statusIcon(result.Status)
			if out.BodyMatch != nil && !*out.BodyMatch {
				icon = colorYellow("⚠")
			} else if result.Status == "up" || result.Status == "unchanged" {
				icon = colorGreen("✓")
			} else if result.Status == "down" || result.Status == "error" {
				icon = colorRed("✗")
			}

			if count > 1 {
				fmt.Printf("[%d/%d] ", i+1, count)
			}
			fmt.Printf("%s %s — %s [%dms]", icon, url, result.Status, result.ResponseTime.Milliseconds())
			if result.Error != "" {
				fmt.Printf(" (%s)", result.Error)
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
			if out.BodyMatch != nil {
				if *out.BodyMatch {
					fmt.Printf(" %s", colorGreen("[keyword: found]"))
				} else {
					fmt.Printf(" %s", colorRed("[keyword: missing]"))
				}
			}
			fmt.Println()
		}

		if count > 1 && i < count-1 {
			time.Sleep(1 * time.Second)
		}
	}

	if count > 1 && !jsonOutput {
		avgMs := totalMs / int64(count)
		fmt.Printf("\n--- %s ping statistics ---\n", url)
		fmt.Printf("%d checks, avg %dms\n", count, avgMs)
	}

	if jsonOutput {
		if count == 1 {
			printJSON(outputs[0])
		} else {
			printJSON(outputs)
		}
	}
}

func containsString(haystack, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 && 
		contains(haystack, needle)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
