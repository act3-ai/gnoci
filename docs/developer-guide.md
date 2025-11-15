# Git Remote Helper for OCI Registries Developer Guide

## Design Patterns

Git Remote Helper for OCI Registries is organized into three layers:

- [`cmd` Package](#cmd-package): CLI commands defined using the `cobra` package
- [`actions` Package](#actions-package): Main functionality of Git Remote Helper for OCI Registries
- [Other Packages](#other-packages): Purpose-separated components of Git Remote Helper for OCI Registries functionality

### `cmd` Package

The `cmd` package uses [`cobra`](https://pkg.go.dev/github.com/spf13/cobra) to define the command line interface for Git Remote Helper for OCI Registries.

> [`cmd` Package](./../cmd/git-remote-oci/cmd)

### `actions` Package

The `actions` package contains the core functionality of Git Remote Helper for OCI Registries. The commands defined in `cmd` run and "action" in the `actions` package.

> [`actions` Package](./../pkg/actions)

### Other Packages

The other packages in the `pkg` folder contain smaller components of the functionality of Git Remote Helper for OCI Registries.

> [Other Packages](./../pkg)

## Build git-remote-oci

It is recommended to build using `dagger` for the convienence of setting build flags, however native go builds may also be done.

Regardless of the build process, the executable *must* be named `git-remote-oci` in order to be recognized by `git`.

### Build for a single platform

By default, the build target platfrom is `linux/amd64`.

```console
dagger call build export --path bin/git-remote-oci
```

### Build for a specific platform

Specify any platform supported by go, see `go tool list dist` for all supported platforms.

```console
dagger call build --platform linux/amd64 export --path bin/git-remote-oci
```

### Build for all platforms

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

### Build using go

Build flags may be added as desired, the following serves as a baseline:

`go build -o bin/git-remote-oc ./cmd/git-remote-oci`

## Linting

The recommended linting method is to use dagger. Linters may be ran all at once in parallel or individually.

### Run all linters

`dagger call lint all`

### Run a single linter

- golangci-lint: `dagger call lint go`
- yamllint: `dagger call lint yaml`
- shellcheck: `dagger call lint shell`
- markdown: `dagger call lint markdown`
- govulncheck: `dagger call lint vulncheck`

## Testing

The recommended testing method is to use `dagger`.

### Run all tests

`dagger call test all`

### Unit Tests

`dagger call test unit`

### Functional Tests

<!-- Describe how to run functional tests -->

## Debugging

The following environment variables are helpful to track git and git-lfs interactions with our remote helpers:

- `GIT_TRACE=1`
- `GIT_TRANSFER_TRACE=1`
- `GIT_LFS_DEBUG=1`

## Releasing

## Miscellanous

### Progress Example

Ran with progress interval `time.Millisecond * 50`. Progress updates for total bytes uploaded and the delta of bytes since the last progress message. This is shown by the standard LFS progress bar.

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
