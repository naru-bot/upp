package checker

import (
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"image/color"
	"image/png"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/naru-bot/watchdog/internal/db"
)

// Patterns for dynamic content that should be ignored when computing content hashes.
// These change on every page load but don't represent meaningful content changes.
var dynamicPatterns = []*regexp.Regexp{
	// CSRF tokens (Laravel, Rails, Django, etc.)
	regexp.MustCompile(`csrf[_-]?token["']?\s*[:=]\s*["'][^"']{20,}["']`),
	regexp.MustCompile(`name=["']_token["']\s+(?:value|content)=["'][^"']+["']`),
	regexp.MustCompile(`(?:content|value)=["'][^"']+["']\s+name=["']_token["']`),
	regexp.MustCompile(`name=["']csrf[_-]?token["']\s+(?:content|value)=["'][^"']+["']`),
	regexp.MustCompile(`(?:content|value)=["'][^"']+["']\s+name=["']csrf[_-]?token["']`),
	// Nonces (CSP, WordPress, etc.)
	regexp.MustCompile(`nonce=["'][^"']+["']`),
	// Inline data-page JSON with csrf_token field (Inertia.js / Laravel)
	regexp.MustCompile(`"csrf_token"\s*:\s*"[^"]+"`),
	// HTML-encoded variants (e.g. in data-page attributes)
	regexp.MustCompile(`(?:&quot;|&#34;)csrf_token(?:&quot;|&#34;)\s*:\s*(?:&quot;|&#34;)[^&]+(?:&quot;|&#34;)`),
	regexp.MustCompile(`(?:&quot;|&#34;)_token(?:&quot;|&#34;)\s*:\s*(?:&quot;|&#34;)[^&]+(?:&quot;|&#34;)`),
	// Cloudflare Rocket Loader tokens (random hex prefix on script type and data-cf-settings)
	regexp.MustCompile(`type="[a-f0-9]{20,}-text/javascript"`),
	regexp.MustCompile(`data-cf-settings="[a-f0-9]{20,}-\|`),
	// Cloudflare beacon tokens
	regexp.MustCompile(`"r":\d+`),
	// Joomla CSRF tokens
	regexp.MustCompile(`"csrf\.token"\s*:\s*"[a-f0-9]+"`),
	regexp.MustCompile(`var\s+mtoken\s*=\s*"[a-f0-9]+"`),
	// Dynamic module/component IDs (hex suffixed identifiers like mod_mt_listings6997d393167fa)
	regexp.MustCompile(`(mod_\w+)[a-f0-9]{10,}`),
	// Hidden form tokens (Laravel _token, etc.)
	regexp.MustCompile(`name=["']_token["']\s+value=["'][^"']+["']`),
	regexp.MustCompile(`value=["'][^"']+["']\s+name=["']_token["']`),
	// Encrypted/base64 form values (honeypot fields, encrypted timestamps)
	regexp.MustCompile(`value=["']eyJ[A-Za-z0-9+/=]{50,}["']`),
	// Wire/Livewire snapshot data
	regexp.MustCompile(`wire:snapshot=["'][^"']+["']`),
	regexp.MustCompile(`wire:effects=["'][^"']+["']`),
}

// stripDynamicContent removes known dynamic tokens from content before hashing.
func stripDynamicContent(content string) string {
	result := content
	for _, pat := range dynamicPatterns {
		result = pat.ReplaceAllString(result, "")
	}
	return result
}

type Result struct {
	Status       string
	StatusCode   int
	ResponseTime time.Duration
	ContentHash  string
	Content      string
	Error        string
	SSLExpiry    *time.Time
	BodyMatch    *bool // nil if no expect keyword, true/false otherwise
	DiffPercent  float64 // Visual diff percentage (for visual checks)
}

func Check(target *db.Target) *Result {
	retries := target.Retries
	if retries <= 0 {
		retries = 1
	}

	var result *Result
	for i := 0; i < retries; i++ {
		result = checkOnce(target)
		if result.Status == "up" || result.Status == "unchanged" || result.Status == "changed" {
			return result
		}
		if i < retries-1 {
			time.Sleep(2 * time.Second) // wait between retries
		}
	}
	return result
}

func checkOnce(target *db.Target) *Result {
	switch target.Type {
	case "http", "https":
		return checkHTTP(target)
	case "tcp":
		return checkTCP(target)
	case "ping":
		return checkPing(target)
	case "dns":
		return checkDNS(target)
	case "visual":
		return checkVisual(target)
	default:
		return checkHTTP(target)
	}
}

func checkHTTP(target *db.Target) *Result {
	start := time.Now()
	result := &Result{}

	timeout := time.Duration(target.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	client := &http.Client{
		Timeout: timeout,
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
	// Strip dynamic tokens (CSRF, nonces, etc.) before hashing
	// so that only meaningful content changes are detected
	normalized := stripDynamicContent(content)
	hash := sha256.Sum256([]byte(normalized))
	result.ContentHash = fmt.Sprintf("%x", hash)

	// Check expected keyword
	if target.Expect != "" {
		matched := strings.Contains(content, target.Expect)
		result.BodyMatch = &matched
	}

	// Determine status
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		// Check keyword match
		if result.BodyMatch != nil && !*result.BodyMatch {
			result.Status = "down"
			result.Error = fmt.Sprintf("expected keyword %q not found", target.Expect)
			return result
		}

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

	timeout := time.Duration(target.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	conn, err := net.DialTimeout("tcp", target.URL, timeout)
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

// getScreenshotDir returns the directory where screenshots are stored
func getScreenshotDir() (string, error) {
	dataDir := filepath.Dir(db.GetDBPath())
	screenshotDir := filepath.Join(dataDir, "screenshots")
	return screenshotDir, os.MkdirAll(screenshotDir, 0755)
}

// findHeadlessBrowser tries to find a suitable headless browser
func findHeadlessBrowser() (string, []string) {
	browsers := []struct {
		binary string
		args   []string
	}{
		{"chrome-headless-shell", []string{"--headless", "--disable-gpu", "--no-sandbox", "--disable-dev-shm-usage"}},
		{"chromium-browser", []string{"--headless", "--disable-gpu", "--no-sandbox", "--disable-dev-shm-usage"}},
		{"chromium", []string{"--headless", "--disable-gpu", "--no-sandbox", "--disable-dev-shm-usage"}},
		{"google-chrome", []string{"--headless", "--disable-gpu", "--no-sandbox", "--disable-dev-shm-usage"}},
		{"google-chrome-stable", []string{"--headless", "--disable-gpu", "--no-sandbox", "--disable-dev-shm-usage"}},
	}

	for _, browser := range browsers {
		if _, err := exec.LookPath(browser.binary); err == nil {
			return browser.binary, browser.args
		}
	}
	return "", nil
}

// snapWritableDir returns a temp directory that snap-confined Chromium can write to.
// Snap Chromium can only write inside ~/snap/chromium/common/ due to AppArmor.
// For non-snap browsers this still works fine as a regular temp dir.
func snapWritableDir() string {
	home := os.Getenv("HOME")
	if home == "" {
		home = "/"
	}
	snapDir := filepath.Join(home, "snap", "chromium", "common", "watchdog-tmp")
	os.MkdirAll(snapDir, 0755)
	return snapDir
}

// takeScreenshot captures a screenshot of the URL using headless browser
func takeScreenshot(url, outputPath string, timeout time.Duration) error {
	binary, args := findHeadlessBrowser()
	if binary == "" {
		return fmt.Errorf("no headless browser found (run 'watchdog doctor' for install instructions)")
	}

	// Use a snap-writable temp path for the screenshot, then move it.
	// Snap-confined Chromium cannot write to arbitrary paths.
	tmpDir := snapWritableDir()
	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("watchdog_shot_%d.png", time.Now().UnixNano()))

	// Build command arguments
	cmdArgs := append(args,
		fmt.Sprintf("--screenshot=%s", tmpFile),
		"--window-size=1920,1080",
		"--hide-scrollbars",
		"--disable-background-timer-throttling",
		"--disable-backgrounding-occluded-windows",
		url,
	)

	cmd := exec.Command(binary, cmdArgs...)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	// Use context-based timeout for clean cancellation
	done := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start browser: %w", err)
	}

	go func() { done <- cmd.Wait() }()

	var err error
	if timeout > 0 {
		select {
		case err = <-done:
		case <-time.After(timeout):
			cmd.Process.Kill()
			return fmt.Errorf("screenshot timed out after %v", timeout)
		}
	} else {
		err = <-done
	}

	if err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("browser exited with error: %w", err)
	}

	// Small delay to ensure file is fully flushed to disk
	time.Sleep(200 * time.Millisecond)

	// Move from snap-writable temp to the actual output path
	if tmpFile != outputPath {
		data, err := os.ReadFile(tmpFile)
		os.Remove(tmpFile)
		if err != nil {
			return fmt.Errorf("failed to read temp screenshot: %w", err)
		}
		if err := os.WriteFile(outputPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write screenshot: %w", err)
		}
	}

	return nil
}

// compareImages compares two PNG images and returns the diff percentage
func compareImages(img1Path, img2Path string) (float64, error) {
	// Read first image
	file1, err := os.Open(img1Path)
	if err != nil {
		return 0, err
	}
	defer file1.Close()

	img1, err := png.Decode(file1)
	if err != nil {
		return 0, err
	}

	// Read second image
	file2, err := os.Open(img2Path)
	if err != nil {
		return 0, err
	}
	defer file2.Close()

	img2, err := png.Decode(file2)
	if err != nil {
		return 0, err
	}

	// Get image bounds
	bounds1 := img1.Bounds()
	bounds2 := img2.Bounds()

	// Images must be same size
	if bounds1.Dx() != bounds2.Dx() || bounds1.Dy() != bounds2.Dy() {
		return 100.0, nil // Complete difference if sizes don't match
	}

	width := bounds1.Dx()
	height := bounds1.Dy()
	totalPixels := width * height
	diffPixels := 0

	// Compare pixels
	for y := bounds1.Min.Y; y < bounds1.Max.Y; y++ {
		for x := bounds1.Min.X; x < bounds1.Max.X; x++ {
			c1 := color.RGBAModel.Convert(img1.At(x, y)).(color.RGBA)
			c2 := color.RGBAModel.Convert(img2.At(x, y)).(color.RGBA)

			// Simple pixel comparison - count as different if any channel differs
			if c1.R != c2.R || c1.G != c2.G || c1.B != c2.B || c1.A != c2.A {
				diffPixels++
			}
		}
	}

	return float64(diffPixels) / float64(totalPixels) * 100.0, nil
}

func checkVisual(target *db.Target) *Result {
	start := time.Now()
	result := &Result{}

	timeout := time.Duration(target.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second // Default 60s for visual checks
	}
	
	threshold := target.Threshold
	if threshold <= 0 {
		threshold = 5.0
	}

	// Get screenshot directory
	screenshotDir, err := getScreenshotDir()
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("failed to create screenshot directory: %v", err)
		result.ResponseTime = time.Since(start)
		return result
	}

	currentPath := filepath.Join(screenshotDir, fmt.Sprintf("%d_current.png", target.ID))
	previousPath := filepath.Join(screenshotDir, fmt.Sprintf("%d_previous.png", target.ID))

	// Move current to previous if it exists
	if _, err := os.Stat(currentPath); err == nil {
		os.Rename(currentPath, previousPath)
	}

	// Take new screenshot
	if err := takeScreenshot(target.URL, currentPath, timeout); err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("failed to take screenshot: %v", err)
		result.ResponseTime = time.Since(start)
		return result
	}

	result.ResponseTime = time.Since(start)

	// Check if screenshot was actually created
	if _, err := os.Stat(currentPath); err != nil {
		result.Status = "error"
		result.Error = "screenshot file was not created"
		return result
	}

	// Read screenshot for hash
	screenshotBytes, err := os.ReadFile(currentPath)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("failed to read screenshot: %v", err)
		return result
	}

	// Compute hash of screenshot
	hash := sha256.Sum256(screenshotBytes)
	result.ContentHash = fmt.Sprintf("%x", hash)

	// Compare with previous if it exists
	if _, err := os.Stat(previousPath); err == nil {
		diffPercent, err := compareImages(currentPath, previousPath)
		if err != nil {
			result.Status = "error"
			result.Error = fmt.Sprintf("failed to compare images: %v", err)
			return result
		}

		result.DiffPercent = diffPercent

		if diffPercent > threshold {
			result.Status = "changed"
			result.Error = fmt.Sprintf("visual diff: %.1f%% (threshold: %.1f%%)", diffPercent, threshold)
		} else {
			result.Status = "unchanged"
		}
	} else {
		result.Status = "up" // First run
		result.DiffPercent = 0.0
	}

	return result
}
