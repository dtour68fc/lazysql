Output assets/demo.gif

Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 50ms

Hide
Type "clear"
Enter
Show

Type "./lazysql"
Sleep 500ms
Enter
Sleep 2500ms

# --- Connection Manager: show navigation
Type "j"
Sleep 800ms
Type "j"
Sleep 800ms
Type "k"
Sleep 800ms

# --- Show the connection form and the 3 modes
Type "e"
Sleep 1200ms
Type "m"
Sleep 1500ms
Type "m"
Sleep 1500ms
Type "m"
Sleep 1200ms
Escape
Sleep 800ms

# --- Connect to a working database
Type "k"
Sleep 400ms
Enter
Sleep 3500ms

# --- Explorer: expand the database
Type "l"
Sleep 1500ms

# --- Switch to the editor pane
Ctrl+[
Sleep 200ms
Shift+Tab
Sleep 1200ms

# --- Write a query in the Vim editor
Type "i"
Sleep 300ms
Type "SELECT datname, numbackends, xact_commit FROM pg_stat_database WHERE datname IS NOT NULL ORDER BY xact_commit DESC LIMIT 8;"
Sleep 1500ms
Escape
Sleep 600ms

# --- Visual-line select and run
Type "V"
Sleep 800ms
Ctrl+r
Sleep 3500ms

# --- Switch to viewer and scroll
Shift+Tab
Sleep 1200ms
Type "jjjj"
Sleep 1000ms
Type "ll"
Sleep 1500ms

# --- Quit
Ctrl+c
Sleep 800ms
