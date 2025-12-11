Output git-pull.gif

Require git
Require git-remote-oci

Set Shell "bash"
Set FontSize 20
Set Width 1650
Set Height 750
Set TypingSpeed 0.09
Set PlaybackSpeed 1.2

Type "git show-ref refs/heads/main"
Sleep 500ms
Enter
Sleep 500ms

Type "ls -l .git/objects/pack/*.pack"
Sleep 500ms
Enter
Sleep 1s

Type "git pull oci://127.0.0.1:5000/demo/source:gnoci main"
Sleep 500ms
Enter
Sleep 1s

Type "git show-ref refs/heads/main"
Sleep 500ms
Enter
Sleep 2s

Type "ls -l .git/objects/pack/*.pack"
Sleep 500ms
Enter
Sleep 6s
