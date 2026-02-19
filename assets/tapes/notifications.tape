# Notification Channels Setup
Output assets/notifications.gif

Set FontSize 16
Set Width 1200
Set Height 650
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms

Sleep 500ms
Type `watchdog notify add --name tg --type telegram --config '{"bot_token":"123:ABC","chat_id":"-100123"}'`
Enter
Sleep 2s

Type `watchdog notify add --name discord --type discord --config '{"webhook_url":"https://discord.com/api/webhooks/123/abc"}'`
Enter
Sleep 2s

Type `watchdog notify add --name alerts --type webhook --config '{"url":"https://hooks.slack.com/services/T00/B00/xxx"}'`
Enter
Sleep 2s

Type "watchdog notify list"
Enter
Sleep 3s

Type "watchdog notify remove alerts"
Enter
Sleep 2s

Sleep 1s
