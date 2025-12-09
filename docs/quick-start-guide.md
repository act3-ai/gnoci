# Git Remote Helper for OCI Registries Quick Start Guide

## Purpose

This Quick Start Guide is for users who have already [installed git-remote-oci and/or git-lfs-remote-oci](installation-guide.md) and are ready to take advantage of its features.

You will be guided through the steps necessary to begin using `git-remote-oci` and/or `git-lfs-remote-oci`.

1. [Configuration](#configuration)
2. [Initial Usage](#initial-usage)

## Configuration

### Recommended git-remote-oci Configuration

It is recommended to configure an OCI registry tag reference as a Git remote. Without this, users *must* specify the `oci://` protocol to invoke `git-remote-oci` (Git discovers this on `$PATH`, following the convention `git-remote-<protocol>`).

```console
$ cd path/to/local/repo
$ git remote add <shortname> oci://<registry>/<repository>/<name>:tag
```

For more information on OCI references see the [OCI Distribution Spec: Pulling Manifest](https://github.com/opencontainers/distribution-spec/blob/main/spec.md#pulling-manifests), note that digest references are *not* acceptable for our use case.

Whenever `git` requires or allows a remote option, simply use `<name>`, e.g. `git push <name>`.

### Required git-lfs-remote-oci Configuration

The following is an overview of [git-lfs custom transfer protocol configuration](https://github.com/git-lfs/git-lfs/blob/main/docs/custom-transfers.md#using-a-custom-transfer-type-without-the-api-server).

- `remote.<remote>.lfsurl <oci://url>` *SHOULD* be used to configure a remote. Alternatively, `lfs.url <oci://url>` will configure *all* `git-lfs` transfers to point to an OCI registry.
- `lfs.<oci://url>.standalonetransferagent oci` *SHOULD* be used to configure an OCI registry as a remote. Alternatively, `lfs.standalonetransferagent oci` will configure `git-lfs-remote-oci` to be the only available transfer agent.
- `lfs.customtransfer.oci.path <path/to/git-lfs-remote-oci>` *MUST* be set to the installation path of `git-lfs-remote-oci`.
- `lfs.customtransfer.oci.args` *MUST* not be set (value `""` is acceptable). `git-lfs-remote-oci` does not accept additional arguments. All configuration is done through environment variables or configuration files.
- `lfs.customtransfer.oci.concurrent` *MUST* be set to `false` (the default is `true`). Due to the design of the OCI data model, this feature is not supported.
- `lfs.customtransfer.oci.direction` can be set to any acceptable value (`download`, `upload`, or `both`; default is `both`). `git-lfs-remote-oci` supports both downloading and uploading LFS files.

## Initial Usage

`git-remote-oci` is intended to be used as a [git remote helper](https://git-scm.com/docs/gitremote-helpers), it is rare a user interacts with it directly. Instead, users configure `git` to use the `oci` protocol and interact with `git` as normal.

To use `git` with an OCI registry, whenever a `git` command allows a remote URL as an option specify the Git remote with a `oci` protocol prefix along with an OCI tag reference, e.g. `oci://<registry>/<repository>/<name>:<tag>`.

See [usage examples](./user-guide.md#usage).

## Additional Resources

- [Documentation](./../README.md#documentation)
- [Support](./../README.md#support)
