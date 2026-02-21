package cmd

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check system dependencies for Upp features",
		Long: `Check system for dependencies needed by Upp features and reports status with install instructions.

This command verifies that required tools are available for advanced features:
- Headless browser (required for visual checks)

Examples:
  upp doctor`,
		Run: runDoctor,
	}
	rootCmd.AddCommand(cmd)
}

type doctorCheck struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"` // "ok", "missing", "error"
	Path        string `json:"path,omitempty"`
	Version     string `json:"version,omitempty"`
	Message     string `json:"message,omitempty"`
}

type doctorOutput struct {
	Checks     []doctorCheck `json:"checks"`
	IssueCount int           `json:"issue_count"`
}

func runDoctor(cmd *cobra.Command, args []string) {
	if !jsonOutput && !quiet {
		fmt.Println("🔍 Upp Doctor")
		fmt.Println()
		fmt.Println("Checking dependencies...")
		fmt.Println()
	}

	var checks []doctorCheck
	issueCount := 0

	// Check for headless browser
	browserCheck := checkHeadlessBrowser()
	if browserCheck.Status != "ok" {
		issueCount++
	}
	checks = append(checks, browserCheck)

	if jsonOutput {
		printJSON(doctorOutput{
			Checks:     checks,
			IssueCount: issueCount,
		})
		return
	}

	// Print results
	for _, check := range checks {
		fmt.Printf("  %s\n", check.Description)
		
		switch check.Status {
		case "ok":
			fmt.Printf("  %s %s %s (%s)\n", colorGreen("✅"), check.Name, check.Version, check.Path)
		case "missing":
			fmt.Printf("  %s %s\n", colorRed("❌"), check.Message)
			if installInstructions := getInstallInstructions(check.Name); installInstructions != "" {
				fmt.Printf("\n  Install one of:\n%s\n", installInstructions)
			}
		case "error":
			fmt.Printf("  %s %s\n", colorRed("❌"), check.Message)
		}
		fmt.Println()
	}

	// Summary
	if issueCount == 0 {
		fmt.Println(colorGreen("All checks passed!"))
	} else {
		fmt.Printf("%s found.\n", colorRed(fmt.Sprintf("%d issue%s", issueCount, pluralize(issueCount))))
	}
}

func checkHeadlessBrowser() doctorCheck {
	browsers := []string{
		"chrome-headless-shell",
		"chromium-browser", 
		"chromium",
		"google-chrome",
		"google-chrome-stable",
	}

	for _, browser := range browsers {
		path, err := exec.LookPath(browser)
		if err != nil {
			continue
		}

		// Try to get version
		version := "unknown version"
		if versionCmd := exec.Command(browser, "--version"); versionCmd != nil {
			if output, err := versionCmd.Output(); err == nil {
				version = strings.TrimSpace(string(output))
				// Clean up version string - extract just the version number
				if idx := strings.LastIndex(version, " "); idx != -1 {
					version = version[idx+1:]
				}
			}
		}

		return doctorCheck{
			Name:        browser,
			Description: "Headless Browser (visual checks)",
			Status:      "ok",
			Path:        path,
			Version:     version,
		}
	}

	return doctorCheck{
		Name:        "headless-browser",
		Description: "Headless Browser (visual checks)", 
		Status:      "missing",
		Message:     "No headless browser found",
	}
}

func getInstallInstructions(component string) string {
	if component != "headless-browser" {
		return ""
	}

	switch runtime.GOOS {
	case "linux":
		// Try to detect Linux distribution
		if distro := detectLinuxDistro(); distro != "" {
			switch distro {
			case "ubuntu", "debian":
				return "    Ubuntu/Debian:  sudo apt install chromium-browser"
			case "arch":
				return "    Arch:           sudo pacman -S chromium"
			case "alpine":
				return "    Alpine:         apk add chromium"
			case "fedora", "rhel", "centos":
				return "    Fedora/RHEL:    sudo dnf install chromium"
			}
		}
		return "    Linux:          Install Chromium or Google Chrome from your package manager"
	case "darwin":
		return "    macOS:          brew install chromium"
	case "windows":
		return "    Windows:        Install Google Chrome or Chromium from the web"
	default:
		return "    Install Chromium or Google Chrome from your package manager"
	}
}

func detectLinuxDistro() string {
	// Try to detect from /etc/os-release
	if output, err := exec.Command("sh", "-c", "grep '^ID=' /etc/os-release 2>/dev/null | cut -d= -f2 | tr -d '\"'").Output(); err == nil {
		distro := strings.TrimSpace(strings.ToLower(string(output)))
		switch distro {
		case "ubuntu", "debian", "arch", "alpine", "fedora", "rhel", "centos":
			return distro
		}
	}

	// Fallback: try common commands
	if _, err := exec.LookPath("apt"); err == nil {
		return "debian"
	}
	if _, err := exec.LookPath("pacman"); err == nil {
		return "arch"
	}
	if _, err := exec.LookPath("apk"); err == nil {
		return "alpine"
	}
	if _, err := exec.LookPath("dnf"); err == nil {
		return "fedora"
	}
	if _, err := exec.LookPath("yum"); err == nil {
		return "rhel"
	}

	return ""
}

func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}