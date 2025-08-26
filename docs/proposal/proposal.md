# Proposal: Storing Git Repositories as OCI Artifacts

**Author:** Nathan Joslin

**Advisors:** Dr. Kyle Tarplee

## Problem Description

Moving Git repositories into air-gapped environments is done for a variety of reasons: building from source, repository mirroring, and shifting development environments. To achieve this goal, existing tools aim to solve three problems: efficient data transfer, efficient storage, and minimal maintenance burden. However, most of these tools lack the ability to scale and often solve one problem well while falling behind in the other two.

Instead of addressing all three problems directly, this project aims to achieve the main goal by combining the strengths of existing solutions.

## Existing and Related Solutions

### Zarf

Zarf is a tool that facilitates software delivery on systems without an internet connection. They provide support for transferring git repositories across air-gap boundaries. However, using Zarf requires learning a large tool, with its own "ecosystem", and they do not appear to support incremental updates for git repositories. Additionally, they do not support git-lfs.

Sources:

- [Git-repositories docs](https://docs.zarf.dev/ref/components/#git-repositories)
- [Zarf codebase](https://github.com/zarf-dev/zarf/tree/main/src/internal/git)

### Git Bundles

Git bundles, native to git, are archives containing references and objects used for the offline transfer of git objects without an active network connection. With "shallow" bundle support, they can be used for incremental updates. For simple use cases git bundles are a great candidate, however their use at scale (i.e. many repositories or large repositories), requires an in-depth knowledge of git and comes with significant organizational penalities for efficient transfers. Furthermore, resolving the git object a reference is associated with requires obtaining and interfacing with the entire bundle, which is not ideal for an incremental update approach. For this reason, this project aims to shift the proof-of-concept data model provided by ASCE Data Tool to use packfiles rather than bundles. Given that bundles are an extension of packfiles to include data on Git references, changing the data model should provided significant benefits.

For example:

From a clone:

*Disclaimer: subject to internet speeds*:

```console
$ time git clone
Cloning into 'linux'...
remote: Enumerating objects: 10920246, done.
remote: Counting objects: 100% (62/62), done.
remote: Compressing objects: 100% (42/42), done.
remote: Total 10920246 (delta 38), reused 23 (delta 20), pack-reused 10920184 (from 2)
Receiving objects: 100% (10920246/10920246), 5.33 GiB | 10.02 MiB/s, done.
Resolving deltas: 100% (8883947/8883947), done.
Updating files: 100% (89751/89751), done.

real   11m45.203s
user   14m9.259s
sys    1m12.536s
``

With powers of `1024`:

```console
$ du -s linux
7689188 linux
```

Sources:

- [Git bundle manpage](https://git-scm.com/docs/git-bundle)

### ORAS

As a Cloud Native Computing Foundation sandbox project, ORAS has proven their worth within the open source community. They approach OCI artifacts as a "graph copy" problem, providing both a CLI and Go package for efficent distribution of OCI artifacts.

## Proposed Solution

Define a specification for storing git repositories as OCI artifacts as well as implement a git remote helper to facilitate pushing and fetching from a git repository stored in an OCI registry. By using packfiles, rather than bundles, we can support efficient workflows by quickly resolving the state of local and remote references to perform the minimum mount of data transfer.

Using git with an OCI remote helper looks as follows after installing the utility. While adding and cloning the repository requires specifying the remote helper, subsequent git commands may be used in their typical manner:

```bash
# Clone directly from OCI registry
$ git clone oci://example.registry.com/repository/name:tag

# Add an OCI registry as a remote
$ git remote add <remote-name> oci://example.registry.com/repository/name:tag
```

```bash
# Fetch from a remote OCI registry
$ git fetch [<repository>] [<refspec>...]

# Push to a remote OCI registry
$ git push [<repository>] [<refspec>...]
```

### Why OCI

The Open Container Initiative (OCI) has become the industry standard for the storage, transfer, and execution of container images. In recent years, the OCI specification has been used for alternative artifact types, extending beyond container images to: helm repositories, cryptographic signatures, software bill of materials, and more. By defining an OCI specification and implementing an interface for git to interact with OCI registries we can utilize existing data transfer tools.

### Why Git Remote Helper

Git remote helpers allow users to install programs that extend git's native capabilities to interface with alternative remote repository formats; in our case OCI. By using a git remote helper users can use git as they normally would, e.g. with VSCode, avoiding a need to learn new commands from a new tool. The [manual page](https://git-scm.com/docs/gitremote-helpers) includes well-defined instructions for implementing a git remote helper.

No known existing remote helpers for this application exist.

## Proof-of-Concept

A prototype specification and tooling is provided by the [ASCE Data Tool](https://github.com/act3-ai/data-tool/tree/main/internal/git), which has shown promising results. Although it is the closest existing solution, it only solves part of the problem. It is capable of storing git repositories as OCI artifacts, with support for git-lfs files. Like Zarf, it uses git references (tags and branches) or hashes to store a part of or an entire git repository in an OCI registry. However, it does not function as a git remote helper and lacks in some areas of speed and efficiency.

```console
$ export IMAGE_NAME=torvalids-linux-$(uuidgen)

$ ace-dt git to-oci git@github.com:torvalds/linux.git ttl.sh/${IMAGE_NAME}:1h

$ ace-dt git from-oci ttl.sh${IMAGE_NAME} git@github.com:user-example.git
```

### Pros

- Facilitates efficient incremental updates by resolving the minimum data needed to perform updates based on the previous synchronization.
- Supports git-lfs.
- Prototype specification.

### Cons

- Slow to resolve the minimum data needed to perform an incremental update.
- As a subcommand of a larger tool, it requires learning new command groups. Not a git remote helper.

## Goals

- Git OCI Specification.
- Git remote helper implementation.
- Git-LFS support.
- Fast and efficient.
- 50% test coverage.

### Bonus Goals

The following goals are potential enhancements if time permits.

- Homebrew tap and formula, for easy installation.
- Git-lfs bundles.
- go-git bundle support (depending on spec).
- 80% test coverage.

## Implementation Details

The git remote helper will be written in Go for the following reasons:

- Familiarity and Experience with Go and external packages (see below)
- Well-written standard library
- Multi-Architecture support

### External Packages

- [ORAS](https://github.com/oras-project/oras-go): Efficient OCI artifact transfers, with private registry support.
- [sourcegraph/conc](https://github.com/sourcegraph/conc): Structured concurrency, for speed and efficiency.
- [go-git](https://github.com/go-git/go-git): Efficient handling of git objects.

### Git Remote Helper Capabilities

Remote helpers are not required to support all capabilities and features. The following tables provide a brief overview of the capabilities *necessary* for a MVP, complimented by more in-depth descriptions for the decision made.

The background research performed for this proposal led to the conclusion that it is likely only the `connect` capabilities for fetching/pushing packfiles to/from an OCI registry is necessary. As a result, it is likely that other capabilities will be added to increase the efficientcy of `gnoci` as a remote helper. Furthermore, the following capabilities are not required to support `git-lfs`. Given it's relatively widepspread usage of support for `git-lfs` is a requirement of this project.

The *Support* column indicates if supporting capability is necessary to solve the problem, a nice-to-have, or no plans to support.

Legend:

- Yes - Support is necessary for MVP.
- Maybe - Support may provide value, if time permits.
- No - Support is not reasonable for project goals.

#### Pushing

| Capability     | Support     |
| ------------- | :-------------: |
| `connect` | Yes |
| `stateless-connect` | No |
| `push` | Maybe |
| `export` | Maybe |
| `no-private-update` | No |

##### `connect` - Yes <!-- markdownlint-disable-line MD024 -->

Connect utilizes git's packfile protocol. The proof-of-concept provided by [ASCE Data Tool](https://github.com/act3-ai/data-tool/tree/main/internal/git) uses git bundles, files indended for the "offline" transfer of git objects. Given that bundles are pack-files extended to include git references, support for *pushing* pack-files is key for efficient data transfer.

##### `stateless-connect` - No <!-- markdownlint-disable-line MD024 -->

Described as experimental and intended for internal use. As such support for this capability is not planned.

##### `push` - Maybe

Used to update remote history and references with local objects. The packfile protocol supported by the `connect` capability is better suited for OCI registries given that many registries limit the sizes of manifests. However, the benefits of packfiles in the context of OCI is limited to reduced metadata and efficient uploads. As such, if a user preferrs faster fetches over pulls, then `push` may provide value given the inclusion of the `fetch` capability.

##### `export` - Maybe

Used to push git objects over a fast-import stream to a remote. This capability may be useful for receving objects from git quickly to build packfiles to be pushed to OCI. More research regarding this capability is needed. Git prefers `push` over `export`.

##### `no-private-update` - No

Adds support for disabling private namespace updates. Managing private namespaces is not within the scope of this project. See [git namespaces](https://git-scm.com/docs/gitnamespaces) for more information.

See the [Capabilities for Pushing](https://git-scm.com/docs/gitremote-helpers#_capabilities_for_pushing) section of the manpage for more information.

#### Fetching

| Capability     | Support     |
| ------------- | :-------------: |
| `connect` | Yes |
| `stateless-connect` | No |
| `fetch` | Maybe |
| `import` | Maybe |
| `check-connectivity` | Maybe |
| `get` | Maybe |

##### `connect` - Yes <!-- markdownlint-disable-line MD024 -->

Connect utilizes git's packfile protocol. The proof-of-concept provided by [ASCE Data Tool](https://github.com/act3-ai/data-tool/tree/main/internal/git) uses git bundles, files indended for the "offline" transfer of git objects. Given that bundles are pack-files extended to include git references, support for *receiving* pack-files is key for efficient data transfer.

##### `stateless-connect` - No <!-- markdownlint-disable-line MD024 -->

Described as experimental and intended for internal use. As such support for this capability is not planned.

##### `fetch` - Maybe

Used to update local history and references with remote objects. The packfile protocol supported by the `connect` capability is better suited for OCI registries given that many registries limit the sizes of manifests. However, the benefits of packfiles in the context of OCI is limited to reduced metadata and efficient uploads. As such, if a user preferrs faster fetches over pulls, then `fetch` may provide value given the inclusion of the `push` capability.

##### `import` - Maybe

Used to fetch git objects over a fast-import stream from a remote. This capability may be useful for receving packfiles from OCI and sending objects to git. More research regarding this capability is needed.

##### `check-connectivity` - Maybe

Used to validate that a received packfile, from a clone, is self contained. Although not necessary, this feature may be helpful when cloning from an OCI registry. Alternatively, users can initialize an empty repositry and fetch from the remote - effectively cloning the fetched references.

##### `get` - Maybe

Used to fetch files from a URI. Although not necessary for an MVP, this capability may be helpful to select users.

See the [Capabilities for Fetching](https://git-scm.com/docs/gitremote-helpers#_capabilities_for_fetching) section of the manpage for more information.

#### Miscellaneous

| Capability     | Support     |
| ------------- | :-------------: |
| `option` | Maybe |
| `refspec` | Maybe |
| `bidi-import` | Maybe |
| `export-marks` | Maybe |
| `import-marks` | Maybe |
| `signed-tags` | Maybe |
| `object-format` | Maybe |

##### `option` - Maybe

Potentially helpful for users who wish to control verbosity and history depth.

##### `refspec` - Maybe

Required for the `export` capability, optional for `import`. Adds support for constraining references to private namespaces. Not required for an MVP.

##### `bidi-import` - Maybe

Modifies the `import` capability. Although not necessary, this capability could improve efficiency of `import`. The [description provided by git](https://git-scm.com/docs/gitremote-helpers#Documentation/gitremote-helpers.txt-embidi-importem) is better read directly rather than summarized here.

##### `export-marks` - Maybe

Modifies the `export` capability. The output of internal marks table provided by git could be useful for effeciently constructing bundles.

##### `import-marks` - Maybe

More information is needed on this capability. Intuitively, it would seem to be the reverse of `export-marks`. However, the description claims it modifies `export` rather than `import`. More research is needed regarding this capability.

##### `signed-tags` - Maybe

Modifies the `export` capability regarding how signed tags are passed to `git-fast-export`. Not necessary for an MVP, but looking into how this capability, `fast-import`, and `fast-export` handle signed tags is worthwhile.

##### `object-format` - Maybe

Indicates the helper can use explicit hash algorithm extensions when interacting with the remote. Although not necessary for an MVP, this capability may be useful for supporting alternative hash algorithms supported by OCI.

See the [Miscellaneous Capabilities](https://git-scm.com/docs/gitremote-helpers#_miscellaneous_capabilities) section of the manpage for more information.

## Other Resources

- [Blog post](https://rovaughn.github.io/2015-2-9.html) on implementing a git remote helper
