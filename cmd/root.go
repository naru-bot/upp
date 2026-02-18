package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cheryeong/watchdog/internal/db"
	"github.com/spf13/cobra"
)

var jsonOutput bool

var rootCmd = &cobra.Command{
	Use:   "watchdog",
	Short: "Website uptime monitoring & change detection CLI",
	Long: `Watchdog — A Swiss Army knife for website monitoring.

Combines uptime monitoring (like Uptime Kuma) and change detection
(like changedetection.io) in a single, lightweight CLI tool.

Designed for both humans and AI agents. Use --json for structured output.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "version" {
			return nil
		}
		return db.Init()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format (AI-friendly)")
}

func printJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func exitError(msg string) {
	if jsonOutput {
		printJSON(map[string]string{"error": msg})
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	}
	os.Exit(1)
}

var Version = "dev"

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			if jsonOutput {
				printJSON(map[string]string{"version": Version})
			} else {
				fmt.Printf("watchdog %s\n", Version)
			}
		},
	})
}
