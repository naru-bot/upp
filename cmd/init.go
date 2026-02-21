package cmd

import (
	"fmt"

	"github.com/naru-bot/upp/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Initialize upp configuration",
		Long: `Create a default configuration file at ~/.config/upp/config.yml.

The config file lets you set default intervals, timeouts, display preferences,
and custom headers that apply to all targets.`,
		Run: func(cmd *cobra.Command, args []string) {
			cfg := config.Default()
			if err := config.Save(cfg); err != nil {
				exitError(err.Error())
			}
			if jsonOutput {
				printJSON(map[string]string{"status": "initialized", "config": "~/.config/upp/config.yml"})
			} else {
				fmt.Println("✓ Configuration initialized at ~/.config/upp/config.yml")
				fmt.Println()
				fmt.Println("Default settings:")
				fmt.Printf("  Check interval: %ds\n", cfg.Defaults.Interval)
				fmt.Printf("  Check type:     %s\n", cfg.Defaults.Type)
				fmt.Printf("  HTTP timeout:   %ds\n", cfg.Defaults.Timeout)
				fmt.Printf("  User agent:     %s\n", cfg.Defaults.UserAgent)
				fmt.Println()
				fmt.Println("Edit the config file to customize defaults.")
			}
		},
	})
}
