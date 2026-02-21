package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/naru-bot/upp/internal/db"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all monitored targets",
		Aliases: []string{"ls"},
		Long: `List all monitored targets, optionally filtered by tag.

Examples:
  upp list
  upp list --tag my-sites
  upp list --tags           # list all tags with counts`,
		Run: runList,
	}
	cmd.Flags().String("tag", "", "Filter targets by tag")
	cmd.Flags().Bool("tags", false, "List all tags with target counts")
	rootCmd.AddCommand(cmd)
}

func runList(cmd *cobra.Command, args []string) {
	// List tags mode
	if showTags, _ := cmd.Flags().GetBool("tags"); showTags {
		listTags()
		return
	}

	tag, _ := cmd.Flags().GetString("tag")
	var targets []db.Target
	var err error
	if tag != "" {
		targets, err = db.ListTargetsByTag(tag)
	} else {
		targets, err = db.ListTargets()
	}
	if err != nil {
		exitError(err.Error())
	}

	if jsonOutput {
		printJSON(targets)
		return
	}

	if len(targets) == 0 {
		if tag != "" {
			fmt.Printf("No targets with tag %q. Use 'upp list --tags' to see all tags.\n", tag)
		} else {
			fmt.Println("No targets configured. Use 'upp add <url>' to start monitoring.")
		}
		return
	}

	tagMap, _ := db.GetTagMap()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tNAME\tURL\tTYPE\tINTERVAL\tTAGS\tSTATUS\n")
	fmt.Fprintf(w, "──\t────\t───\t────\t────────\t────\t──────\n")

	for _, t := range targets {
		status := "active"
		if t.Paused {
			status = "paused"
		}
		results, err := db.GetCheckHistory(t.ID, 1)
		if err == nil && len(results) > 0 {
			last := results[0]
			age := time.Since(last.CheckedAt).Round(time.Second)
			status = fmt.Sprintf("%s (%s ago)", last.Status, age)
		}

		tags := ""
		if tt, ok := tagMap[t.ID]; ok {
			tags = strings.Join(tt, ",")
		}

		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%ds\t%s\t%s\n",
			t.ID, t.Name, truncate(t.URL, 40), t.Type, t.Interval, tags, status)
	}
	w.Flush()
}

func listTags() {
	tags, err := db.ListAllTags()
	if err != nil {
		exitError(err.Error())
	}
	if len(tags) == 0 {
		fmt.Println("No tags defined. Use 'upp add <url> --tag <tag>' or 'upp edit <target> --tag <tag>'.")
		return
	}

	if jsonOutput {
		// Build tag -> count map
		type tagInfo struct {
			Tag   string `json:"tag"`
			Count int    `json:"count"`
		}
		var info []tagInfo
		for _, tag := range tags {
			targets, _ := db.ListTargetsByTag(tag)
			info = append(info, tagInfo{Tag: tag, Count: len(targets)})
		}
		printJSON(info)
		return
	}

	fmt.Println("Tags:")
	for _, tag := range tags {
		targets, _ := db.ListTargetsByTag(tag)
		fmt.Printf("  %-20s %d targets\n", tag, len(targets))
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
