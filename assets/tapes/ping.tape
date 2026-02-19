# Quick Ping Diagnostics
Output assets/ping.gif

Set FontSize 17
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms

Sleep 500ms
Type "# Quick check — no database, no setup"
Enter
Sleep 1s

Type "watchdog ping https://api.example.com/health"
Enter
Sleep 3s

Type "watchdog ping https://httpbin.org/status/503"
Enter
Sleep 3s

Type "watchdog ping https://example.com"
Enter
Sleep 3s

Sleep 1s
