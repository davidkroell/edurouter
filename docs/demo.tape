# Where should we write the GIF?
Output demo.gif

# Set up a 1200x600 terminal with 14px font.
Set FontSize 18
Set Width 1200
Set Height 800
Set WindowBar Colorful
Set Margin 20
Set MarginFill "#ffffff"
Set BorderRadius 20
Set TypingSpeed 0.1

Type "sudo ./edurouter"

Sleep 100ms
Enter

Type "vers"
Tab@1000ms
Sleep 1000ms
Enter

Tab@500ms 4
Backspace 10

Sleep 100ms

Type "if add --interface wlp0s20f3 -a 192.168.0.50/24"
Enter
Sleep 100ms

Type "if li"
Tab@1000ms
Sleep 200ms
Enter

Type@200ms "route li"
Tab@1000ms
Sleep 200ms
Enter

Sleep 100ms
Type "ping 192.168.0.1"
Enter

Sleep 4000ms

Type "exit"
Sleep 100ms
Enter

