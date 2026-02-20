<div align="center">

# 🐕 Watchdog

![Watchdog Demo](assets/demo.gif)

[![GitHub stars](https://img.shields.io/github/stars/naru-bot/watchdog?style=flat-square)](https://github.com/naru-bot/watchdog/stargazers)
[![Release](https://img.shields.io/github/v/release/naru-bot/watchdog?style=flat-square)](https://github.com/naru-bot/watchdog/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/naru-bot/watchdog?style=flat-square)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue?style=flat-square)](LICENSE)

**Uptime monitoring + change detection in a single binary. No Docker. No browser. Just your terminal.**

</div>

---

## Table of Contents

- [Why Watchdog?](#why-watchdog)
- [Quick Start](#quick-start)
- [Features](#features)
  - [Interactive TUI Dashboard](#-interactive-tui-dashboard)
  - [Uptime Monitoring with Sparklines](#-uptime-monitoring-with-sparklines)
  - [Change Detection + Diff](#-change-detection--diff)
  - [Quick Ping Diagnostics](#-quick-ping-diagnostics)
  - [JSON Output for AI Agents](#-json-output-for-ai-agents)
  - [Notifications](#-notifications)
  - [Daemon Mode](#-daemon-mode)
- [Check Types](#check-types)
- [Target Configuration Fields](#target-configuration-fields)
- [Tutorial: Monitor Your First Website in 60 Seconds](#tutorial-monitor-your-first-website-in-60-seconds)
- [AI Agent Integration](#ai-agent-integration)
- [How Watchdog Is Different](#how-watchdog-is-different)
- [Installation](#installation)
- [All Commands](#all-commands)
- [Configuration](#configuration)
- [Running as a Background Service](#running-as-a-background-service)
- [Tech Stack](#tech-stack)
- [Contributing](#contributing)
- [License](#license)

---

## Why Watchdog?

You just want to know if your website is up. Maybe get alerted when a pricing page changes. Simple, right?

There are plenty of great monitoring tools out there — dashboards, status pages, change trackers. They work well for what they're built for. But most share a common pattern:

1. **Docker or server required** — spin up a container, expose a port, manage volumes
2. **Web UI only** — need a browser to check on things
3. **Single purpose** — want uptime monitoring *and* change detection? Run separate services
4. **Hard to script** — integrating with cron jobs or AI agents means wrestling with REST APIs

For teams running production infrastructure with public status pages, these tools are the right choice. But if you just want to monitor a handful of sites from your terminal — or let an AI agent keep an eye on things — there should be a simpler way.

**That's why Watchdog exists.**

One 17MB binary. Works from your terminal. Works over SSH. Works in cron jobs. Works with AI agents out of the box. Install it, add a URL, done. No containers, no web UIs, no port forwarding.

---

## Quick Start

```bash
# Install
go install github.com/naru-bot/watchdog@latest

# Add a site
watchdog add https://example.com --name "My Site"

# Check it
watchdog check

# See results
watchdog status
```

That's it. You're monitoring a website.

---

## Features

### 🖥 Interactive TUI Dashboard

Full terminal UI with keyboard navigation. Browse targets, view stats, trigger checks — all without leaving your terminal. Works beautifully over SSH.

```bash
watchdog tui
```

![TUI Dashboard](assets/tui.gif)

Navigate with `↑↓`/`jk`, `Enter` for details, `c` to check, `p` to pause, `d` to delete.

---

### 📈 Uptime Monitoring with Sparklines

Track HTTP status codes, response times, availability percentage, and SSL certificate expiry. Response time trends are visualized as sparkline charts right in your terminal.

```bash
watchdog status --period 7d
watchdog watch --refresh 10    # Live auto-refreshing dashboard
```

![Uptime Monitoring](assets/uptime.gif)

---

### 🔍 Change Detection + Diff

Monitor pages for content changes. Target specific elements with CSS selectors. View colored unified diffs of what changed.

```bash
watchdog add https://example.com/pricing --name "Pricing" --selector "div.price"
watchdog check "Pricing"
watchdog diff "Pricing"
```

![Change Detection](assets/change-detection.gif)

---

### ⚡ Quick Ping Diagnostics

One-off checks without saving anything to the database. Perfect for quick debugging.

```bash
watchdog ping https://api.example.com/health
```

![Ping Diagnostics](assets/ping.gif)

---

### 🤖 JSON Output for AI Agents

Every single command supports `--json`. Consistent, structured output that's trivial to parse. This is what makes Watchdog different from every GUI-based monitor.

```bash
watchdog check --json | jq '.[] | select(.status == "down")'
```

```json
[
  {
    "target": "My Site",
    "url": "https://example.com",
    "status": "up",
    "status_code": 200,
    "response_time_ms": 142,
    "changed": false,
    "ssl_days_left": 85
  }
]
```

![JSON Output](assets/json.gif)

---

### 🔔 Notifications

Get alerted on Telegram, Discord, Slack, webhooks, or custom shell commands when things go wrong.

```bash
# Telegram
watchdog notify add --name tg --type telegram \
  --config '{"bot_token":"123:ABC","chat_id":"-100123"}'

# Discord
watchdog notify add --name discord --type discord \
  --config '{"webhook_url":"https://discord.com/api/webhooks/..."}'

# Webhook (Slack, etc.)
watchdog notify add --name alerts --type webhook \
  --config '{"url":"https://hooks.slack.com/services/..."}'

# Custom shell command
watchdog notify add --name logger --type command \
  --config '{"command":"echo \"{target} is {status}\" >> /var/log/watchdog.log"}'

# Manage
watchdog notify list
watchdog notify remove alerts
```

![Notifications](assets/notifications.gif)

---

### 👻 Daemon Mode

Run Watchdog as a background service. Checks run on schedule, notifications fire automatically.

```bash
watchdog daemon                      # Foreground
nohup watchdog daemon &              # Background
```



See [Systemd Service](#systemd-service) for production setup.

---

## Check Types

Watchdog supports multiple monitoring approaches for different use cases:

### HTTP (default)
- Monitors HTTP/HTTPS endpoints
- Tracks status codes, response times, SSL expiry
- Supports CSS selectors for targeted change detection
- Supports expected keyword matching
- Examples:
  ```bash
  watchdog add https://example.com --name "My Site"
  watchdog add https://example.com/pricing --selector "div.price" --name "Pricing"
  watchdog add https://api.example.com/health --expect "ok" --name "API Health"
  ```

### TCP
- Tests TCP port connectivity
- Example: `watchdog add example.com:3306 --type tcp --name "MySQL"`

### Ping
- ICMP-style connectivity check
- Example: `watchdog add example.com --type ping --name "Server Ping"`

### DNS
- DNS resolution check
- Example: `watchdog add example.com --type dns --name "DNS Check"`

### Visual (screenshot diff)
- Takes screenshots via headless browser and compares pixel-by-pixel
- Configurable threshold percentage (default 5%)
- Requires a headless browser (run `watchdog doctor` to check)
- Examples:
  ```bash
  watchdog add https://example.com --type visual --name "Homepage Visual"
  watchdog add https://example.com --type visual --threshold 10.0 --name "Loose Visual"
  ```

### WHOIS (domain monitoring)
- Monitors WHOIS data for changes (registrar, nameservers, status)
- Tracks domain expiry and warns when <30 days
- Domain is extracted from URL automatically
- Example:
  ```bash
  watchdog add https://example.com --type whois --name "Domain WHOIS"
  ```

---

## Target Configuration Fields

When using the TUI add/edit screen, these fields control how your targets are monitored:

| Field | Description | Applies to |
|-------|-------------|------------|
| Name | Display name for the target | All types |
| URL | Target URL or address | All types |
| Type | Check type (http, tcp, ping, dns, visual, whois) | All types |
| Interval | Seconds between checks (default: 300) | All types |
| Timeout | Request timeout in seconds (default: 30, visual: 60 recommended) | All types |
| Retries | Retry count before marking down (default: 1) | All types |
| Selector | CSS selector to monitor specific page element | http |
| Expect | Expected keyword in response body | http |
| Threshold (%) | Visual diff percentage to trigger change (default: 5.0) | visual |

---

## Tutorial: Monitor Your First Website in 60 Seconds

**Step 1** — Install Watchdog:

```bash
go install github.com/naru-bot/watchdog@latest
```

**Step 2** — Add a website to monitor:

```bash
watchdog add https://yoursite.com --name "My App"
```

**Step 3** — Run your first check:

```bash
watchdog check "My App"
```

You'll see status code, response time, SSL days remaining, and whether content changed.

**Step 4** — Set up a notification so you know when it goes down:

```bash
watchdog notify add --name tg --type telegram \
  --config '{"bot_token":"YOUR_BOT_TOKEN","chat_id":"YOUR_CHAT_ID"}'
```

**Step 5** — Start the daemon for continuous monitoring:

```bash
watchdog daemon
```

Done. You're monitoring a website with notifications. No Docker, no browser, no account signups.

**Bonus** — Launch the TUI for a beautiful overview:

```bash
watchdog tui
```

---

## AI Agent Integration

This is Watchdog's superpower. Every command speaks JSON natively. No API servers, no authentication tokens, no SDK. Just pipe and parse.

### Use with AI coding agents

```bash
# "Are any of my sites down?"
watchdog check --json | jq '.[] | select(.status == "down")'

# "Which sites have low uptime?"
watchdog status --json | jq '.[] | select(.uptime_percent < 99)'

# "Did anything change?"
watchdog check --json | jq '.[] | select(.changed == true)'

# "Show me the diff"
watchdog diff "Pricing Page" --json
```

### JSON conventions

- **Lists** → JSON arrays: `[{...}, {...}]`
- **Single items** → JSON objects: `{...}`
- **Errors** → `{"error": "message"}`
- **Timestamps** → RFC3339 format

### Cron integration

```bash
# Check every 5 minutes, email on failures
*/5 * * * * watchdog check --json | jq -e '.[] | select(.status == "down")' && \
  echo "ALERT" | mail -s "Site down" admin@example.com
```

---

## How Watchdog Is Different

| | Typical web-based monitors | **Watchdog** |
|---|---|---|
| Install | Docker / server setup | **Single binary, zero dependencies** |
| Interface | Web browser required | **Terminal / TUI / JSON** |
| Check types | HTTP only | **HTTP, TCP, Ping, DNS, Visual, WHOIS** |
| Uptime + change detection | Usually separate tools | **All-in-one** |
| AI & automation friendly | REST API wrappers | **Native CLI + JSON on every command** |
| Interactive dashboard | Browser tab | **TUI that works over SSH** |
| Resource usage | 100-300MB+ RAM | **~10MB** |
| Scriptable | Needs API tokens & HTTP calls | **Pipe-friendly, works in cron/shell** |

---

## Installation

### Go install (recommended)

```bash
go install github.com/naru-bot/watchdog@latest
```

Requires Go 1.24+.

### Binary releases

<details>
<summary><b>macOS (Apple Silicon)</b></summary>

```bash
curl -LO https://github.com/naru-bot/watchdog/releases/latest/download/watchdog_darwin_arm64.tar.gz
tar xzf watchdog_darwin_arm64.tar.gz
sudo mv watchdog /usr/local/bin/
```
</details>

<details>
<summary><b>macOS (Intel)</b></summary>

```bash
curl -LO https://github.com/naru-bot/watchdog/releases/latest/download/watchdog_darwin_amd64.tar.gz
tar xzf watchdog_darwin_amd64.tar.gz
sudo mv watchdog /usr/local/bin/
```
</details>

<details>
<summary><b>Linux (x86_64)</b></summary>

```bash
curl -LO https://github.com/naru-bot/watchdog/releases/latest/download/watchdog_linux_amd64.tar.gz
tar xzf watchdog_linux_amd64.tar.gz
sudo mv watchdog /usr/local/bin/
```
</details>

<details>
<summary><b>Linux (ARM64)</b></summary>

```bash
curl -LO https://github.com/naru-bot/watchdog/releases/latest/download/watchdog_linux_arm64.tar.gz
tar xzf watchdog_linux_arm64.tar.gz
sudo mv watchdog /usr/local/bin/
```
</details>

### Build from source

```bash
git clone https://github.com/naru-bot/watchdog.git
cd watchdog
make build
sudo mv watchdog /usr/local/bin/
```

### Cross-compilation

```bash
make cross
# Produces: watchdog-darwin-arm64, watchdog-darwin-amd64, watchdog-linux-amd64, watchdog-linux-arm64
```

---

## All Commands

| Command | Description |
|---------|-------------|
| `init` | Initialize configuration file |
| `add <url>` | Add a URL to monitor |
| `remove <target>` | Remove a monitored target |
| `list` / `ls` | List all monitored targets |
| `check [target]` | Run checks (all or specific) |
| `status [target]` | Show uptime stats and summary |
| `tui` | Interactive terminal dashboard |
| `watch` | Live auto-refreshing dashboard |
| `ping <url>` | Quick one-off check (no DB save) |
| `import <file>` | Bulk import targets from YAML |
| `diff <target>` | Show content changes between snapshots |
| `history <target>` | Show check history |
| `pause <target>` | Pause monitoring |
| `unpause <target>` | Resume monitoring |
| `notify add\|list\|remove` | Manage notification channels |
| `export` | Export data as JSON or CSV |
| `daemon` | Run as background service |
| `doctor` | Check system dependencies (headless browser for visual checks) |
| `completion` | Generate shell completions (bash/zsh/fish/powershell) |
| `version` | Print version |

### Global Flags

| Flag | Description |
|------|-------------|
| `--json` | Output in JSON format (all commands) |
| `--no-color` | Disable colored output |
| `-v, --verbose` | Verbose output |
| `-q, --quiet` | Suppress non-essential output |

### Add command flags

```bash
watchdog add <url> [flags]
  --name         Target name (auto-generated from URL if omitted)
  --type         Check type: http, tcp, ping, dns, visual, whois (default: http)
  --interval     Check interval in seconds (default: 300)
  --selector     CSS selector for change detection (http type)
  --expect       Expected keyword in response body (http type)
  --timeout      Request timeout in seconds (default: 30)
  --retries      Retry count before marking as down (default: 1)
  --threshold    Visual diff threshold percentage (visual type, default: 5.0)
```

---

## Configuration

```bash
watchdog init    # Creates ~/.config/watchdog/config.yml
```

```yaml
defaults:
  interval: 300        # Check interval (seconds)
  type: http           # Default check type
  timeout: 30          # HTTP timeout (seconds)
  retry_count: 1       # Retries before marking down
  user_agent: watchdog/1.0

display:
  color: true
  format: table
  verbose: false
```

### Data storage

All data lives in `~/.watchdog/watchdog.db` (SQLite). Back up by copying the file, query with any SQLite client, or export via `watchdog export`.

---

## Running as a Background Service

To run Watchdog in the background and survive reboots, set it up as a systemd service.

**One-liner setup:**

```bash
sudo tee /etc/systemd/system/watchdog-monitor.service << 'EOF'
[Unit]
Description=Watchdog Website Monitor
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/watchdog daemon
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl enable --now watchdog-monitor
```

**Check status:**

```bash
sudo systemctl status watchdog-monitor
```

**View logs:**

```bash
journalctl -u watchdog-monitor -f
```

**Stop / restart:**

```bash
sudo systemctl stop watchdog-monitor
sudo systemctl restart watchdog-monitor
```

### Removing the Service

```bash
sudo systemctl stop watchdog-monitor
sudo systemctl disable watchdog-monitor
sudo rm /etc/systemd/system/watchdog-monitor.service
sudo systemctl daemon-reload
```

---

## Uninstalling Watchdog

To completely remove Watchdog from your system:

```bash
# 1. Stop and remove the service (if running)
sudo systemctl stop watchdog-monitor
sudo systemctl disable watchdog-monitor
sudo rm /etc/systemd/system/watchdog-monitor.service
sudo systemctl daemon-reload

# 2. Remove the binary
sudo rm /usr/local/bin/watchdog
# Or if installed via go install:
rm $(go env GOPATH)/bin/watchdog

# 3. Remove data and config
rm -rf ~/.watchdog
rm -rf ~/.config/watchdog
```

---

## Tech Stack

Built with some excellent Go libraries:

- [Cobra](https://github.com/spf13/cobra) — CLI framework
- [Bubbletea](https://github.com/charmbracelet/bubbletea) + [Bubbles](https://github.com/charmbracelet/bubbles) + [Lipgloss](https://github.com/charmbracelet/lipgloss) — TUI
- [modernc.org/sqlite](https://modernc.org/sqlite) — Pure Go SQLite (zero CGO)
- [goquery](https://github.com/PuerkitoBio/goquery) — HTML parsing
- Custom LCS-based line diff engine

---

## Contributing

Contributions welcome! Open an issue or PR on [GitHub](https://github.com/naru-bot/watchdog).

```bash
git clone https://github.com/naru-bot/watchdog.git
cd watchdog
make build
# hack away
```

---

## License

[MIT](LICENSE)
