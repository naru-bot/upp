package checker

import (
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/cheryeong/watchdog/internal/db"
)

type Result struct {
	Status       string
	StatusCode   int
	ResponseTime time.Duration
	ContentHash  string
	Content      string
	Error        string
	SSLExpiry    *time.Time
}

func Check(target *db.Target) *Result {
	switch target.Type {
	case "http", "https":
		return checkHTTP(target)
	case "tcp":
		return checkTCP(target)
	case "ping":
		return checkPing(target)
	case "dns":
		return checkDNS(target)
	default:
		return checkHTTP(target)
	}
}

func checkHTTP(target *db.Target) *Result {
	start := time.Now()
	result := &Result{}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		},
	}

	req, err := http.NewRequest("GET", target.URL, nil)
	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		result.ResponseTime = time.Since(start)
		return result
	}
	req.Header.Set("User-Agent", "watchdog/1.0")

	resp, err := client.Do(req)
	result.ResponseTime = time.Since(start)

	if err != nil {
		result.Status = "down"
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	// Check SSL
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		expiry := resp.TLS.PeerCertificates[0].NotAfter
		result.SSLExpiry = &expiry
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Status = "error"
		result.Error = "failed to read body: " + err.Error()
		return result
	}

	// Extract content based on selector
	content := string(body)
	if target.Selector != "" {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
		if err == nil {
			var selected []string
			doc.Find(target.Selector).Each(func(i int, s *goquery.Selection) {
				selected = append(selected, strings.TrimSpace(s.Text()))
			})
			if len(selected) > 0 {
				content = strings.Join(selected, "\n")
			}
		}
	}

	result.Content = content
	hash := sha256.Sum256([]byte(content))
	result.ContentHash = fmt.Sprintf("%x", hash)

	// Determine status by comparing with previous snapshot
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		snaps, err := db.GetLatestSnapshots(target.ID, 1)
		if err == nil && len(snaps) > 0 {
			if snaps[0].Hash != result.ContentHash {
				result.Status = "changed"
			} else {
				result.Status = "unchanged"
			}
		} else {
			result.Status = "up"
		}
	} else {
		result.Status = "down"
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return result
}

func checkTCP(target *db.Target) *Result {
	start := time.Now()
	result := &Result{}

	conn, err := net.DialTimeout("tcp", target.URL, 10*time.Second)
	result.ResponseTime = time.Since(start)

	if err != nil {
		result.Status = "down"
		result.Error = err.Error()
		return result
	}
	conn.Close()
	result.Status = "up"
	return result
}

func checkPing(target *db.Target) *Result {
	start := time.Now()
	result := &Result{}

	cmd := exec.Command("ping", "-c", "1", "-W", "5", target.URL)
	err := cmd.Run()
	result.ResponseTime = time.Since(start)

	if err != nil {
		result.Status = "down"
		result.Error = "ping failed"
		return result
	}
	result.Status = "up"
	return result
}

func checkDNS(target *db.Target) *Result {
	start := time.Now()
	result := &Result{}

	_, err := net.LookupHost(target.URL)
	result.ResponseTime = time.Since(start)

	if err != nil {
		result.Status = "down"
		result.Error = err.Error()
		return result
	}
	result.Status = "up"
	return result
}
