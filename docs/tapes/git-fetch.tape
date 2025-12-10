Output git-fetch.gif

Require git
Require git-remote-oci

Set Shell "bash"
Set FontSize 20
Set Width 1450
Set Height 600
Set TypingSpeed 0.09
Set PlaybackSpeed 1.2

Type "mkdir demo"
Sleep 500ms
Enter
Sleep 500ms

Type "cd demo"
Sleep 500ms
Enter
Sleep 500ms

Type "git init"
Sleep 500ms
Enter
Sleep 500ms

Type "git fetch oci://127.0.0.1:5000/demos:gnoci main"
Sleep 500ms
Enter
Sleep 2s

Type "ls -lah .git/objects/pack"
Sleep 500ms
Enter
Sleep 5s
