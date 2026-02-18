# 🐕 Watchdog

A Swiss Army knife CLI for website **uptime monitoring** and **change detection**. Combines the functionality of [Uptime Kuma](https://github.com/louislam/uptime-kuma) and [changedetection.io](https://github.com/dgtlmoon/changedetection.io) into a single, lightweight command-line tool.

**Designed for both humans and AI agents.** Every command supports `--json` for structured output.

## Features

- **Uptime Monitoring** — HTTP(s) status codes, response times, availability percentage
- **Change Detection** — Content diffing, CSS selector targeting, hash-based change tracking
- **Multiple Check Types** — HTTP, TCP, Ping, DNS
- **SSL Certificate Monitoring** — Days until expiry
- **Notifications** — Webhook, Slack, Telegram, Discord, shell commands
- **Daemon Mode** — Background service with scheduled checks
- **AI-Friendly** — `--json` flag on all commands for structured, parseable output
- **Zero Dependencies** — Single binary, no Docker, no runtime requirements
- **Cross-Platform** — macOS (Intel + Apple Silicon) and Linux (x86_64 + ARM64)
- **SQLite Storage** — Persistent history at `~/.watchdog/watchdog.db`

## Quick Start

### Install

**From source (requires Go 1.24+):**
```bash
go install github.com/cheryeong/watchdog@latest
```

**From binary releases:**
```bash
# macOS (Apple Silicon)
curl -LO https://github.com/cheryeong/watchdog/releases/latest/download/watchdog_darwin_arm64.tar.gz
tar xzf watchdog_darwin_arm64.tar.gz
sudo mv watchdog /usr/local/bin/

# Linux (x86_64)
curl -LO https://github.com/cheryeong/watchdog/releases/latest/download/watchdog_linux_amd64.tar.gz
tar xzf watchdog_linux_amd64.tar.gz
sudo mv watchdog /usr/local/bin/
```

### Add a target

```bash
# Basic HTTP monitoring
watchdog add https://example.com --name "My Site"

# Monitor a specific element (change detection)
watchdog add https://example.com/pricing --name "Pricing" --selector "div.price"

# TCP port check
watchdog add 192.168.1.1:3306 --type tcp --name "MySQL"

# DNS check
watchdog add example.com --type dns --name "DNS Check"

# Ping check
watchdog add 8.8.8.8 --type ping --name "Google DNS"

# Custom interval (every 60 seconds)
watchdog add https://api.example.com/health --name "API Health" --interval 60
```

### Run checks

```bash
# Check all targets
watchdog check

# Check specific target
watchdog check "My Site"
```

### View status

```bash
# Summary of all targets
watchdog status

# Specific target with 7-day stats
watchdog status "My Site" --period 7d

# JSON output
watchdog status --json
```

### View changes

```bash
# See what changed (unified diff)
watchdog diff "My Site"

# JSON diff output
watchdog diff "My Site" --json
```

### Check history

```bash
watchdog history "My Site"
watchdog history "My Site" --limit 50
```

### Notifications

```bash
# Webhook
watchdog notify add --name alerts --type webhook \
  --config '{"url":"https://hooks.slack.com/services/..."}'

# Telegram
watchdog notify add --name telegram --type telegram \
  --config '{"bot_token":"123:ABC","chat_id":"-100123"}'

# Discord
watchdog notify add --name discord --type discord \
  --config '{"webhook_url":"https://discord.com/api/webhooks/..."}'

# Shell command
watchdog notify add --name logger --type command \
  --config '{"command":"echo \"{target} is {status}\" >> /var/log/watchdog.log"}'

# List configured notifications
watchdog notify list

# Remove
watchdog notify remove alerts
```

### Daemon mode

```bash
# Run in foreground
watchdog daemon

# Run in background
nohup watchdog daemon > /var/log/watchdog.log 2>&1 &

# Or with systemd (see below)
```

### Export data

```bash
# JSON export
watchdog export --json > backup.json

# CSV export
watchdog export --format csv > data.csv
```

### Other commands

```bash
watchdog list          # List all targets
watchdog remove "My Site"  # Remove a target
watchdog version       # Show version
```

## All Commands

| Command | Description |
|---------|-------------|
| `add <url>` | Add a URL to monitor |
| `remove <name\|url\|id>` | Remove a monitored target |
| `list` | List all monitored targets |
| `check [target]` | Run checks (all or specific) |
| `status [target]` | Show uptime stats and summary |
| `diff <target>` | Show content changes between snapshots |
| `history <target>` | Show check history |
| `notify add\|list\|remove` | Manage notification channels |
| `export` | Export data as JSON or CSV |
| `daemon` | Run as background service |
| `version` | Print version |

## Global Flags

| Flag | Description |
|------|-------------|
| `--json` | Output in JSON format (all commands) |

## AI Agent Usage

Watchdog is designed to be used by AI agents and automation scripts. The `--json` flag produces structured output on every command.

### Example: AI agent monitoring workflow

```bash
# Add targets
watchdog add https://myapp.com --name myapp --json
# {"id":1,"name":"myapp","url":"https://myapp.com","type":"http",...}

# Run checks and parse results
watchdog check --json
# [{"target":"myapp","url":"https://myapp.com","status":"up","response_time_ms":142,"changed":false}]

# Get status
watchdog status --json
# [{"target":"myapp","uptime_percent":99.8,"avg_response_ms":145,"total_checks":288}]

# Check for content changes
watchdog diff myapp --json
# {"has_changes":true,"summary":"+3 lines, -1 lines","changes":[...]}

# Export everything
watchdog export --json
```

### Integration with AI tools

```bash
# Use in a shell pipeline
watchdog check --json | jq '.[] | select(.status == "down")'

# Cron job: check every 5 minutes
*/5 * * * * /usr/local/bin/watchdog check >/dev/null 2>&1

# One-liner: add + check + report
watchdog add https://api.example.com --name api --json && watchdog check api --json
```

### Machine-readable output format

All `--json` output follows consistent patterns:
- **Lists** return JSON arrays: `[{...}, {...}]`
- **Single items** return JSON objects: `{...}`
- **Errors** return: `{"error": "message"}`
- **All timestamps** are RFC3339/ISO8601 format

## Systemd Service

Create `/etc/systemd/system/watchdog.service`:

```ini
[Unit]
Description=Watchdog - Website Monitor
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
sudo systemctl enable watchdog
sudo systemctl start watchdog
```

## Data Storage

All data is stored in `~/.watchdog/watchdog.db` (SQLite). You can:
- Back it up by copying the file
- Query it directly with any SQLite client
- Export via `watchdog export`

## Building from Source

```bash
git clone https://github.com/cheryeong/watchdog.git
cd watchdog
make build

# Cross-compile for all platforms
make cross
```

## Tech Stack

- **Language:** Go
- **CLI Framework:** [Cobra](https://github.com/spf13/cobra)
- **Database:** SQLite via [modernc.org/sqlite](https://modernc.org/sqlite) (pure Go, no CGO)
- **HTML Parsing:** [goquery](https://github.com/PuerkitoBio/goquery)
- **Diffing:** Custom LCS-based line diff

## License

MIT
