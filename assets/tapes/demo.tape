# Hero Demo: add → check → status
Output assets/demo.gif

Set FontSize 17
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms

Sleep 500ms
Type "# 🐕 Watchdog — Uptime monitoring in your terminal"
Enter
Sleep 1s

Type "watchdog add https://example.com --name 'My Site'"
Enter
Sleep 2s

Type "watchdog add https://api.example.com/health --name 'API'"
Enter
Sleep 2s

Type "watchdog check"
Enter
Sleep 3s

Type "watchdog status"
Enter
Sleep 4s

Sleep 1s
