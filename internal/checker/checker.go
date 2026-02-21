package checker

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"image/color"
	"image/png"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/itchyny/gojq"
	"github.com/likexian/whois"
	whoisparser "github.com/likexian/whois-parser"
	"github.com/naru-bot/upp/internal/db"
)

// isAcceptedStatus checks if a status code is in the accept-status spec.
// Format: "200,201,300-399,404" — comma-separated codes or ranges.
func isAcceptedStatus(code int, spec string) bool {
	if spec == "" {
		return code >= 200 && code < 400
	}
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if idx := strings.Index(part, "-"); idx > 0 {
			lo, err1 := strconv.Atoi(strings.TrimSpace(part[:idx]))
			hi, err2 := strconv.Atoi(strings.TrimSpace(part[idx+1:]))
			if err1 == nil && err2 == nil && code >= lo && code <= hi {
				return true
			}
		} else {
			if v, err := strconv.Atoi(part); err == nil && code == v {
				return true
			}
		}
	}
	return false
}

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
	BodyMatch    *bool   // nil if no expect keyword, true/false otherwise
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
	case "whois":
		return checkWhois(target)
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

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: target.Insecure},
	}
	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
	if target.NoFollow {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	method := strings.ToUpper(target.Method)
	if method == "" {
		method = "GET"
	}
	var bodyReader io.Reader
	if target.Body != "" {
		bodyReader = strings.NewReader(target.Body)
	}
	req, err := http.NewRequest(method, target.URL, bodyReader)
	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		result.ResponseTime = time.Since(start)
		return result
	}
	req.Header.Set("User-Agent", "upp/1.0")
	if target.Body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if target.Headers != "" {
		var customHeaders map[string]string
		if err := json.Unmarshal([]byte(target.Headers), &customHeaders); err == nil {
			for k, v := range customHeaders {
				req.Header.Set(k, v)
			}
		}
	}

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

	// Apply jq filter if set (for JSON API monitoring)
	content := string(body)
	if target.JQFilter != "" {
		var jsonData interface{}
		if err := json.Unmarshal(body, &jsonData); err != nil {
			result.Status = "error"
			result.Error = "response is not valid JSON: " + err.Error()
			return result
		}
		query, err := gojq.Parse(target.JQFilter)
		if err != nil {
			result.Status = "error"
			result.Error = "invalid jq filter: " + err.Error()
			return result
		}
		var filtered []string
		iter := query.Run(jsonData)
		for {
			v, ok := iter.Next()
			if !ok {
				break
			}
			if err, isErr := v.(error); isErr {
				result.Status = "error"
				result.Error = "jq filter error: " + err.Error()
				return result
			}
			switch val := v.(type) {
			case string:
				filtered = append(filtered, val)
			default:
				b, _ := json.MarshalIndent(val, "", "  ")
				filtered = append(filtered, string(b))
			}
		}
		content = strings.Join(filtered, "\n")
	}

	// Extract content based on selector (for HTML pages)
	if target.JQFilter == "" && target.Selector != "" {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
		if err == nil {
			var selected []string
			doc.Find(target.Selector).Each(func(i int, s *goquery.Selection) {
				// Strip style/script so CSS/JS doesn't pollute extracted text.
				s.Find("style,script").Remove()
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
	if isAcceptedStatus(resp.StatusCode, target.AcceptStatus) {
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

	host := target.URL
	// Strip protocol/path if a full URL was provided
	if strings.Contains(host, "://") {
		if u, err := url.Parse(host); err == nil {
			host = u.Hostname()
		}
	}

	addrs, err := net.LookupHost(host)
	result.ResponseTime = time.Since(start)

	if err != nil {
		result.Status = "down"
		result.Error = err.Error()
		content := fmt.Sprintf("Domain: %s\nStatus: not resolving\nError: %s", host, err.Error())
		result.Content = content
		hash := sha256.Sum256([]byte("unresolved"))
		result.ContentHash = fmt.Sprintf("%x", hash)
		return result
	}

	// Build content with resolved addresses
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Domain: %s\n", host))
	sb.WriteString(fmt.Sprintf("Resolved: %s\n", strings.Join(addrs, ", ")))

	// Also try MX, NS, TXT records
	if mx, err := net.LookupMX(host); err == nil && len(mx) > 0 {
		var mxHosts []string
		for _, m := range mx {
			mxHosts = append(mxHosts, fmt.Sprintf("%s (pri %d)", m.Host, m.Pref))
		}
		sb.WriteString(fmt.Sprintf("MX: %s\n", strings.Join(mxHosts, ", ")))
	}
	if ns, err := net.LookupNS(host); err == nil && len(ns) > 0 {
		var nsHosts []string
		for _, n := range ns {
			nsHosts = append(nsHosts, n.Host)
		}
		sb.WriteString(fmt.Sprintf("NS: %s\n", strings.Join(nsHosts, ", ")))
	}
	if txt, err := net.LookupTXT(host); err == nil && len(txt) > 0 {
		sb.WriteString(fmt.Sprintf("TXT: %s\n", strings.Join(txt, "; ")))
	}

	result.Content = sb.String()
	hash := sha256.Sum256([]byte(result.Content))
	result.ContentHash = fmt.Sprintf("%x", hash)
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
		if h, err := os.UserHomeDir(); err == nil && h != "" {
			home = h
		} else if u, err := user.Current(); err == nil {
			home = u.HomeDir
		} else {
			home = "/root"
		}
	}
	snapDir := filepath.Join(home, "snap", "chromium", "common", "upp-tmp")
	os.MkdirAll(snapDir, 0755)
	return snapDir
}

// takeScreenshot captures a screenshot of the URL using headless browser
func takeScreenshot(url, outputPath string, timeout time.Duration) error {
	binary, args := findHeadlessBrowser()
	if binary == "" {
		return fmt.Errorf("no headless browser found (run 'upp doctor' for install instructions)")
	}

	// Use a snap-writable temp path for the screenshot, then move it.
	// Snap-confined Chromium cannot write to arbitrary paths.
	tmpDir := snapWritableDir()
	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("upp_shot_%d.png", time.Now().UnixNano()))

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

func checkWhois(target *db.Target) *Result {
	start := time.Now()
	result := &Result{}

	// Extract domain from URL
	domain, err := extractDomain(target.URL)
	if err != nil {
		result.Status = "error"
		result.Error = "failed to extract domain: " + err.Error()
		result.ResponseTime = time.Since(start)
		return result
	}

	// Query WHOIS
	whoisResult, err := whois.Whois(domain)
	if err != nil {
		result.Status = "error"
		result.Error = "whois query failed: " + err.Error()
		result.ResponseTime = time.Since(start)
		return result
	}

	result.ResponseTime = time.Since(start)

	// Parse WHOIS result
	info, err := whoisparser.Parse(whoisResult)
	if err != nil {
		result.Status = "error"
		result.Error = "failed to parse whois: " + err.Error()
		return result
	}

	// Format content
	content := formatWhoisContent(domain, &info)
	result.Content = content

	// Strip dynamic content and hash for change detection
	normalized := stripWhoisDynamicContent(content)
	hash := sha256.Sum256([]byte(normalized))
	result.ContentHash = fmt.Sprintf("%x", hash)

	// Check for expiry warning
	if info.Domain != nil && info.Domain.ExpirationDate != "" {
		if expiryDate, err := time.Parse("2006-01-02", info.Domain.ExpirationDate); err == nil {
			daysUntilExpiry := int(time.Until(expiryDate).Hours() / 24)
			if daysUntilExpiry < 30 {
				result.Error = fmt.Sprintf("⚠ Domain expires in %d days", daysUntilExpiry)
			}
		}
	}

	// Compare with previous snapshot
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

	return result
}

// extractDomain extracts the registrable domain from a URL
func extractDomain(rawURL string) (string, error) {
	// Add protocol if missing
	if !strings.Contains(rawURL, "://") {
		rawURL = "http://" + rawURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	domain := u.Hostname()
	if domain == "" {
		return "", fmt.Errorf("no hostname in URL")
	}

	// Remove common subdomains to get registrable domain
	parts := strings.Split(domain, ".")
	if len(parts) >= 2 {
		// For now, just use last two parts (domain.tld)
		// This is a simple approach that works for most cases
		return strings.Join(parts[len(parts)-2:], "."), nil
	}

	return domain, nil
}

// formatWhoisContent formats the parsed WHOIS info into a readable format
func formatWhoisContent(domain string, info *whoisparser.WhoisInfo) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Domain: %s\n", domain))

	if info.Registrar != nil {
		sb.WriteString(fmt.Sprintf("Registrar: %s\n", info.Registrar.Name))
	}

	if info.Domain != nil {
		if info.Domain.CreatedDate != "" {
			sb.WriteString(fmt.Sprintf("Created: %s\n", info.Domain.CreatedDate))
		}
		
		if info.Domain.ExpirationDate != "" {
			if expiryDate, err := time.Parse("2006-01-02", info.Domain.ExpirationDate); err == nil {
				daysUntilExpiry := int(time.Until(expiryDate).Hours() / 24)
				sb.WriteString(fmt.Sprintf("Expires: %s (%d days)\n", info.Domain.ExpirationDate, daysUntilExpiry))
			} else {
				sb.WriteString(fmt.Sprintf("Expires: %s\n", info.Domain.ExpirationDate))
			}
		}

		if len(info.Domain.Status) > 0 {
			sb.WriteString(fmt.Sprintf("Status: %s\n", strings.Join(info.Domain.Status, ", ")))
		}
	}

	if len(info.Domain.NameServers) > 0 {
		nameservers := strings.Join(info.Domain.NameServers, ", ")
		sb.WriteString(fmt.Sprintf("Nameservers: %s\n", nameservers))
	}

	return sb.String()
}

// stripWhoisDynamicContent removes frequently changing fields before hashing
func stripWhoisDynamicContent(content string) string {
	// Remove timestamps and dates that change frequently
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`Updated:\s*[^\n]+`),
		regexp.MustCompile(`Last updated on:\s*[^\n]+`),
		regexp.MustCompile(`Last Modified:\s*[^\n]+`),
		regexp.MustCompile(`>>> Last update of.*`),
		regexp.MustCompile(`Record last updated.*`),
		regexp.MustCompile(`Database last updated.*`),
		// Remove dynamic query information
		regexp.MustCompile(`Query time:\s*[^\n]+`),
		regexp.MustCompile(`No match for.*`),
	}

	result := content
	for _, pat := range patterns {
		result = pat.ReplaceAllString(result, "")
	}

	// Normalize whitespace
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")
	return strings.TrimSpace(result)
}
