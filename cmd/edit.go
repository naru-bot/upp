package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/naru-bot/upp/internal/db"
	"github.com/naru-bot/upp/internal/trigger"
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
  upp edit 1 --headers '{"Authorization":"Bearer xxx"}'
  upp edit "My API" --jq '.data.status'
  upp edit "My Site" --trigger-if "contains:error"
  upp edit "My API" --method POST --body '{"query":"health"}'
  upp edit "My Site" --no-follow --accept-status "301"
  upp edit "My Site" --auth-bearer "newtoken"`,
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
	cmd.Flags().String("trigger-if", "", "Conditional trigger rule (e.g. 'contains:text', 'regex:pattern')")
	cmd.Flags().String("jq", "", "jq filter for JSON API responses")
	cmd.Flags().Bool("clear-selector", false, "Clear the CSS selector")
	cmd.Flags().Bool("clear-headers", false, "Clear custom headers")
	cmd.Flags().Bool("clear-expect", false, "Clear expected keyword")
	cmd.Flags().Bool("clear-trigger", false, "Clear the trigger rule")
	cmd.Flags().Bool("clear-jq", false, "Clear the jq filter")
	cmd.Flags().String("method", "", "HTTP method (GET, POST, PUT, PATCH, DELETE, HEAD)")
	cmd.Flags().String("body", "", "Request body (for POST/PUT/PATCH)")
	cmd.Flags().String("auth-basic", "", "Basic auth credentials (user:pass)")
	cmd.Flags().String("auth-bearer", "", "Bearer token for Authorization header")
	cmd.Flags().Bool("no-follow", false, "Don't follow redirects")
	cmd.Flags().Bool("follow", false, "Re-enable following redirects")
	cmd.Flags().String("accept-status", "", "Accepted HTTP status codes (e.g. '200-299,301,404')")
	cmd.Flags().Bool("insecure", false, "Skip TLS certificate verification")
	cmd.Flags().Bool("secure", false, "Re-enable TLS certificate verification")
	cmd.Flags().Bool("clear-method", false, "Reset method to GET")
	cmd.Flags().Bool("clear-body", false, "Clear request body")
	cmd.Flags().Bool("clear-accept-status", false, "Reset to default status acceptance")
	cmd.Flags().StringSlice("tag", nil, "Add tag(s) to the target")
	cmd.Flags().StringSlice("untag", nil, "Remove tag(s) from the target")
	cmd.Flags().Bool("clear-tags", false, "Remove all tags")

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
	if cmd.Flags().Changed("trigger-if") {
		triggerIF, _ := cmd.Flags().GetString("trigger-if")
		rule, err := trigger.ParseShorthand(triggerIF)
		if err != nil {
			exitError(err.Error())
		}
		target.TriggerRule = rule
		changed = true
	}
	if cmd.Flags().Changed("jq") {
		target.JQFilter, _ = cmd.Flags().GetString("jq")
		changed = true
	}
	if v, _ := cmd.Flags().GetBool("clear-trigger"); v {
		target.TriggerRule = ""
		changed = true
	}
	if v, _ := cmd.Flags().GetBool("clear-jq"); v {
		target.JQFilter = ""
		changed = true
	}
	if cmd.Flags().Changed("method") {
		target.Method, _ = cmd.Flags().GetString("method")
		changed = true
	}
	if cmd.Flags().Changed("body") {
		target.Body, _ = cmd.Flags().GetString("body")
		changed = true
	}
	if cmd.Flags().Changed("auth-basic") {
		v, _ := cmd.Flags().GetString("auth-basic")
		h := make(map[string]string)
		if target.Headers != "" {
			json.Unmarshal([]byte(target.Headers), &h)
		}
		h["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(v))
		b, _ := json.Marshal(h)
		target.Headers = string(b)
		changed = true
	}
	if cmd.Flags().Changed("auth-bearer") {
		v, _ := cmd.Flags().GetString("auth-bearer")
		h := make(map[string]string)
		if target.Headers != "" {
			json.Unmarshal([]byte(target.Headers), &h)
		}
		h["Authorization"] = "Bearer " + v
		b, _ := json.Marshal(h)
		target.Headers = string(b)
		changed = true
	}
	if v, _ := cmd.Flags().GetBool("no-follow"); v {
		target.NoFollow = true
		changed = true
	}
	if v, _ := cmd.Flags().GetBool("follow"); v {
		target.NoFollow = false
		changed = true
	}
	if cmd.Flags().Changed("accept-status") {
		target.AcceptStatus, _ = cmd.Flags().GetString("accept-status")
		changed = true
	}
	if v, _ := cmd.Flags().GetBool("insecure"); v {
		target.Insecure = true
		changed = true
	}
	if v, _ := cmd.Flags().GetBool("secure"); v {
		target.Insecure = false
		changed = true
	}
	if v, _ := cmd.Flags().GetBool("clear-method"); v {
		target.Method = ""
		changed = true
	}
	if v, _ := cmd.Flags().GetBool("clear-body"); v {
		target.Body = ""
		changed = true
	}
	if v, _ := cmd.Flags().GetBool("clear-accept-status"); v {
		target.AcceptStatus = ""
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

	// Handle tags (these don't use the changed flag since they're separate table)
	tagsChanged := false
	if tags, _ := cmd.Flags().GetStringSlice("tag"); len(tags) > 0 {
		db.AddTags(target.ID, tags)
		tagsChanged = true
	}
	if untags, _ := cmd.Flags().GetStringSlice("untag"); len(untags) > 0 {
		db.RemoveTags(target.ID, untags)
		tagsChanged = true
	}
	if v, _ := cmd.Flags().GetBool("clear-tags"); v {
		db.ClearTags(target.ID)
		tagsChanged = true
	}

	if !changed && !tagsChanged {
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
		if target.JQFilter != "" {
			fmt.Printf(" | jq: %s", target.JQFilter)
		}
		if target.Method != "" {
			fmt.Printf(" | Method: %s", target.Method)
		}
		if target.NoFollow {
			fmt.Printf(" | No-Follow")
		}
		if target.AcceptStatus != "" {
			fmt.Printf(" | Accept: %s", target.AcceptStatus)
		}
		if target.Insecure {
			fmt.Printf(" | Insecure")
		}
		if target.TriggerRule != "" {
			fmt.Printf(" | Trigger: %s", trigger.Describe(target.TriggerRule))
		}
		if tags, _ := db.GetTags(target.ID); len(tags) > 0 {
			fmt.Printf(" | Tags: %s", strings.Join(tags, ", "))
		}
		fmt.Println()
	}
}
