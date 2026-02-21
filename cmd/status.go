package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"

	"github.com/naru-bot/upp/internal/db"
	"github.com/spf13/cobra"
)

// Available columns for status output
var availableColumns = []string{
	"name", "url", "type", "tags", "uptime", "avg", "min", "max",
	"checks", "changes", "trend", "status", "last_checked", "interval",
}

var defaultColumns = []string{
	"name", "type", "uptime", "avg", "checks", "changes", "trend", "status",
}

func init() {
	cmd := &cobra.Command{
		Use:   "status [name|url|id]",
		Short: "Show uptime statistics and status summary",
		Long: `Display uptime percentage, average response time, and recent status.

Without arguments, shows summary for all targets.

Customize columns with --columns (comma-separated):
  name, url, type, tags, uptime, avg, min, max,
  checks, changes, trend, status, last_checked, interval

Examples:
  upp status
  upp status "My Site"
  upp status --period 7d
  upp status --tag my-sites
  upp status --columns name,uptime,avg,status
  upp status --columns name,url,tags,uptime,trend,status
  upp status --columns all`,
		Run: runStatus,
	}
	cmd.Flags().StringP("period", "p", "24h", "Stats period: 1h, 24h, 7d, 30d")
	cmd.Flags().String("tag", "", "Filter targets by tag")
	cmd.Flags().String("columns", "", "Columns to display (comma-separated, or 'all')")
	rootCmd.AddCommand(cmd)
}

type statusOutput struct {
	Target        string  `json:"target"`
	URL           string  `json:"url"`
	Type          string  `json:"type"`
	Tags          string  `json:"tags,omitempty"`
	UptimePercent float64 `json:"uptime_percent"`
	AvgResponseMs float64 `json:"avg_response_ms"`
	MinResponseMs int64   `json:"min_response_ms"`
	MaxResponseMs int64   `json:"max_response_ms"`
	TotalChecks   int     `json:"total_checks"`
	LastStatus    string  `json:"last_status"`
	LastError     string  `json:"last_error,omitempty"`
	LastChecked   string  `json:"last_checked,omitempty"`
	Changes       int     `json:"content_changes"`
	Sparkline     string  `json:"sparkline,omitempty"`
	Interval      int     `json:"interval_seconds"`
}

func parseColumns(input string) []string {
	if input == "" {
		return defaultColumns
	}
	if input == "all" {
		return availableColumns
	}
	parts := strings.Split(input, ",")
	var cols []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// Validate
		valid := false
		for _, a := range availableColumns {
			if p == a {
				valid = true
				break
			}
		}
		if valid {
			cols = append(cols, p)
		} else {
			fmt.Fprintf(os.Stderr, "Warning: unknown column %q (available: %s)\n", p, strings.Join(availableColumns, ", "))
		}
	}
	if len(cols) == 0 {
		return defaultColumns
	}
	return cols
}

func columnHeader(col string) string {
	switch col {
	case "name":
		return "TARGET"
	case "url":
		return "URL"
	case "type":
		return "TYPE"
	case "tags":
		return "TAGS"
	case "uptime":
		return "UPTIME"
	case "avg":
		return "AVG RESP"
	case "min":
		return "MIN RESP"
	case "max":
		return "MAX RESP"
	case "checks":
		return "CHECKS"
	case "changes":
		return "CHANGES"
	case "trend":
		return "TREND"
	case "status":
		return "STATUS"
	case "last_checked":
		return "LAST CHECKED"
	case "interval":
		return "INTERVAL"
	default:
		return strings.ToUpper(col)
	}
}

func columnSeparator(col string) string {
	h := columnHeader(col)
	return strings.Repeat("─", len(h))
}

func columnValue(col string, o *statusOutput) string {
	switch col {
	case "name":
		return truncate(o.Target, 25)
	case "url":
		return truncate(o.URL, 40)
	case "type":
		return o.Type
	case "tags":
		return truncate(o.Tags, 20)
	case "uptime":
		s := fmt.Sprintf("%.1f%%", o.UptimePercent)
		if !noColor && !jsonOutput {
			if o.UptimePercent >= 99.9 {
				s = colorGreen(s)
			} else if o.UptimePercent >= 95 {
				s = colorYellow(s)
			} else if o.TotalChecks > 0 {
				s = colorRed(s)
			}
		}
		return s
	case "avg":
		return fmt.Sprintf("%.0fms", o.AvgResponseMs)
	case "min":
		return fmt.Sprintf("%dms", o.MinResponseMs)
	case "max":
		return fmt.Sprintf("%dms", o.MaxResponseMs)
	case "checks":
		return fmt.Sprintf("%d", o.TotalChecks)
	case "changes":
		return fmt.Sprintf("%d", o.Changes)
	case "trend":
		return o.Sparkline
	case "status":
		s := o.LastStatus
		if !noColor && !jsonOutput {
			switch o.LastStatus {
			case "up", "unchanged":
				s = colorGreen("● " + o.LastStatus)
			case "changed":
				s = colorYellow("△ " + o.LastStatus)
			case "down", "error":
				s = colorRed("✗ " + o.LastStatus)
			}
		}
		if o.LastError != "" && (o.LastStatus == "down" || o.LastStatus == "error") {
			shortErr := shortenError(o.LastError)
			if !noColor && !jsonOutput {
				shortErr = colorRed(shortErr)
			}
			s += " " + shortErr
		}
		return s
	case "last_checked":
		if o.LastChecked == "" {
			return "—"
		}
		if t, err := time.Parse(time.RFC3339, o.LastChecked); err == nil {
			return time.Since(t).Round(time.Second).String() + " ago"
		}
		return o.LastChecked
	case "interval":
		return fmt.Sprintf("%ds", o.Interval)
	default:
		return ""
	}
}

func runStatus(cmd *cobra.Command, args []string) {
	period, _ := cmd.Flags().GetString("period")
	since := parsePeriod(period)
	tag, _ := cmd.Flags().GetString("tag")
	colStr, _ := cmd.Flags().GetString("columns")
	cols := parseColumns(colStr)

	var targets []db.Target
	if len(args) > 0 {
		t, err := db.GetTarget(args[0])
		if err != nil {
			exitError(err.Error())
		}
		targets = []db.Target{*t}
	} else if tag != "" {
		var err error
		targets, err = db.ListTargetsByTag(tag)
		if err != nil {
			exitError(err.Error())
		}
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
			fmt.Println("No targets configured.")
		}
		return
	}

	// Load tags if needed
	var tagMap map[int64][]string
	for _, c := range cols {
		if c == "tags" {
			tagMap, _ = db.GetTagMap()
			break
		}
	}

	var outputs []statusOutput

	for _, t := range targets {
		total, up, avgMs, err := db.GetUptimeStats(t.ID, since)
		if err != nil {
			continue
		}

		var uptimePct float64
		if total > 0 {
			uptimePct = float64(up) / float64(total) * 100
		}

		results, _ := db.GetCheckHistory(t.ID, 1000)
		changes := 0
		lastStatus := "unknown"
		lastError := ""
		lastChecked := ""
		var minMs, maxMs int64
		var responseTimes []int64
		for i, r := range results {
			if i == 0 {
				lastStatus = r.Status
				lastError = r.Error
				lastChecked = r.CheckedAt.Format(time.RFC3339)
				minMs = r.ResponseTime
				maxMs = r.ResponseTime
			}
			if r.Status == "changed" {
				changes++
			}
			if r.ResponseTime < minMs {
				minMs = r.ResponseTime
			}
			if r.ResponseTime > maxMs {
				maxMs = r.ResponseTime
			}
			responseTimes = append(responseTimes, r.ResponseTime)
		}

		spark := buildSparkline(responseTimes, 20)

		tags := ""
		if tagMap != nil {
			if tt, ok := tagMap[t.ID]; ok {
				tags = strings.Join(tt, ",")
			}
		}

		out := statusOutput{
			Target:        t.Name,
			URL:           t.URL,
			Type:          t.Type,
			Tags:          tags,
			UptimePercent: uptimePct,
			AvgResponseMs: avgMs,
			MinResponseMs: minMs,
			MaxResponseMs: maxMs,
			TotalChecks:   total,
			LastStatus:    lastStatus,
			LastError:     lastError,
			LastChecked:   lastChecked,
			Changes:       changes,
			Sparkline:     spark,
			Interval:      t.Interval,
		}
		outputs = append(outputs, out)
	}

	if jsonOutput {
		printJSON(outputs)
		return
	}

	// Build all cell values first to compute column widths
	// Use visible length (stripping ANSI) for alignment
	ansiRe := regexp.MustCompile(`\x1b\[[0-9;]*m`)

	numCols := len(cols)
	// Header row + separator + data rows
	allRows := make([][]string, 0, len(outputs)+2)

	// Header
	headerRow := make([]string, numCols)
	for i, c := range cols {
		headerRow[i] = columnHeader(c)
	}
	allRows = append(allRows, headerRow)

	// Data rows
	for i := range outputs {
		row := make([]string, numCols)
		for j, c := range cols {
			row[j] = columnValue(c, &outputs[i])
		}
		allRows = append(allRows, row)
	}

	// Compute max visible width per column (using display width, not byte count)
	colWidths := make([]int, numCols)
	for _, row := range allRows {
		for j, cell := range row {
			visible := runewidth.StringWidth(ansiRe.ReplaceAllString(cell, ""))
			if visible > colWidths[j] {
				colWidths[j] = visible
			}
		}
	}

	// Print header
	printPaddedRow(os.Stdout, allRows[0], colWidths, ansiRe)
	// Separator (─ is 1 display width)
	sepRow := make([]string, numCols)
	for i, w := range colWidths {
		sepRow[i] = strings.Repeat("─", w)
	}
	// Separator uses single-width chars so it aligns correctly
	printPaddedRow(os.Stdout, sepRow, colWidths, ansiRe)
	// Data
	for _, row := range allRows[1:] {
		printPaddedRow(os.Stdout, row, colWidths, ansiRe)
	}
}

func printPaddedRow(w *os.File, row []string, widths []int, ansiRe *regexp.Regexp) {
	for i, cell := range row {
		visLen := runewidth.StringWidth(ansiRe.ReplaceAllString(cell, ""))
		pad := widths[i] - visLen
		if pad < 0 {
			pad = 0
		}
		if i > 0 {
			fmt.Fprint(w, "  ")
		}
		fmt.Fprint(w, cell)
		if i < len(row)-1 {
			fmt.Fprint(w, strings.Repeat(" ", pad))
		}
	}
	fmt.Fprintln(w)
}

func shortenError(err string) string {
	if idx := strings.LastIndex(err, ": "); idx != -1 {
		err = err[idx+2:]
	}
	if len(err) > 40 {
		err = err[:37] + "..."
	}
	return "(" + err + ")"
}

func buildSparkline(values []int64, maxLen int) string {
	if len(values) == 0 {
		return ""
	}

	reversed := make([]int64, len(values))
	for i, v := range values {
		reversed[len(values)-1-i] = v
	}

	if len(reversed) > maxLen {
		reversed = reversed[len(reversed)-maxLen:]
	}

	min, max := reversed[0], reversed[0]
	for _, v := range reversed {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	blocks := []rune("▁▂▃▄▅▆▇█")
	spread := max - min
	if spread == 0 {
		spread = 1
	}

	var result []rune
	for _, v := range reversed {
		idx := int(float64(v-min) / float64(spread) * float64(len(blocks)-1))
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}
		result = append(result, blocks[idx])
	}
	return string(result)
}

func parsePeriod(p string) time.Time {
	now := time.Now()
	switch p {
	case "1h":
		return now.Add(-1 * time.Hour)
	case "7d":
		return now.Add(-7 * 24 * time.Hour)
	case "30d":
		return now.Add(-30 * 24 * time.Hour)
	default:
		return now.Add(-24 * time.Hour)
	}
}
