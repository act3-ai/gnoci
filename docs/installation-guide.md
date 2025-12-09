# Git Remote Helper for OCI Registries Installation Guide

This Installation Guide provides the steps necessary to set up `git-remote-oci` and `git-lfs-remote-oci`.

Regardless of installation method:

- `git-remote-oci` *must* be made available on `$PATH` to be accessible by `git`.
- `git-lfs-remote-oci` *should* be made available on `$PATH` to be accessible by `git-lfs`, however setting a path with `git config lfs.customtransfer.oci.path <path>` is sufficient.

## Installing From Source

### Using go native builds

As with all go projects, you can clone, build, and move to `$PATH` if desired.

1. Clone source repository
   - `git clone git@github.com:act3-ai/gnoci.git`
2. Build from source
   - `go build -o bin/git-remote-oci ./cmd/git-remote-oci`
   - `go build -o bin/git-lfs-remote-oci ./cmd/git-lfs-remote-oci`
3. Make build(s) available on `$PATH`
   - `sudo cp ~/go/bin/git-remote-oci`
   - `sudo cp ~/go/bin/git-lfs-remote-oci`

### Using dagger

Dagger is a tool we use to build reusable pipelines, utilized in both CI and local dev environments. You can utilize our pipeline build process to build from source. See [dagger installation docs](https://docs.dagger.io/getting-started/installation).

The default build platform is `linux/amd64`:

- `dagger call build-git export --path <installation-path>`
- `dagger call build-git-lfs export --path <installation-path>`

For other platforms use `--platform "OS/ARCH"`:

- `dagger call build-git --platform "linux/arm64" export --path <installation-path>`
- `dagger call build-git-lfs --platform "linux/arm64" export --path <installation-path>`
