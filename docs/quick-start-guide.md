# Git Remote Helper for OCI Registries Quick Start Guide

## Purpose

This Quick Start Guide is for users who have already [installed Git Remote Helper for OCI Registries](installation-guide.md) and who are ready to take advantage of its features.

You will be guided through the steps necessary to begin using `git-remote-oci`.

1. [Configuration](#configuration)
2. [Initial Usage](#initial-usage)

## Configuration

*Coming soon...*

## Initial Usage

`git-remote-oci` is intended to be used as a [git remote helper](https://git-scm.com/docs/gitremote-helpers), it is rare a user interacts with it directly. Instead, users configure `git` to use the `oci` protocol and interact with `git` as normal.

To use `git` with an OCI registry, whenever a `git` command allows a remote URL as an option specify the Git remote with a `oci` protocol prefix along with an OCI tag reference, e.g. `oci://<registry>/<repository>/<name>:<tag>`.

See [usage examples](./user-guide.md#usage).

### Recommended Usage

It is recommended to configure an OCI registry tag reference as a Git remote:

```console
$ cd path/to/local/repo
$ git remote add <name> oci://<registry>/<repository>/<name>:tag
```

Whenever `git` requires or allows a remote option, simply use `<name>`, e.g. `git push <name>`

## Additional Resources

- [Documentation](./../README.md#documentation)
- [Support](./../README.md#support)
