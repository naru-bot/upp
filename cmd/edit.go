package cmd

import (
	"fmt"

	"github.com/naru-bot/upp/internal/db"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "edit <name|url|id>",
		Short: "Edit a monitored target",
		Long: `Edit properties of an existing monitored target.

Only specified flags are updated; unset flags are left unchanged.

Examples:
  upp edit "My Site" --name "New Name"
  upp edit 1 --url https://new-url.com
  upp edit "My Site" --interval 60 --timeout 10
  upp edit 1 --selector "div.content" --expect "Welcome"
  upp edit "My Site" --retries 3 --type tcp
  upp edit 1 --headers '{"Authorization":"Bearer xxx"}'`,
		Args: requireArgs(1),
		Run:  runEdit,
	}

	cmd.Flags().StringP("name", "n", "", "New name for the target")
	cmd.Flags().String("url", "", "New URL to monitor")
	cmd.Flags().StringP("type", "t", "", "Check type: http, tcp, ping, dns")
	cmd.Flags().IntP("interval", "i", 0, "Check interval in seconds")
	cmd.Flags().StringP("selector", "s", "", "CSS selector for change detection")
	cmd.Flags().String("headers", "", "Custom headers as JSON string")
	cmd.Flags().String("expect", "", "Expected keyword in response body")
	cmd.Flags().Int("timeout", 0, "Request timeout in seconds")
	cmd.Flags().Int("retries", 0, "Retry count before marking as down")
	cmd.Flags().Bool("clear-selector", false, "Clear the CSS selector")
	cmd.Flags().Bool("clear-headers", false, "Clear custom headers")
	cmd.Flags().Bool("clear-expect", false, "Clear expected keyword")

	rootCmd.AddCommand(cmd)
}

func runEdit(cmd *cobra.Command, args []string) {
	target, err := db.GetTarget(args[0])
	if err != nil {
		exitError(err.Error())
	}

	changed := false

	if cmd.Flags().Changed("name") {
		target.Name, _ = cmd.Flags().GetString("name")
		changed = true
	}
	if cmd.Flags().Changed("url") {
		target.URL, _ = cmd.Flags().GetString("url")
		changed = true
	}
	if cmd.Flags().Changed("type") {
		target.Type, _ = cmd.Flags().GetString("type")
		changed = true
	}
	if cmd.Flags().Changed("interval") {
		target.Interval, _ = cmd.Flags().GetInt("interval")
		changed = true
	}
	if cmd.Flags().Changed("selector") {
		target.Selector, _ = cmd.Flags().GetString("selector")
		changed = true
	}
	if cmd.Flags().Changed("headers") {
		target.Headers, _ = cmd.Flags().GetString("headers")
		changed = true
	}
	if cmd.Flags().Changed("expect") {
		target.Expect, _ = cmd.Flags().GetString("expect")
		changed = true
	}
	if cmd.Flags().Changed("timeout") {
		target.Timeout, _ = cmd.Flags().GetInt("timeout")
		changed = true
	}
	if cmd.Flags().Changed("retries") {
		target.Retries, _ = cmd.Flags().GetInt("retries")
		changed = true
	}
	if v, _ := cmd.Flags().GetBool("clear-selector"); v {
		target.Selector = ""
		changed = true
	}
	if v, _ := cmd.Flags().GetBool("clear-headers"); v {
		target.Headers = ""
		changed = true
	}
	if v, _ := cmd.Flags().GetBool("clear-expect"); v {
		target.Expect = ""
		changed = true
	}

	if !changed {
		exitError("nothing to update — specify at least one flag (see upp edit --help)")
	}

	if err := db.UpdateTarget(target); err != nil {
		exitError(err.Error())
	}

	if jsonOutput {
		printJSON(target)
	} else {
		fmt.Printf("✓ Updated: %s (%s)\n", target.Name, target.URL)
		fmt.Printf("  Type: %s | Interval: %ds | Timeout: %ds | Retries: %d", target.Type, target.Interval, target.Timeout, target.Retries)
		if target.Selector != "" {
			fmt.Printf(" | Selector: %s", target.Selector)
		}
		if target.Expect != "" {
			fmt.Printf(" | Expect: %q", target.Expect)
		}
		fmt.Println()
	}
}
