# Git Remote Helper for OCI Registries User Guide

This Quick Start Guide is for users who have already [installed Git Remote Helper for OCI Registries](installation-guide.md) and who are ready to take advantage of its features.

You will be guided through the steps necessary to begin using `git-remote-oci` as well as example use cases.

1. [Configuration](#configuration)
2. [Usage](#usage)

## Overview

`git-remote-oci` is intended to be used as a [git remote helper](https://git-scm.com/docs/gitremote-helpers), it is rare a user interacts with it directly. Instead, users configure `git` to use the `oci` protocol and interact with `git` as normal.

## Configuration

### Configuring git-remote-oci Itself

*Coming soon...*

### Configuring Git to Use git-remote-oci

It is recommended to configure an OCI registry tag reference as a Git remote:

```console
$ cd path/to/local/repo
$ git remote add <name> oci://<registry>/<repository>/<name>:tag
```

## Usage

### Configured OCI Remote

Simply specify `<name>` when performing remote operations.

Example:

```bash
# Within a local git repo
git push <name> HEAD
```

### Without a Configured OCI Remote

Whenever a `git` command allows a remote URL as an option specify the Git remote with a `oci` protocol prefix along with an OCI tag reference, e.g. `oci://<registry>/<repository>/<name>:<tag>`.

## Examples

The following examples build off of each other.

### Push

Within an existing repository, push to `127.0.0.1:5000/repo/test:example-clone`:

```console
$ git push --all oci://127.0.0.1:5000/repo/test:example-clone
To oci://127.0.0.1:5000/repo/demo:gnoci8
 * [new branch]      act3-pt/blueprints/render-orphan -> act3-pt/blueprints/render-orphan
 * [new branch]      command-fetch -> command-fetch
 * [new branch]      command-list -> command-list
 * [new branch]      command-push -> command-push
 * [new branch]      design-docs -> design-docs
 * [new branch]      main -> main
 * [new branch]      setup-project -> setup-project
```

Git will not automatically create a new remote:

```console
$ git remote -v
origin  git@github.com:act3-ai/gnoci.git (fetch)
origin  git@github.com:act3-ai/gnoci.git (push)
```

### Clone

To clone a Git repository with OCI tag reference `127.0.0.1:5000/repo/test:example-clone`:

```console
$ git clone oci://127.0.0.1:5000/repo/test:example-clone
```

The cloned repository exists in `./example-clone`, with the `origin` remote added:

```console
$ ls
example-clone

$ cd example-clone

$ git remote -v
origin	oci://127.0.0.1:5000/repo/test:example-clone (fetch)
origin	oci://127.0.0.1:5000/repo/test:example-clone (push)
```

### List Remote

Building off of the [clone example](#clone):

```console
$ git ls-remote
From oci://127.0.0.1:5000/repo/test:example-clone
56db9a50f127dae1a0da3563cb205a45ee077208	refs/heads/command-fetch
8741ea9910f853de6ea90709e2195a0558b5f3ce	refs/heads/command-list
fa789ae0105a816244715e76f692cef04a6701a6	refs/heads/command-push
056a62e2dd0795884384ad562c5ca68a236be7e4	refs/heads/design-docs
21023d360200012cefcd8f077b3b24aea7cb20f2	refs/heads/main
ced7d65b2dbdd91b9231b8edab51bac53415de2f	refs/heads/setup-project
d2d51a405e5168f1945a02035c8f6089da53cb04	refs/heads/act3-pt/blueprints/render-orphan
```

### Push after modifications

Building off of the [clone example](#clone):

```console
$ echo "foobar" >> foo.txt

$ git add --all

$ git commit -m "testing push"
[main f03fa9b] testing push
 1 file changed, 1 insertion(+)
 create mode 100644 foo.txt

$ git push
Everything up-to-date
```

### Fetch

Building off of the [push after modifications example](#push-after-modifications).

Here we fetch in a secondary repository with older history:

```console
$ git fetch
From oci://127.0.0.1:5000/repo/test:example-clone
   21023d3..f03fa9b  main       -> origin/main

$ git log
commit 21023d360200012cefcd8f077b3b24aea7cb20f2 (HEAD -> main)
Author: Nathan Joslin <112950243+nathan-joslin@users.noreply.github.com>
Date:   Wed Jul 30 18:34:01 2025 -0400

    add fetch prototype (#33)
    
    includes some bugfixes
```

### Pull

Building off of the [fetch example](#fetch):

```console
$ git pull
Updating 21023d3..f03fa9b
Fast-forward
 foo.txt | 1 +
 1 file changed, 1 insertion(+)
 create mode 100644 foo.txt

$ git log
commit f03fa9b01da2a64ebc96419a36cbba71ae95f8eb (HEAD -> main, origin/main)
Author: nathan-joslin <nathan.joslin@udri.udayton.edu>
Date:   Thu Jul 31 18:46:18 2025 -0400

    testing push
```

## Additional Resources

- [Documentation](./../README.md#documentation)
- [Support](./../README.md#support)
