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
- [Tutorial: Monitor Your First Website in 60 Seconds](#tutorial-monitor-your-first-website-in-60-seconds)
- [AI Agent Integration](#ai-agent-integration)
- [Comparison](#comparison)
- [Installation](#installation)
- [All Commands](#all-commands)
- [Configuration](#configuration)
- [Systemd Service](#systemd-service)
- [Tech Stack](#tech-stack)
- [Contributing](#contributing)
- [License](#license)

---

## Why Watchdog?

You just want to know if your website is up. Maybe get alerted when a pricing page changes. Simple, right?

**Nope.** In 2024, here's what that actually looks like:

1. Spin up an Uptime Kuma Docker container for monitoring
2. Spin up a changedetection.io Docker container for change detection
3. Configure both through their web UIs
4. Keep both running 24/7, eating RAM on your server
5. Open a browser every time you want to check status
6. Try to integrate with your scripts or AI agent — good luck parsing those web APIs
7. Realize you're running **two separate services with two databases** just to watch three websites

**Watchdog is the tool I built because I got tired of this nonsense.**

One 17MB binary. Works from your terminal. Works over SSH. Works in cron jobs. Works with AI agents out of the box. Install it, add a URL, done. No YAML manifests, no Docker Compose files, no port forwarding, no reverse proxies.

If you've ever mass-`docker rm`'d monitoring containers in frustration, this is for you.

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

## Comparison

| | Uptime Kuma | changedetection.io | **Watchdog** |
|---|---|---|---|
| Install | Docker + Web UI | Docker + Web UI | **Single binary** |
| Dependencies | Node.js, SQLite | Python, Playwright | **None** |
| Interface | Web browser | Web browser | **Terminal / JSON** |
| AI-friendly | ❌ API only | ❌ API only | **✅ Native CLI + JSON** |
| Uptime monitoring | ✅ | ❌ | **✅** |
| Change detection | ❌ | ✅ | **✅** |
| Combined | Need both 🤷 | Need both 🤷 | **All-in-one** |
| RAM usage | ~150MB+ | ~300MB+ | **~10MB** |
| Works over SSH | ❌ | ❌ | **✅** |

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
  --name        Target name
  --type        Check type: http, tcp, ping, dns
  --interval    Check interval in seconds (default: 300)
  --selector    CSS selector for change detection
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

## Systemd Service

Create `/etc/systemd/system/watchdog-monitor.service`:

```ini
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
```

```bash
sudo systemctl enable watchdog-monitor
sudo systemctl start watchdog-monitor
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
