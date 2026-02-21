package cmd

import (
	"fmt"
	"os"

	"github.com/naru-bot/upp/internal/db"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "import <file.yml>",
		Short: "Import targets from a YAML file",
		Long: `Bulk import monitoring targets from a YAML configuration file.

YAML format:
  targets:
    - name: My Site
      url: https://example.com
      type: http
      interval: 60
      selector: "div.content"
      expect: "Welcome"
      timeout: 10
      retries: 3
    - name: MySQL
      url: 192.168.1.1:3306
      type: tcp

Examples:
  upp import targets.yml
  upp import targets.yml --json`,
		Args: requireArgs(1),
		Run:  runImport,
	})
}

type importFile struct {
	Targets []importTarget `yaml:"targets"`
}

type importTarget struct {
	Name      string  `yaml:"name"`
	URL       string  `yaml:"url"`
	Type      string  `yaml:"type"`
	Interval  int     `yaml:"interval"`
	Selector  string  `yaml:"selector"`
	Headers   string  `yaml:"headers"`
	Expect    string  `yaml:"expect"`
	Timeout   int     `yaml:"timeout"`
	Retries   int     `yaml:"retries"`
	Threshold   float64 `yaml:"threshold"`
	TriggerRule   string  `yaml:"trigger_rule"`
	JQFilter      string  `yaml:"jq_filter"`
	Method        string  `yaml:"method"`
	Body          string  `yaml:"body"`
	NoFollow      bool    `yaml:"no_follow"`
	AcceptStatus  string  `yaml:"accept_status"`
	Insecure      bool    `yaml:"insecure"`
}

func runImport(cmd *cobra.Command, args []string) {
	data, err := os.ReadFile(args[0])
	if err != nil {
		exitError("failed to read file: " + err.Error())
	}

	var imp importFile
	if err := yaml.Unmarshal(data, &imp); err != nil {
		exitError("failed to parse YAML: " + err.Error())
	}

	if len(imp.Targets) == 0 {
		exitError("no targets found in file")
	}

	type result struct {
		Name   string `json:"name"`
		URL    string `json:"url"`
		Status string `json:"status"`
		Error  string `json:"error,omitempty"`
	}

	var results []result
	added := 0

	for _, t := range imp.Targets {
		if t.URL == "" {
			continue
		}
		if t.Type == "" {
			t.Type = "http"
		}
		if t.Interval <= 0 {
			t.Interval = 300
		}
		if t.Timeout <= 0 {
			t.Timeout = 30
		}
		if t.Retries <= 0 {
			t.Retries = 1
		}
		if t.Threshold <= 0 {
			t.Threshold = 5.0
		}

		_, err := db.AddTarget(t.Name, t.URL, t.Type, t.Interval, t.Selector, t.Headers, t.Expect, t.Timeout, t.Retries, t.Threshold, db.AddTargetOpts{
				TriggerRule: t.TriggerRule, JQFilter: t.JQFilter, Method: t.Method, Body: t.Body, NoFollow: t.NoFollow, AcceptStatus: t.AcceptStatus, Insecure: t.Insecure,
			})
		r := result{Name: t.Name, URL: t.URL}
		if err != nil {
			r.Status = "error"
			r.Error = err.Error()
		} else {
			r.Status = "added"
			added++
		}
		results = append(results, r)

		if !jsonOutput {
			if r.Error != "" {
				fmt.Printf("  %s %s — %s\n", colorRed("✗"), t.Name, r.Error)
			} else {
				fmt.Printf("  %s %s (%s)\n", colorGreen("✓"), t.Name, t.URL)
			}
		}
	}

	if jsonOutput {
		printJSON(map[string]interface{}{
			"imported": added,
			"total":    len(imp.Targets),
			"results":  results,
		})
	} else {
		fmt.Printf("\n%s imported, %d total\n", colorBold(fmt.Sprintf("%d", added)), len(imp.Targets))
	}
}
