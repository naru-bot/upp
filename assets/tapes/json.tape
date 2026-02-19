# JSON Output for AI Agents
Output assets/json.gif

Set FontSize 20
Set Width 950
Set Height 550
Set LineHeight 1.3
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms

Sleep 500ms
Type "watchdog check --json | jq ."
Enter
Sleep 4s

Type "watchdog status --json | jq '.[].uptime_percent'"
Enter
Sleep 3s

Sleep 1s
