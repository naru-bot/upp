package cmd

import (
	"fmt"

	"github.com/cheryeong/watchdog/internal/db"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "remove <name|url|id>",
		Short: "Remove a monitored target",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			err := db.RemoveTarget(args[0])
			if err != nil {
				exitError(err.Error())
			}
			if jsonOutput {
				printJSON(map[string]string{"status": "removed", "target": args[0]})
			} else {
				fmt.Printf("✓ Removed: %s\n", args[0])
			}
		},
	})
}
