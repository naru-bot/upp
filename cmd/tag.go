package cmd

import (
	"fmt"
	"strings"

	"github.com/naru-bot/upp/internal/db"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "tag <name|url|id> <tag> [tag...]",
		Short: "Add tags to a target",
		Long: `Add one or more tags to a target for organizing monitors.

Examples:
  upp tag "My Site" my-sites
  upp tag 1 my-sites production
  upp tag https://example.com web`,
		Args: cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			target, err := db.GetTarget(args[0])
			if err != nil {
				exitError(err.Error())
			}
			tags := args[1:]
			if err := db.AddTags(target.ID, tags); err != nil {
				exitError(err.Error())
			}
			allTags, _ := db.GetTags(target.ID)
			if jsonOutput {
				printJSON(map[string]interface{}{"target": target.Name, "tags": allTags})
			} else {
				fmt.Printf("✓ %s — tags: %s\n", target.Name, strings.Join(allTags, ", "))
			}
		},
	}
	rootCmd.AddCommand(cmd)

	untagCmd := &cobra.Command{
		Use:   "untag <name|url|id> <tag> [tag...]",
		Short: "Remove tags from a target",
		Long: `Remove one or more tags from a target.

Examples:
  upp untag "My Site" old-tag
  upp untag 1 staging`,
		Args: cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			target, err := db.GetTarget(args[0])
			if err != nil {
				exitError(err.Error())
			}
			tags := args[1:]
			db.RemoveTags(target.ID, tags)
			allTags, _ := db.GetTags(target.ID)
			if jsonOutput {
				printJSON(map[string]interface{}{"target": target.Name, "tags": allTags})
			} else {
				tagStr := "(none)"
				if len(allTags) > 0 {
					tagStr = strings.Join(allTags, ", ")
				}
				fmt.Printf("✓ %s — tags: %s\n", target.Name, tagStr)
			}
		},
	}
	rootCmd.AddCommand(untagCmd)
}
