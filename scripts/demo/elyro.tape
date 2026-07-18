Output docs/assets/elyro-demo.gif

Set Shell "zsh"
Set FontSize 18
Set Width 1000
Set Height 620
Set Padding 24
Set Theme { "name": "Elyro", "black": "#1b1b24", "red": "#ff5f87", "green": "#5fd787", "yellow": "#ffd75f", "blue": "#5fafff", "magenta": "#d787ff", "cyan": "#5fd7d7", "white": "#e4e4e7", "brightBlack": "#62626d", "brightRed": "#ff87af", "brightGreen": "#87d7a5", "brightYellow": "#ffe087", "brightBlue": "#87bfff", "brightMagenta": "#e0a0ff", "brightCyan": "#87e0e0", "brightWhite": "#ffffff", "background": "#111118", "foreground": "#e4e4e7", "selectionBackground": "#343442", "cursorAccent": "#d787ff" }
Set TypingSpeed 35ms

Type "cd /tmp/elyro-terminal-demo"
Enter
Sleep 1s

Type "elyro up --toolchain go"
Enter
Sleep 3s

Type "elyro exec -- go version"
Enter
Sleep 2s

Type "elyro status"
Enter
Sleep 3s

Type "elyro down"
Enter
Sleep 2s
