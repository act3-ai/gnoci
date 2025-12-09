# Git Remote Helper for OCI Registries Developer Guide

This document is intended for developers to familiarize themselves with standard practices for this project.

## Table of Contents

- [Git Remote Helper for OCI Registries Developer Guide](#git-remote-helper-for-oci-registries-developer-guide)
  - [Table of Contents](#table-of-contents)
  - [Design Patterns](#design-patterns)
  - [Developer Workflows](#developer-workflows)
    - [Taskfile](#taskfile)
    - [Build git-remote-oci and git-lfs-remote-oci](#build-git-remote-oci-and-git-lfs-remote-oci)
      - [Build for a single platform](#build-for-a-single-platform)
      - [Build for a specific platform](#build-for-a-specific-platform)
      - [Build for all platforms](#build-for-all-platforms)
      - [Build using go](#build-using-go)
    - [Linting](#linting)
      - [Run all linters](#run-all-linters)
      - [Run a single linter](#run-a-single-linter)
    - [Testing](#testing)
      - [Run all tests](#run-all-tests)
      - [Unit Tests](#unit-tests)
      - [Functional Tests](#functional-tests)
  - [Debugging](#debugging)
  - [Releasing](#releasing)
  - [Miscellanous](#miscellanous)
    - [Progress Example](#progress-example)

## Design Patterns

Git Remote Helper for OCI Registries is organized into four main groups:

- [`cmd/*`](../cmd): CLI commands for `git-remote-oci` and `git-lfs-remote-oci` defined using the `cobra` package, utilizes `internal/actions`.
- [`internal/actions/*`](../internal/actions): Main functionality of `git-remote-oci` and `git-lfs-remote-oci`, utilizes `internal/*` and `pkg/*`. Ideally as small as possible, most logic should live in `internal/*` or `pkg/*`.
- [`internal/*](../internal/): Helper packages for actions. Most code should live here unless there's a concrete reason to make it public.
- [`pkg/*`](../pkg/): Intentionally public interfaces that may use useful for other projects.

## Developer Workflows

This section defines standard practices for building, linting, and testing.

### Taskfile

[`Taskfile.dist.yaml`](../Taskfile.dist.yaml) contains shortcuts to the majority of actions defined in this section. Note that most tasks rely on other tools, e.g. `dagger`.

You can install `task` with homebrew, e.g. `brew install task`.

Example: `task lint` to run all linters.

See [Taskfile Docs](https://taskfile.dev/docs/guide) for more information about task files.

### Build git-remote-oci and git-lfs-remote-oci

It is recommended to build using `dagger` for the convienence of setting build flags, however native go builds may also be done.

Regardless of the build process, the Git remote helperexecutable *must* be named `git-remote-oci` in order to be recognized by `git`. The `git-lfs` remote helper traditionally follows the same convention, `git-lfs-remote-oci`, but may be named anything as long as `git-lfs` is properly configured; see [Quick Start Guide](./quick-start-guide.md) for configuration details.

#### Build for a single platform

By default, the default build target platfrom is `linux/amd64`.

```console
dagger call build export --path bin/git-remote-oci
```

#### Build for a specific platform

Specify any platform supported by go, see `go tool list dist` for all supported platforms.

```console
dagger call build --platform linux/amd64 export --path bin/git-remote-oci
```

#### Build for all platforms

By default, builds are made for `linux/amd64`, `linux/arm64`, and `darwin/arm64`.

```console
dagger call build-platforms --platforms=linux/amd64,linux/arm64,darwin/arm64 export --path bin
```

Executables are placed within platform directories:

```console
$ tree -a bin
bin
├── darwin-arm64
│   └── git-remote-oci
├── linux-amd64
│   └── git-remote-oci
├── linux-arm64
│   └── git-remote-oci
```

#### Build using go

Build flags may be added as desired, the following serves as a baseline:

`go build -o bin/git-remote-oci ./cmd/git-remote-oci`

### Linting

The recommended linting method is to use dagger.

#### Run all linters

`dagger call lint all` runs all linters in parallel.

#### Run a single linter

- golangci-lint: `dagger call lint go`
- yamllint: `dagger call lint yaml`
- shellcheck: `dagger call lint shell`
- markdown: `dagger call lint markdown`
- govulncheck: `dagger call lint vulncheck`

### Testing

The recommended testing method is to use `dagger`.

#### Run all tests

`dagger call test all`

#### Unit Tests

`dagger call test unit`

#### Functional Tests

<!-- Describe how to run functional tests -->

## Debugging

The following environment variables are helpful to track git and git-lfs interactions with our remote helpers:

- `GIT_TRACE=1`
- `GIT_TRANSFER_TRACE=1`
- `GIT_LFS_DEBUG=1`

## Releasing

Run `task release` to be taken through the release process.

Note that `GITHUB_TOKEN` must be set.

## Miscellanous

### Progress Example

Ran with progress interval `time.Millisecond * 50` (default is 500ms). Progress updates for total bytes uploaded and the delta of bytes since the last progress message. This is shown by the standard LFS progress bar.

```console
20:23:31.209588 trace git-lfs: xfer: Custom adapter worker 0 sending message: {"event":"upload","oid":"22c3c9aa4876932c4daeecf036b8c4780693fbb32555b61ccb9bd73599562a70","size":29400000,"path":"/home/nathan/code/testdata/lfstestsource/.git/lfs/objects/22/c3/22c3c9aa4876932c4daeecf036b8c4780693fbb32555b61ccb9bd73599562a70","action":null}
20:23:31.330411 trace git-lfs: xfer: Custom adapter worker 0 received response: {"event":"progress","oid":"22c3c9aa4876932c4daeecf036b8c4780693fbb32555b61ccb9bd73599562a70","bytesSoFar":10780672,"bytesSinceLast":10780672}
20:23:31.380962 trace git-lfs: xfer: Custom adapter worker 0 received response: {"event":"progress","oid":"22c3c9aa4876932c4daeecf036b8c4780693fbb32555b61ccb9bd73599562a70","bytesSoFar":16875520,"bytesSinceLast":6094848}
20:23:31.431571 trace git-lfs: xfer: Custom adapter worker 0 received response: {"event":"progress","oid":"22c3c9aa4876932c4daeecf036b8c4780693fbb32555b61ccb9bd73599562a70","bytesSoFar":21856256,"bytesSinceLast":4980736}
20:23:31.480682 trace git-lfs: xfer: Custom adapter worker 0 received response: {"event":"progress","oid":"22c3c9aa4876932c4daeecf036b8c4780693fbb32555b61ccb9bd73599562a70","bytesSoFar":28278784,"bytesSinceLast":6422528}
20:23:31.530176 trace git-lfs: xfer: Custom adapter worker 0 received response: {"event":"progress","oid":"22c3c9aa4876932c4daeecf036b8c4780693fbb32555b61ccb9bd73599562a70","bytesSoFar":29400000,"bytesSinceLast":1121216}
20:23:31.590561 trace git-lfs: xfer: Custom adapter worker 0 received response: {"event":"complete","oid":"22c3c9aa4876932c4daeecf036b8c4780693fbb32555b61ccb9bd73599562a70"}
20:23:31.590589 trace git-lfs: xfer: adapter "oci" worker 0 finished job for "22c3c9aa4876932c4daeecf036b8c4780693fbb32555b61ccb9bd73599562a70"

```
