Output git-lfs-pull.gif

Require git
Require git-remote-oci
Require tree

Set Shell "bash"
Set FontSize 20
Set Width 1100
Set Height 850
Set TypingSpeed 0.09
Set PlaybackSpeed 1.2

Type "tree -a .git/lfs/objects"
Sleep 500ms
Enter
Sleep 1s

Type "git pull oci-registry main"
Sleep 500ms
Enter
Sleep 1s

Type "tree -a .git/lfs/objects"
Sleep 500ms
Enter
Sleep 6s
