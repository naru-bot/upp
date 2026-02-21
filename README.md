<div align="center">

# 🐕 Upp

![Watchdog Demo](assets/demo.gif)

[![GitHub stars](https://img.shields.io/github/stars/naru-bot/watchdog?style=flat-square)](https://github.com/naru-bot/watchdog/stargazers)
[![Release](https://img.shields.io/github/v/release/naru-bot/watchdog?style=flat-square)](https://github.com/naru-bot/watchdog/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/naru-bot/watchdog?style=flat-square)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue?style=flat-square)](LICENSE)

**Uptime monitoring + change detection in a single binary. No Docker. No browser. Just your terminal.**

</div>

---

## Table of Contents

- [Why Upp?](#why-upp)
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
- [How Upp Is Different](#how-upp-is-different)
- [Installation](#installation)
- [All Commands](#all-commands)
- [Configuration](#configuration)
  - [Reference](#reference)
- [Running as a Background Service](#running-as-a-background-service)
- [Tech Stack](#tech-stack)
- [Contributing](#contributing)
- [License](#license)

---

## Why Upp?

You just want to know if your website is up. Maybe get alerted when a pricing page changes. Simple, right?

There are plenty of great monitoring tools out there — dashboards, status pages, change trackers. They work well for what they're built for. But most share a common pattern:

1. **Docker or server required** — spin up a container, expose a port, manage volumes
2. **Web UI only** — need a browser to check on things
3. **Single purpose** — want uptime monitoring *and* change detection? Run separate services
4. **Hard to script** — integrating with cron jobs or AI agents means wrestling with REST APIs

For teams running production infrastructure with public status pages, these tools are the right choice. But if you just want to monitor a handful of sites from your terminal — or let an AI agent keep an eye on things — there should be a simpler way.

**That's why Upp exists.**

One 17MB binary. Works from your terminal. Works over SSH. Works in cron jobs. Works with AI agents out of the box. Install it, add a URL, done. No containers, no web UIs, no port forwarding.

---

## Quick Start

```bash
# Install
go install github.com/naru-bot/upp@latest

# Add a site
upp add https://example.com --name "My Site"

# Check it
upp check

# See results
upp status
```

That's it. You're monitoring a website.

---

## Features

### 🖥 Interactive TUI Dashboard

Full terminal UI with keyboard navigation. Browse targets, view stats, trigger checks — all without leaving your terminal. Works beautifully over SSH.

```bash
upp tui
```

![TUI Dashboard](assets/tui.gif)

Navigate with `↑↓`/`jk`, `Enter` for details, `c` to check, `p` to pause, `d` to delete.

---

### 📈 Uptime Monitoring with Sparklines

Track HTTP status codes, response times, availability percentage, and SSL certificate expiry. Response time trends are visualized as sparkline charts right in your terminal.

```bash
upp status --period 7d
upp watch --refresh 10    # Live auto-refreshing dashboard
```

![Uptime Monitoring](assets/uptime.gif)

---

### 🔍 Change Detection + Diff

Monitor pages for content changes. Target specific elements with CSS selectors. View colored unified diffs of what changed.

```bash
upp add https://example.com/pricing --name "Pricing" --selector "div.price"
upp check "Pricing"
upp diff "Pricing"
```

![Change Detection](assets/change-detection.gif)

---

### ⚡ Quick Ping Diagnostics

One-off checks without saving anything to the database. Perfect for quick debugging.

```bash
upp ping https://api.example.com/health
```

![Ping Diagnostics](assets/ping.gif)

---

### 🤖 JSON Output for AI Agents

Every single command supports `--json`. Consistent, structured output that's trivial to parse. This is what makes Upp different from every GUI-based monitor.

```bash
upp check --json | jq '.[] | select(.status == "down")'
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
upp notify add --name tg --type telegram \
  --config '{"bot_token":"123:ABC","chat_id":"-100123"}'

# Discord
upp notify add --name discord --type discord \
  --config '{"webhook_url":"https://discord.com/api/webhooks/..."}'

# Webhook (Slack, etc.)
upp notify add --name alerts --type webhook \
  --config '{"url":"https://hooks.slack.com/services/..."}'

# Custom shell command
upp notify add --name logger --type command \
  --config '{"command":"echo \"{target} is {status}\" >> /var/log/upp.log"}'

# Manage
upp notify list
upp notify remove alerts
```

![Notifications](assets/notifications.gif)

---

### 👻 Daemon Mode

Run Upp as a background service. Checks run on schedule, notifications fire automatically.

```bash
upp daemon                      # Foreground
nohup upp daemon &              # Background
```



See [Systemd Service](#systemd-service) for production setup.

---

## Check Types

Upp supports multiple monitoring approaches for different use cases:

### HTTP (default)
- Monitors HTTP/HTTPS endpoints
- Tracks status codes, response times, SSL expiry
- Supports CSS selectors for targeted change detection
- Supports expected keyword matching
- Examples:
  ```bash
  upp add https://example.com --name "My Site"
  upp add https://example.com/pricing --selector "div.price" --name "Pricing"
  upp add https://api.example.com/health --expect "ok" --name "API Health"
  ```

### TCP
- Tests TCP port connectivity
- Example: `upp add example.com:3306 --type tcp --name "MySQL"`

### Ping
- ICMP-style connectivity check
- Example: `upp add example.com --type ping --name "Server Ping"`

### DNS
- DNS resolution check
- Example: `upp add example.com --type dns --name "DNS Check"`

### Visual (screenshot diff)
- Takes screenshots via headless browser and compares pixel-by-pixel
- Configurable threshold percentage (default 5%)
- Requires a headless browser (run `upp doctor` to check)
- Examples:
  ```bash
  upp add https://example.com --type visual --name "Homepage Visual"
  upp add https://example.com --type visual --threshold 10.0 --name "Loose Visual"
  ```

### WHOIS (domain monitoring)
- Monitors WHOIS data for changes (registrar, nameservers, status)
- Tracks domain expiry and warns when <30 days
- Domain is extracted from URL automatically
- Example:
  ```bash
  upp add https://example.com --type whois --name "Domain WHOIS"
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

**Step 1** — Install Upp:

```bash
go install github.com/naru-bot/upp@latest
```

**Step 2** — Add a website to monitor:

```bash
upp add https://yoursite.com --name "My App"
```

**Step 3** — Run your first check:

```bash
upp check "My App"
```

You'll see status code, response time, SSL days remaining, and whether content changed.

**Step 4** — Set up a notification so you know when it goes down:

```bash
upp notify add --name tg --type telegram \
  --config '{"bot_token":"YOUR_BOT_TOKEN","chat_id":"YOUR_CHAT_ID"}'
```

**Step 5** — Start the daemon for continuous monitoring:

```bash
upp daemon
```

Done. You're monitoring a website with notifications. No Docker, no browser, no account signups.

**Bonus** — Launch the TUI for a beautiful overview:

```bash
upp tui
```

---

## AI Agent Integration

This is Upp's superpower. Every command speaks JSON natively. No API servers, no authentication tokens, no SDK. Just pipe and parse.

### Use with AI coding agents

```bash
# "Are any of my sites down?"
upp check --json | jq '.[] | select(.status == "down")'

# "Which sites have low uptime?"
upp status --json | jq '.[] | select(.uptime_percent < 99)'

# "Did anything change?"
upp check --json | jq '.[] | select(.changed == true)'

# "Show me the diff"
upp diff "Pricing Page" --json
```

### JSON conventions

- **Lists** → JSON arrays: `[{...}, {...}]`
- **Single items** → JSON objects: `{...}`
- **Errors** → `{"error": "message"}`
- **Timestamps** → RFC3339 format

### Cron integration

```bash
# Check every 5 minutes, email on failures
*/5 * * * * upp check --json | jq -e '.[] | select(.status == "down")' && \
  echo "ALERT" | mail -s "Site down" admin@example.com
```

---

## How Upp Is Different

| | Typical web-based monitors | **Upp** |
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
go install github.com/naru-bot/upp@latest
```

Requires Go 1.24+.

### Binary releases

<details>
<summary><b>macOS (Apple Silicon)</b></summary>

```bash
curl -LO https://github.com/naru-bot/watchdog/releases/latest/download/watchdog_darwin_arm64.tar.gz
tar xzf upp_darwin_arm64.tar.gz
sudo mv upp /usr/local/bin/
```
</details>

<details>
<summary><b>macOS (Intel)</b></summary>

```bash
curl -LO https://github.com/naru-bot/watchdog/releases/latest/download/watchdog_darwin_amd64.tar.gz
tar xzf upp_darwin_amd64.tar.gz
sudo mv upp /usr/local/bin/
```
</details>

<details>
<summary><b>Linux (x86_64)</b></summary>

```bash
curl -LO https://github.com/naru-bot/watchdog/releases/latest/download/watchdog_linux_amd64.tar.gz
tar xzf upp_linux_amd64.tar.gz
sudo mv upp /usr/local/bin/
```
</details>

<details>
<summary><b>Linux (ARM64)</b></summary>

```bash
curl -LO https://github.com/naru-bot/watchdog/releases/latest/download/watchdog_linux_arm64.tar.gz
tar xzf upp_linux_arm64.tar.gz
sudo mv upp /usr/local/bin/
```
</details>

### Build from source

```bash
git clone https://github.com/naru-bot/watchdog.git
cd upp
make build
sudo mv upp /usr/local/bin/
```

### Cross-compilation

```bash
make cross
# Produces: upp-darwin-arm64, upp-darwin-amd64, upp-linux-amd64, upp-linux-arm64
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
upp add <url> [flags]
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
upp init    # Creates ~/.config/upp/config.yml
```

Config file location: `~/.config/upp/config.yml` (or `$XDG_CONFIG_HOME/upp/config.yml`)

### Full example

```yaml
defaults:
  interval: 300
  type: http
  timeout: 30
  retry_count: 1
  user_agent: upp/1.0

display:
  color: true
  format: table
  verbose: false

thresholds:
  ssl_warn_days: 30

headers:
  Authorization: Bearer my-token
  X-Custom: value
```

### Reference

#### `defaults` — Default values for new targets

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `interval` | int | `300` | Check interval in seconds. Applied to new targets when `--interval` is not specified. |
| `type` | string | `http` | Default check type when `--type` is not specified. One of: `http`, `tcp`, `ping`, `dns`, `visual`, `whois`. |
| `timeout` | int | `30` | HTTP/TCP request timeout in seconds. For visual checks, consider increasing to 60. |
| `retry_count` | int | `1` | Number of retries before marking a target as down. Helps avoid false positives from transient failures. |
| `user_agent` | string | `upp/1.0` | User-Agent header sent with HTTP requests. Some sites block default Go user agents. |

#### `display` — Output formatting

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `color` | bool | `true` | Enable colored output (status indicators, diffs, warnings). Disable for piping to files. Overridden by `--no-color` flag. |
| `format` | string | `table` | Default output format: `table`, `json`, or `compact`. Overridden by `--json` flag. |
| `verbose` | bool | `false` | Show additional detail in output (response headers, timing breakdown). Overridden by `-v` flag. |

#### `thresholds` — Warning thresholds

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `ssl_warn_days` | int | `30` | Show SSL certificate expiry warning when days remaining is below this value. Certs with more days left are hidden from output. Red warning at half this value (e.g., <15 days at default). Set to `0` to always hide, or `365` to always show. |

#### `headers` — Custom HTTP headers

Key-value pairs added to every HTTP request. Useful for authentication tokens, custom identifiers, or bypassing certain WAF rules.

```yaml
headers:
  Authorization: Bearer my-api-token
  Accept-Language: en-US
```

### Data storage

All data lives in `~/.upp/upp.db` (SQLite). Back up by copying the file, query with any SQLite client, or export via `upp export`.

---

## Running as a Background Service

To run Upp in the background and survive reboots, set it up as a systemd service.

**One-liner setup:**

```bash
sudo tee /etc/systemd/system/upp-monitor.service << 'EOF'
[Unit]
Description=Upp Website Monitor
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/upp daemon
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl enable --now upp-monitor
```

**Check status:**

```bash
sudo systemctl status upp-monitor
```

**View logs:**

```bash
journalctl -u upp-monitor -f
```

**Stop / restart:**

```bash
sudo systemctl stop upp-monitor
sudo systemctl restart upp-monitor
```

### Removing the Service

```bash
sudo systemctl stop upp-monitor
sudo systemctl disable upp-monitor
sudo rm /etc/systemd/system/upp-monitor.service
sudo systemctl daemon-reload
```

---

## Uninstalling Upp

To completely remove Upp from your system:

```bash
# 1. Stop and remove the service (if running)
sudo systemctl stop upp-monitor
sudo systemctl disable upp-monitor
sudo rm /etc/systemd/system/upp-monitor.service
sudo systemctl daemon-reload

# 2. Remove the binary
sudo rm /usr/local/bin/upp
# Or if installed via go install:
rm $(go env GOPATH)/bin/upp

# 3. Remove data and config
rm -rf ~/.upp
rm -rf ~/.config/upp
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
cd upp
make build
# hack away
```

---

## License

[MIT](LICENSE)
