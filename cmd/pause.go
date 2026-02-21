package cmd

import (
	"fmt"

	"github.com/naru-bot/upp/internal/db"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "pause <name|url|id>",
		Short: "Pause monitoring for a target",
		Args:  requireArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := db.SetPaused(args[0], true); err != nil {
				exitError(err.Error())
			}
			if jsonOutput {
				printJSON(map[string]string{"status": "paused", "target": args[0]})
			} else {
				fmt.Printf("⏸ Paused: %s\n", args[0])
			}
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "unpause <name|url|id>",
		Short: "Resume monitoring for a target",
		Aliases: []string{"resume"},
		Args:  requireArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := db.SetPaused(args[0], false); err != nil {
				exitError(err.Error())
			}
			if jsonOutput {
				printJSON(map[string]string{"status": "resumed", "target": args[0]})
			} else {
				fmt.Printf("▶ Resumed: %s\n", args[0])
			}
		},
	})
}
