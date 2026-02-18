# 🐕 Watchdog

A Swiss Army knife CLI for website **uptime monitoring** and **change detection**. Combines the functionality of [Uptime Kuma](https://github.com/louislam/uptime-kuma) and [changedetection.io](https://github.com/dgtlmoon/changedetection.io) into a single, lightweight command-line tool.

**Designed for both humans and AI agents.** Every command supports `--json` for structured output.

## Why Watchdog?

| | Uptime Kuma | changedetection.io | **Watchdog** |
|---|---|---|---|
| Install | Docker + Web UI | Docker + Web UI | **Single binary** |
| Dependencies | Node.js, SQLite | Python, Playwright | **None** |
| Interface | Web browser | Web browser | **Terminal / JSON** |
| AI-friendly | ❌ API only | ❌ API only | **✅ Native CLI + JSON** |
| Uptime monitoring | ✅ | ❌ | **✅** |
| Change detection | ❌ | ✅ | **✅** |
| Combined | Need both | Need both | **All-in-one** |

## Features

- **Uptime Monitoring** — HTTP(s) status codes, response times, availability percentage
- **Change Detection** — Content diffing, CSS selector targeting, hash-based change tracking
- **Multiple Check Types** — HTTP, TCP, Ping, DNS
- **SSL Certificate Monitoring** — Days until expiry warnings
- **Live Dashboard** — `watchdog watch` for real-time terminal monitoring
- **Notifications** — Webhook, Slack, Telegram, Discord, shell commands
- **Daemon Mode** — Background service with scheduled checks
- **AI-Friendly** — `--json` flag on all commands for structured, parseable output
- **Zero Dependencies** — Single binary, no Docker, no runtime requirements
- **Cross-Platform** — macOS (Intel + Apple Silicon) and Linux (x86_64 + ARM64)
- **SQLite Storage** — Persistent history at `~/.watchdog/watchdog.db`
- **Shell Completions** — Bash, Zsh, Fish, PowerShell
- **Config File** — `~/.config/watchdog/config.yml` for persistent defaults

## Quick Start

### Install

**From source (requires Go 1.24+):**
```bash
go install github.com/naru-bot/watchdog@latest
```

**From binary releases:**
```bash
# macOS (Apple Silicon)
curl -LO https://github.com/naru-bot/watchdog/releases/latest/download/watchdog_darwin_arm64.tar.gz
tar xzf watchdog_darwin_arm64.tar.gz
sudo mv watchdog /usr/local/bin/

# Linux (x86_64)
curl -LO https://github.com/naru-bot/watchdog/releases/latest/download/watchdog_linux_amd64.tar.gz
tar xzf watchdog_linux_amd64.tar.gz
sudo mv watchdog /usr/local/bin/
```

**Build from source:**
```bash
git clone https://github.com/naru-bot/watchdog.git
cd watchdog
make build
# Binary is at ./watchdog
```

### Initialize (optional)

```bash
watchdog init
# Creates ~/.config/watchdog/config.yml with default settings
```

### Add targets

```bash
# Basic HTTP monitoring
watchdog add https://example.com --name "My Site"

# Monitor a specific CSS element for changes
watchdog add https://example.com/pricing --name "Pricing" --selector "div.price"

# TCP port check
watchdog add 192.168.1.1:3306 --type tcp --name "MySQL"

# DNS resolution check
watchdog add example.com --type dns --name "DNS Check"

# Ping check
watchdog add 8.8.8.8 --type ping --name "Google DNS"

# Custom interval (every 60 seconds)
watchdog add https://api.example.com/health --name "API" --interval 60
```

### Run checks

```bash
# Check all targets
watchdog check

# Check specific target
watchdog check "My Site"

# JSON output (for AI agents / scripts)
watchdog check --json
```

### View status

```bash
watchdog status                      # All targets, last 24h
watchdog status "My Site"            # Specific target
watchdog status --period 7d          # Last 7 days
watchdog status --json               # JSON output
```

### Live dashboard

```bash
watchdog watch                       # Auto-refresh every 30s
watchdog watch --refresh 10          # Refresh every 10s
```

### View content changes

```bash
watchdog diff "My Site"              # Unified diff (colored)
watchdog diff "My Site" --json       # Structured diff
```

### Check history

```bash
watchdog history "My Site"           # Last 20 checks
watchdog history "My Site" -l 100    # Last 100 checks
```

### Pause / Resume

```bash
watchdog pause "My Site"             # Pause monitoring
watchdog unpause "My Site"           # Resume monitoring
```

### Notifications

```bash
# Webhook
watchdog notify add --name alerts --type webhook \
  --config '{"url":"https://hooks.slack.com/services/..."}'

# Telegram
watchdog notify add --name tg --type telegram \
  --config '{"bot_token":"123:ABC","chat_id":"-100123"}'

# Discord
watchdog notify add --name discord --type discord \
  --config '{"webhook_url":"https://discord.com/api/webhooks/..."}'

# Shell command (variables: {target}, {url}, {status}, {message})
watchdog notify add --name logger --type command \
  --config '{"command":"echo \"{target} is {status}\" >> /var/log/watchdog.log"}'

# List / Remove
watchdog notify list
watchdog notify remove alerts
```

### Daemon mode

```bash
watchdog daemon                      # Foreground
nohup watchdog daemon &              # Background
```

### Export data

```bash
watchdog export --json > backup.json
watchdog export --format csv > data.csv
```

### Other commands

```bash
watchdog list                        # List all targets (alias: ls)
watchdog remove "My Site"            # Remove target
watchdog version                     # Version info
watchdog completion bash             # Shell completions
```

## All Commands

| Command | Description |
|---------|-------------|
| `init` | Initialize configuration file |
| `add <url>` | Add a URL to monitor |
| `remove <target>` | Remove a monitored target |
| `list` | List all monitored targets |
| `check [target]` | Run checks (all or specific) |
| `status [target]` | Show uptime stats and summary |
| `watch` | Live-updating terminal dashboard |
| `diff <target>` | Show content changes between snapshots |
| `history <target>` | Show check history |
| `pause <target>` | Pause monitoring |
| `unpause <target>` | Resume monitoring |
| `notify add\|list\|remove` | Manage notification channels |
| `export` | Export data as JSON or CSV |
| `daemon` | Run as background service |
| `completion` | Generate shell completion scripts |
| `version` | Print version |

## Global Flags

| Flag | Description |
|------|-------------|
| `--json` | Output in JSON format (all commands) |
| `--no-color` | Disable colored output |
| `-v, --verbose` | Verbose output |
| `-q, --quiet` | Suppress non-essential output |

## Configuration

Run `watchdog init` to generate `~/.config/watchdog/config.yml`:

```yaml
defaults:
  interval: 300        # Default check interval (seconds)
  type: http           # Default check type
  timeout: 30          # HTTP timeout (seconds)
  retry_count: 1       # Retries before marking as down
  user_agent: watchdog/1.0

display:
  color: true          # Colored terminal output
  format: table        # Default output format
  verbose: false
```

## AI Agent Usage

Watchdog is designed to be used by AI agents and automation scripts. The `--json` flag produces consistent, structured output on every command.

### Quick integration

```bash
# Add + check in one go
watchdog add https://api.example.com --name api --json
watchdog check api --json

# Parse with jq
watchdog check --json | jq '.[] | select(.status == "down")'
watchdog status --json | jq '.[] | select(.uptime_percent < 99)'

# Get changed targets
watchdog check --json | jq '.[] | select(.changed == true)'
```

### JSON output format

All `--json` output follows consistent patterns:
- **Lists** → JSON arrays: `[{...}, {...}]`
- **Single items** → JSON objects: `{...}`
- **Errors** → `{"error": "message"}`
- **Timestamps** → RFC3339 format

### Example: check output

```json
[
  {
    "target": "My Site",
    "url": "https://example.com",
    "status": "up",
    "status_code": 200,
    "response_time_ms": 142,
    "content_hash": "a1b2c3...",
    "changed": false,
    "ssl_days_left": 85
  }
]
```

### Cron integration

```bash
# Check every 5 minutes, alert on failures
*/5 * * * * watchdog check --json | jq -e '.[] | select(.status == "down")' && echo "ALERT" | mail -s "Site down" admin@example.com
```

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

## Data Storage

All data is stored in `~/.watchdog/watchdog.db` (SQLite). You can:
- Back up by copying the file
- Query directly with any SQLite client
- Export via `watchdog export`

## Cross-Compilation

```bash
make cross
# Produces:
#   watchdog-darwin-arm64   (macOS Apple Silicon)
#   watchdog-darwin-amd64   (macOS Intel)
#   watchdog-linux-amd64    (Linux x86_64)
#   watchdog-linux-arm64    (Linux ARM64)
```

## Tech Stack

- **Language:** [Go](https://go.dev)
- **CLI Framework:** [Cobra](https://github.com/spf13/cobra)
- **Database:** SQLite via [modernc.org/sqlite](https://modernc.org/sqlite) (pure Go, zero CGO)
- **HTML Parsing:** [goquery](https://github.com/PuerkitoBio/goquery)
- **Config:** [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3)
- **Diffing:** Custom LCS-based line diff engine

## License

MIT — see [LICENSE](LICENSE)
