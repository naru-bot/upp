# TUI Dashboard Demo
Output assets/tui.gif

Set FontSize 16
Set Width 1200
Set Height 650
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms

Sleep 500ms
Type "watchdog add https://example.com --name 'My Site'"
Enter
Sleep 1s
Type "watchdog add https://api.example.com/health --name 'API'"
Enter
Sleep 1s
Type "watchdog add https://blog.example.com --name 'Blog'"
Enter
Sleep 1s

Type "watchdog check"
Enter
Sleep 2s

Type "watchdog tui"
Enter
Sleep 3s

# Navigate the TUI
Down
Sleep 800ms
Down
Sleep 800ms
Up
Sleep 800ms

# Open details
Enter
Sleep 2s

# Trigger a check
Type "c"
Sleep 2s

# Quit
Type "q"
Sleep 1s
