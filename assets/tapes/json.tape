# JSON Output for AI Agents
Output assets/json.gif

Set FontSize 18
Set Width 1200
Set Height 550
Set LineHeight 1.3
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms

Sleep 500ms
Type "upp check --json | jq ."
Enter
Sleep 4s

Type "upp status --json | jq '.[].uptime_percent'"
Enter
Sleep 3s

Sleep 1s
