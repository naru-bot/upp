package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/naru-bot/upp/internal/db"
	"github.com/spf13/cobra"
)

func init() {
	notifyCmd := &cobra.Command{
		Use:   "notify",
		Short: "Manage notification channels",
	}

	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a notification channel",
		Long: `Add a notification channel for alerts.

Examples:
  upp notify add --name alerts --type webhook --config '{"url":"https://hooks.slack.com/..."}'
  upp notify add --name telegram --type telegram --config '{"bot_token":"...","chat_id":"..."}'
  upp notify add --name discord --type discord --config '{"webhook_url":"..."}'
  upp notify add --name runner --type command --config '{"command":"echo {target} is {status}"}'`,
		Run: runNotifyAdd,
	}
	addCmd.Flags().String("name", "", "Name for this notification channel")
	addCmd.Flags().String("type", "", "Type: webhook, command, slack, telegram, discord")
	addCmd.Flags().String("config", "", "JSON configuration for the channel")
	addCmd.MarkFlagRequired("name")
	addCmd.MarkFlagRequired("type")
	addCmd.MarkFlagRequired("config")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List notification channels",
		Aliases: []string{"ls"},
		Run: func(cmd *cobra.Command, args []string) {
			configs, err := db.ListNotifyConfigs()
			if err != nil {
				exitError(err.Error())
			}
			if jsonOutput {
				printJSON(configs)
				return
			}
			if len(configs) == 0 {
				fmt.Println("No notification channels configured.")
				return
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "ID\tNAME\tTYPE\tENABLED\n")
			for _, c := range configs {
				fmt.Fprintf(w, "%d\t%s\t%s\t%v\n", c.ID, c.Name, c.Type, c.Enabled)
			}
			w.Flush()
		},
	}

	removeCmd := &cobra.Command{
		Use:   "remove <name|id>",
		Short: "Remove a notification channel",
		Args:  requireArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			err := db.RemoveNotifyConfig(args[0])
			if err != nil {
				exitError(err.Error())
			}
			if jsonOutput {
				printJSON(map[string]string{"status": "removed"})
			} else {
				fmt.Printf("✓ Removed notification channel: %s\n", args[0])
			}
		},
	}

	notifyCmd.AddCommand(addCmd, listCmd, removeCmd)
	rootCmd.AddCommand(notifyCmd)
}

func runNotifyAdd(cmd *cobra.Command, args []string) {
	name, _ := cmd.Flags().GetString("name")
	typ, _ := cmd.Flags().GetString("type")
	config, _ := cmd.Flags().GetString("config")

	// Validate JSON
	var js json.RawMessage
	if err := json.Unmarshal([]byte(config), &js); err != nil {
		exitError("Invalid JSON config: " + err.Error())
	}

	if err := db.SaveNotifyConfig(name, typ, config); err != nil {
		exitError(err.Error())
	}

	if jsonOutput {
		printJSON(map[string]string{"status": "added", "name": name, "type": typ})
	} else {
		fmt.Printf("✓ Added notification channel: %s (%s)\n", name, typ)
	}
}
