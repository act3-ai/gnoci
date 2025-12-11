Output git-lfs-push.gif

Require git
Require git-remote-oci

Set Shell "bash"
Set FontSize 20
Set Width 1100
Set Height 850
Set TypingSpeed 0.09
Set PlaybackSpeed 1.2

Type "git ls-remote oci://127.0.0.1:5000/demo/source:gnoci"
Sleep 500ms
Enter
Sleep 1s

Type "git status"
Sleep 500ms
Enter
Sleep 1s

Type "git add newfile.lfs"
Sleep 500ms
Enter
Sleep 500ms

Type 'git commit -m "pushing commit to OCI"'
Sleep 500ms
Enter
Sleep 500ms

Type "git push oci://127.0.0.1:5000/demo/source:gnoci main"
Sleep 500ms
Enter
Sleep 1s

Type "git ls-remote oci://127.0.0.1:5000/demo/source:gnoci"
Sleep 500ms
Enter
Sleep 6s