package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/naru-bot/upp/internal/db"
	"github.com/naru-bot/upp/internal/trigger"
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
  upp add https://example.com --type visual --threshold 7.5
  upp add https://example.com --trigger-if "contains:out of stock"
  upp add https://example.com --trigger-if "not_contains:in stock"
  upp add https://example.com --trigger-if "regex:price.*\$[0-9]+"
  upp add https://api.example.com/data --jq '.items[].name'
  upp add https://api.example.com/v1/status --jq '.status' --trigger-if "not_contains:healthy"
  upp add https://api.example.com/data --method POST --body '{"query":"health"}'
  upp add https://example.com --auth-bearer "token123"
  upp add https://example.com --auth-basic "user:pass"
  upp add https://example.com --no-follow --accept-status "301"
  upp add https://internal.example.com --insecure`,
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
	cmd.Flags().String("trigger-if", "", "Conditional trigger rule (e.g. 'contains:text', 'regex:pattern')")
	cmd.Flags().String("jq", "", "jq filter for JSON API responses")
	cmd.Flags().String("method", "", "HTTP method (GET, POST, PUT, PATCH, DELETE, HEAD)")
	cmd.Flags().String("body", "", "Request body (for POST/PUT/PATCH)")
	cmd.Flags().String("auth-basic", "", "Basic auth credentials (user:pass)")
	cmd.Flags().String("auth-bearer", "", "Bearer token for Authorization header")
	cmd.Flags().Bool("no-follow", false, "Don't follow redirects")
	cmd.Flags().String("accept-status", "", "Accepted HTTP status codes (e.g. '200-299,301,404')")
	cmd.Flags().Bool("insecure", false, "Skip TLS certificate verification")

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
	triggerIF, _ := cmd.Flags().GetString("trigger-if")
	jqFilter, _ := cmd.Flags().GetString("jq")

	method, _ := cmd.Flags().GetString("method")
	body, _ := cmd.Flags().GetString("body")
	authBasic, _ := cmd.Flags().GetString("auth-basic")
	authBearer, _ := cmd.Flags().GetString("auth-bearer")
	noFollow, _ := cmd.Flags().GetBool("no-follow")
	acceptStatus, _ := cmd.Flags().GetString("accept-status")
	insecure, _ := cmd.Flags().GetBool("insecure")

	// Parse trigger rule shorthand
	var triggerRule string
	if triggerIF != "" {
		rule, err := trigger.ParseShorthand(triggerIF)
		if err != nil {
			exitError(err.Error())
		}
		triggerRule = rule
	}

	// Apply auth shortcuts to headers
	headers = applyAuth(headers, authBasic, authBearer)

	opts := db.AddTargetOpts{
		TriggerRule:  triggerRule,
		JQFilter:     jqFilter,
		Method:       method,
		Body:         body,
		NoFollow:     noFollow,
		AcceptStatus: acceptStatus,
		Insecure:     insecure,
	}

	target, err := db.AddTarget(name, url, typ, interval, selector, headers, expect, timeout, retries, threshold, opts)
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
		if target.JQFilter != "" {
			fmt.Printf(" | jq: %s", target.JQFilter)
		}
		if target.Method != "" {
			fmt.Printf(" | Method: %s", target.Method)
		}
		if target.Body != "" {
			fmt.Printf(" | Body: %s", truncateStr(target.Body, 40))
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
		fmt.Println()
	}
}

// applyAuth merges auth shortcuts into the headers JSON string.
func applyAuth(headers, authBasic, authBearer string) string {
	if authBasic == "" && authBearer == "" {
		return headers
	}
	h := make(map[string]string)
	if headers != "" {
		json.Unmarshal([]byte(headers), &h)
	}
	if authBasic != "" {
		encoded := base64.StdEncoding.EncodeToString([]byte(authBasic))
		h["Authorization"] = "Basic " + encoded
	}
	if authBearer != "" {
		h["Authorization"] = "Bearer " + authBearer
	}
	b, _ := json.Marshal(h)
	return string(b)
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

