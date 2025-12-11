Output git-remote-add.gif

Require git
Require git-remote-oci

Set Shell "bash"
Set FontSize 20
Set Width 1100
Set Height 350
Set TypingSpeed 0.09
Set PlaybackSpeed 1.2

Type "git remote add oci-registry oci://127.0.0.1:5000/demo/source:gnoci"
Sleep 500ms
Enter

Type "git ls-remote oci-registry"
Sleep 500ms
Enter
Sleep 4s