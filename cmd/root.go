package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"

	"github.com/naru-bot/upp/internal/config"
	"github.com/naru-bot/upp/internal/db"
	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
	noColor    bool
	verbose    bool
	quiet      bool
)

var rootCmd = &cobra.Command{
	Use:   "upp",
	Short: "Website uptime monitoring & change detection CLI",
	Long: `Upp — A Swiss Army knife for website monitoring.

Combines uptime monitoring (like Uptime Kuma) and change detection
(like changedetection.io) in a single, lightweight CLI tool.

Designed for both humans and AI agents. Use --json for structured output.

Getting started:
  upp add https://example.com --name "My Site"
  upp check
  upp status

Documentation: https://github.com/naru-bot/upp`,
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
				fmt.Printf("upp %s (%s/%s)\n", Version, runtime.GOOS, runtime.GOARCH)
			}
		},
	})

	// Shell completion generation
	rootCmd.AddCommand(&cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for upp.

To load completions:

Bash:
  $ source <(upp completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ upp completion bash > /etc/bash_completion.d/upp
  # macOS:
  $ upp completion bash > $(brew --prefix)/etc/bash_completion.d/upp

Zsh:
  $ source <(upp completion zsh)
  # To load completions for each session, execute once:
  $ upp completion zsh > "${fpath[1]}/_upp"

Fish:
  $ upp completion fish | source
  # To load completions for each session, execute once:
  $ upp completion fish > ~/.config/fish/completions/upp.fish

PowerShell:
  PS> upp completion powershell | Out-String | Invoke-Expression`,
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

// requireArgs returns a cobra.PositionalArgs that shows helpful usage when args are missing.
func requireArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) < n {
			fmt.Fprintf(os.Stderr, "Usage: %s\n", cmd.UseLine())
			if cmd.Example != "" {
				fmt.Fprintf(os.Stderr, "\nExamples:\n%s\n", cmd.Example)
			} else if cmd.Long != "" {
				// Try to extract examples from Long description
				if idx := findExamplesInLong(cmd.Long); idx != "" {
					fmt.Fprintf(os.Stderr, "\nExamples:\n%s\n", idx)
				}
			}
			return fmt.Errorf("requires %d argument(s), see usage above", n)
		}
		if len(args) > n {
			return fmt.Errorf("accepts %d argument(s), received %d", n, len(args))
		}
		return nil
	}
}

// findExamplesInLong extracts example lines from a Long description.
func findExamplesInLong(long string) string {
	lines := splitLines(long)
	inExamples := false
	var examples []string
	for _, line := range lines {
		trimmed := trimString(line)
		if trimmed == "Examples:" {
			inExamples = true
			continue
		}
		if inExamples {
			if trimmed == "" && len(examples) > 0 {
				break
			}
			if len(trimmed) > 0 {
				examples = append(examples, line)
			}
		}
	}
	if len(examples) == 0 {
		return ""
	}
	result := ""
	for i, e := range examples {
		if i > 0 {
			result += "\n"
		}
		result += e
	}
	return result
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func trimString(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

// end of file
