package cmd

import (
	"fmt"

	"github.com/naru-bot/upp/internal/db"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "add <url>",
		Short: "Add a URL to monitor",
		Long: `Add a URL for uptime monitoring and change detection.

Examples:
  upp add https://example.com
  upp add https://example.com --name "My Site" --interval 60
  upp add https://example.com --selector "div.price" --name "Price Watch"
  upp add https://api.example.com/health --expect "ok" --name "API Health"
  upp add 192.168.1.1:3306 --type tcp --name "MySQL"
  upp add example.com --type ping
  upp add example.com --type dns
  upp add https://example.com --retries 3 --timeout 10
  upp add https://example.com --type visual --threshold 7.5`,
		Args: requireArgs(1),
		Run:  runAdd,
	}

	cmd.Flags().StringP("name", "n", "", "Friendly name for the target")
	cmd.Flags().StringP("type", "t", "http", "Check type: http, tcp, ping, dns, visual")
	cmd.Flags().IntP("interval", "i", 300, "Check interval in seconds")
	cmd.Flags().StringP("selector", "s", "", "CSS selector for change detection")
	cmd.Flags().String("headers", "", "Custom headers as JSON string")
	cmd.Flags().String("expect", "", "Expected keyword in response body")
	cmd.Flags().Int("timeout", 30, "Request timeout in seconds")
	cmd.Flags().Int("retries", 1, "Retry count before marking as down")
	cmd.Flags().Float64("threshold", 5.0, "Visual diff threshold percentage (visual type only)")

	rootCmd.AddCommand(cmd)
}

func runAdd(cmd *cobra.Command, args []string) {
	url := args[0]
	name, _ := cmd.Flags().GetString("name")
	typ, _ := cmd.Flags().GetString("type")
	interval, _ := cmd.Flags().GetInt("interval")
	selector, _ := cmd.Flags().GetString("selector")
	headers, _ := cmd.Flags().GetString("headers")
	expect, _ := cmd.Flags().GetString("expect")
	timeout, _ := cmd.Flags().GetInt("timeout")
	retries, _ := cmd.Flags().GetInt("retries")
	threshold, _ := cmd.Flags().GetFloat64("threshold")

	target, err := db.AddTarget(name, url, typ, interval, selector, headers, expect, timeout, retries, threshold)
	if err != nil {
		exitError(err.Error())
	}

	if jsonOutput {
		printJSON(target)
	} else {
		fmt.Printf("✓ Added: %s (%s)\n", target.Name, target.URL)
		fmt.Printf("  Type: %s | Interval: %ds | Timeout: %ds | Retries: %d", target.Type, target.Interval, target.Timeout, target.Retries)
		if target.Selector != "" {
			fmt.Printf(" | Selector: %s", target.Selector)
		}
		if target.Expect != "" {
			fmt.Printf(" | Expect: %q", target.Expect)
		}
		if target.Type == "visual" && target.Threshold > 0 {
			fmt.Printf(" | Threshold: %.1f%%", target.Threshold)
		}
		fmt.Println()
	}
}
