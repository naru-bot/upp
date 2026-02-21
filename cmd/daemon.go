package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/naru-bot/upp/internal/checker"
	"github.com/naru-bot/upp/internal/db"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "daemon",
		Short: "Run as background daemon with scheduled checks",
		Long: `Start upp as a long-running process that checks all targets
on their configured intervals.

Examples:
  upp daemon
  upp daemon &           # run in background
  nohup upp daemon &     # survive terminal close`,
		Run: runDaemon,
	})
}

func runDaemon(cmd *cobra.Command, args []string) {
	fmt.Println("🐕 Upp daemon started")
	fmt.Println("Press Ctrl+C to stop")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	lastCheck := make(map[int64]time.Time)

	for {
		select {
		case <-sig:
			fmt.Println("\n🐕 Upp daemon stopped")
			return
		case <-ticker.C:
			targets, err := db.ListTargets()
			if err != nil {
				continue
			}

			now := time.Now()
			for _, t := range targets {
				if t.Paused {
					continue
				}

				last, ok := lastCheck[t.ID]
				if ok && now.Sub(last) < time.Duration(t.Interval)*time.Second {
					continue
				}

				result := checker.Check(&t)
				lastCheck[t.ID] = now

				cr := &db.CheckResult{
					TargetID:     t.ID,
					Status:       result.Status,
					StatusCode:   result.StatusCode,
					ResponseTime: result.ResponseTime.Milliseconds(),
					ContentHash:  result.ContentHash,
					Error:        result.Error,
				}
				db.SaveCheckResult(cr)

				if result.Content != "" && result.ContentHash != "" {
					snaps, _ := db.GetLatestSnapshots(t.ID, 1)
					if len(snaps) == 0 || snaps[0].Hash != result.ContentHash {
						db.SaveSnapshot(t.ID, result.Content, result.ContentHash)
					}
				}

				icon := statusIcon(result.Status)
				fmt.Printf("[%s] %s %s — %s [%dms]\n",
					now.Format("15:04:05"), icon, t.Name, result.Status, result.ResponseTime.Milliseconds())

				if result.Status == "down" || result.Status == "changed" || result.Status == "error" {
					sendNotifications(t.Name, t.URL, result.Status, result.Error)
				}
			}
		}
	}
}
