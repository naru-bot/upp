# Hero Demo: add → check → status
Output assets/demo.gif

Set FontSize 22
Set Width 900
Set Height 500
Set LineHeight 1.4
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms

Sleep 500ms
Type "watchdog add https://example.com --name 'My Site'"
Enter
Sleep 2.5s

Type "watchdog check"
Enter
Sleep 3s

Type "watchdog status"
Enter
Sleep 4s

Sleep 1s
