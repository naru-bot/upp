# Change Detection + Diff
Output assets/change-detection.gif

Set FontSize 17
Set Width 1200
Set Height 650
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms

Sleep 500ms
Type `watchdog add https://example.com/pricing --name "Pricing" --selector "div.price"`
Enter
Sleep 2s

Type `watchdog check "Pricing"`
Enter
Sleep 3s

Type `watchdog check "Pricing"`
Enter
Sleep 3s

Type `watchdog diff "Pricing"`
Enter
Sleep 4s

Sleep 1s
