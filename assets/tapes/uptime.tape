# Uptime Monitoring with Sparklines
Output assets/uptime.gif

Set FontSize 17
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms

Sleep 500ms
Type "watchdog add https://myapp.io --name 'Production'"
Enter
Sleep 1.5s

Type "watchdog add https://staging.myapp.io --name 'Staging'"
Enter
Sleep 1.5s

Type "watchdog add https://api.myapp.io/health --name 'API'"
Enter
Sleep 1.5s

Type "watchdog check"
Enter
Sleep 3s

Type "watchdog status --period 7d"
Enter
Sleep 4s

Sleep 1s
