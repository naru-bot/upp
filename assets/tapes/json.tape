# JSON Output for AI Agents
Output assets/json.gif

Set FontSize 16
Set Width 1200
Set Height 650
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms

Sleep 500ms
Type "watchdog add https://example.com --name 'My Site'"
Enter
Sleep 1.5s
Type "watchdog add https://api.example.com/health --name 'API'"
Enter
Sleep 1.5s

Type "watchdog check"
Enter
Sleep 2s

Type "# Every command speaks JSON natively"
Enter
Sleep 1s

Type "watchdog check --json | jq ."
Enter
Sleep 3s

Type `watchdog status --json | jq '.[] | {name: .target, uptime: .uptime_percent}'`
Enter
Sleep 3s

Sleep 1s
