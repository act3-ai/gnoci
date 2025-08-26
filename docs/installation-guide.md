# Git Remote Helper for OCI Registries Installation Guide

This Installation Guide provides the steps necessary to set up `git-remote-oci`.

Regardless of installation method, `git-remote-oci` *must* be made available on `$PATH` to be accessible by `git`.

## Install From Srouce

1. Clone source repository
   - e.g. `git clone git@github.com:act3-ai/gnoci.git`
2. Build from source
   - e.g. `go build -o bin/git-remote-oci ./cmd/git-remote-oci`
3. Make build available on `$PATH`
   - e.g. `sudo cp ~/go/bin/git-remote-oci`

## Install with Homebrew

*Coming soon...*
