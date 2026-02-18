package cmd

import (
	"fmt"

	"github.com/cheryeong/watchdog/internal/db"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "add <url>",
		Short: "Add a URL to monitor",
		Long: `Add a URL for uptime monitoring and change detection.

Examples:
  watchdog add https://example.com
  watchdog add https://example.com --name "My Site" --interval 60
  watchdog add https://example.com --selector "div.price" --name "Price Watch"
  watchdog add 192.168.1.1:3306 --type tcp --name "MySQL"
  watchdog add example.com --type ping
  watchdog add example.com --type dns`,
		Args: cobra.ExactArgs(1),
		Run:  runAdd,
	}

	cmd.Flags().StringP("name", "n", "", "Friendly name for the target")
	cmd.Flags().StringP("type", "t", "http", "Check type: http, tcp, ping, dns")
	cmd.Flags().IntP("interval", "i", 300, "Check interval in seconds")
	cmd.Flags().StringP("selector", "s", "", "CSS selector for change detection")
	cmd.Flags().String("headers", "", "Custom headers as JSON string")

	rootCmd.AddCommand(cmd)
}

func runAdd(cmd *cobra.Command, args []string) {
	url := args[0]
	name, _ := cmd.Flags().GetString("name")
	typ, _ := cmd.Flags().GetString("type")
	interval, _ := cmd.Flags().GetInt("interval")
	selector, _ := cmd.Flags().GetString("selector")
	headers, _ := cmd.Flags().GetString("headers")

	target, err := db.AddTarget(name, url, typ, interval, selector, headers)
	if err != nil {
		exitError(err.Error())
	}

	if jsonOutput {
		printJSON(target)
	} else {
		fmt.Printf("✓ Added: %s (%s)\n", target.Name, target.URL)
		fmt.Printf("  Type: %s | Interval: %ds", target.Type, target.Interval)
		if target.Selector != "" {
			fmt.Printf(" | Selector: %s", target.Selector)
		}
		fmt.Println()
	}
}
