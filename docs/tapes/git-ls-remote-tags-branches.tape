Output git-ls-remote-tags-branches.gif

Require git
Require git-remote-oci

Set Shell "bash"
Set FontSize 20
Set Width 1100
Set Height 400
Set TypingSpeed 0.09
Set PlaybackSpeed 1.2

Type "git status"
Sleep 500ms
Enter
Sleep 1s
Type "git ls-remote --tags oci://127.0.0.1:5000/demo/source:gnoci"
Sleep 500ms
Enter
Sleep 1s
Type "git ls-remote --branches oci://127.0.0.1:5000/demo/source:gnoci"
Sleep 500ms
Enter
Sleep 5s