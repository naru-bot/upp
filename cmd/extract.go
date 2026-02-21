package cmd

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "extract <url>",
		Short: "Fetch a URL and show extracted content",
		Long: `Fetch a URL once and print the extracted content.

Examples:
  upp extract https://example.com --selector "main"
  upp extract https://example.com/pricing --selector "div.price"`,
		Args: requireArgs(1),
		Run:  runExtract,
	}
	cmd.Flags().StringP("selector", "s", "", "CSS selector to extract content")
	cmd.Flags().Int("timeout", 30, "Request timeout in seconds")
	rootCmd.AddCommand(cmd)
}

type extractOutput struct {
	URL          string `json:"url"`
	Selector     string `json:"selector,omitempty"`
	StatusCode   int    `json:"status_code"`
	ResponseTime int64  `json:"response_time_ms"`
	Content      string `json:"content"`
}

func runExtract(cmd *cobra.Command, args []string) {
	url := args[0]
	selector, _ := cmd.Flags().GetString("selector")
	timeoutSeconds, _ := cmd.Flags().GetInt("timeout")
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}

	start := time.Now()
	client := &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		exitError(err.Error())
	}
	req.Header.Set("User-Agent", "upp/1.0")

	resp, err := client.Do(req)
	responseTime := time.Since(start)
	if err != nil {
		exitError(err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		exitError("failed to read body: " + err.Error())
	}

	content := string(body)
	if selector != "" {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
		if err == nil {
			var selected []string
			doc.Find(selector).Each(func(i int, s *goquery.Selection) {
				// Strip style/script so CSS/JS doesn't pollute extracted text.
				s.Find("style,script").Remove()
				selected = append(selected, strings.TrimSpace(s.Text()))
			})
			if len(selected) > 0 {
				content = strings.Join(selected, "\n")
			} else {
				content = ""
			}
		}
	}

	if jsonOutput {
		printJSON(extractOutput{
			URL:          url,
			Selector:     selector,
			StatusCode:   resp.StatusCode,
			ResponseTime: responseTime.Milliseconds(),
			Content:      content,
		})
		return
	}

	if selector != "" {
		fmt.Printf("URL: %s\nSelector: %s\nStatus: %d\n\n", url, selector, resp.StatusCode)
	} else {
		fmt.Printf("URL: %s\nStatus: %d\n\n", url, resp.StatusCode)
	}
	fmt.Print(content)
	if len(content) > 0 && content[len(content)-1] != '\n' {
		fmt.Print("\n")
	}
}
