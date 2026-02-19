# Notifications Setup
Output assets/notifications.gif

Set FontSize 18
Set Width 950
Set Height 500
Set LineHeight 1.3
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 35ms

Sleep 500ms
Type `watchdog notify add --name tg --type telegram --config '{"bot_token":"123:ABC","chat_id":"-100"}'`
Enter
Sleep 2.5s

Type "watchdog notify list"
Enter
Sleep 3s

Type "watchdog notify remove tg"
Enter
Sleep 2s

Sleep 1s
