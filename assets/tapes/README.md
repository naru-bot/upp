# VHS Tape Files

These `.tape` files are used with [charmbracelet/vhs](https://github.com/charmbracelet/vhs) to generate terminal GIF demos for the README.

## Prerequisites

```bash
go install github.com/charmbracelet/vhs@latest
```

VHS also requires `ffmpeg` and `ttyd` — see [VHS installation docs](https://github.com/charmbracelet/vhs#installation).

## Generate GIFs

```bash
# Generate all demos
for tape in assets/tapes/*.tape; do
  vhs "$tape"
done

# Or generate individually
vhs assets/tapes/demo.tape
vhs assets/tapes/tui.tape
vhs assets/tapes/uptime.tape
vhs assets/tapes/change-detection.tape
vhs assets/tapes/ping.tape
vhs assets/tapes/json.tape
vhs assets/tapes/notifications.tape
```

Output GIFs are written to `assets/` (e.g. `assets/demo.gif`).

## Notes

- **Watchdog must be installed** and available in `$PATH` before recording
- Tapes add example targets, run checks, and show output — they're self-contained demos
- Theme: Catppuccin Mocha (dark, high contrast)
- Each tape runs ~10-15 seconds
- Edit tape files to adjust timing (`Sleep`) or typing speed (`Set TypingSpeed`)
