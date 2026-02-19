# Change Detection + Diff
Output assets/change-detection.gif

Set FontSize 18
Set Width 900
Set Height 500
Set LineHeight 1.4
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms

Sleep 500ms
Type "watchdog add https://example.com/pricing --name 'Pricing' --selector 'div.price'"
Enter
Sleep 2s

Type "watchdog check 'Pricing'"
Enter
Sleep 3s

Type "watchdog diff 'Pricing'"
Enter
Sleep 4s

Sleep 1s
