package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"

	"github.com/naru-bot/watchdog/internal/config"
	"github.com/naru-bot/watchdog/internal/db"
	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
	noColor    bool
	verbose    bool
	quiet      bool
)

var rootCmd = &cobra.Command{
	Use:   "watchdog",
	Short: "Website uptime monitoring & change detection CLI",
	Long: `Watchdog — A Swiss Army knife for website monitoring.

Combines uptime monitoring (like Uptime Kuma) and change detection
(like changedetection.io) in a single, lightweight CLI tool.

Designed for both humans and AI agents. Use --json for structured output.

Getting started:
  watchdog add https://example.com --name "My Site"
  watchdog check
  watchdog status

Documentation: https://github.com/naru-bot/watchdog`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip DB init for commands that don't need it
		switch cmd.Name() {
		case "version", "completion", "init":
			return nil
		}
		config.Load()
		return db.Init()
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if jsonOutput {
			printJSON(map[string]string{"error": err.Error()})
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		}
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format (AI-friendly)")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-essential output")
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

// Color helpers
func colorRed(s string) string {
	if noColor || jsonOutput {
		return s
	}
	return "\033[31m" + s + "\033[0m"
}

func colorGreen(s string) string {
	if noColor || jsonOutput {
		return s
	}
	return "\033[32m" + s + "\033[0m"
}

func colorYellow(s string) string {
	if noColor || jsonOutput {
		return s
	}
	return "\033[33m" + s + "\033[0m"
}

func colorCyan(s string) string {
	if noColor || jsonOutput {
		return s
	}
	return "\033[36m" + s + "\033[0m"
}

func colorBold(s string) string {
	if noColor || jsonOutput {
		return s
	}
	return "\033[1m" + s + "\033[0m"
}

var Version = "dev"

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			if jsonOutput {
				printJSON(map[string]string{"version": Version, "platform": fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)})
			} else {
				fmt.Printf("watchdog %s (%s/%s)\n", Version, runtime.GOOS, runtime.GOARCH)
			}
		},
	})

	// Shell completion generation
	rootCmd.AddCommand(&cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for watchdog.

To load completions:

Bash:
  $ source <(watchdog completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ watchdog completion bash > /etc/bash_completion.d/watchdog
  # macOS:
  $ watchdog completion bash > $(brew --prefix)/etc/bash_completion.d/watchdog

Zsh:
  $ source <(watchdog completion zsh)
  # To load completions for each session, execute once:
  $ watchdog completion zsh > "${fpath[1]}/_watchdog"

Fish:
  $ watchdog completion fish | source
  # To load completions for each session, execute once:
  $ watchdog completion fish > ~/.config/fish/completions/watchdog.fish

PowerShell:
  PS> watchdog completion powershell | Out-String | Invoke-Expression`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			}
		},
	})
}

// end of file
