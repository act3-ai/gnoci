# Final Presentation Demo


## Preparations

- Build from source: `go build -o bin/git-remote-oci ./cmd/git-remote-oci`
- Ensure availability on `$PATH`: `sudo cp ./bin/git-remote-oci ~/go/bin/git-remote-oci`
- Startup local registry: use alias `startreg`

## Demo

### Push Local Repo to OCI Remote

```console
$ git push --all oci://127.0.0.1:5000/repo/demo:gitoci
To oci://127.0.0.1:5000/repo/demo:gitoci
 * [new branch]      act3-pt/blueprints/render-orphan -> act3-pt/blueprints/render-orphan
 * [new branch]      command-fetch -> command-fetch
 * [new branch]      command-list -> command-list
 * [new branch]      command-push -> command-push
 * [new branch]      design-docs -> design-docs
 * [new branch]      main -> main
 * [new branch]      setup-project -> setup-project
```

#### View Remote

##### Manifest

```console
$ oras manifest fetch --plain-http 127.0.0.1:5000/repo/demo:gitoci | jq
```

```json
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "artifactType": "application/vnd.ai.act3.git.repo.v1+json",
  "config": {
    "mediaType": "application/vnd.ai.act3.git.config.v1+json",
    "digest": "sha256:3b63a77a5140b79e652d7d8846314b3aef4c392939e5e9f163771812f4347bde",
    "size": 1168
  },
  "layers": [
    {
      "mediaType": "application/vnd.ai.act3.git.pack.v1",
      "digest": "sha256:fd024234ac54e760f69d4cf35992aaeb9a0f018cd1455e594c6ba00bc66b0801",
      "size": 3236797,
      "annotations": {
        "org.opencontainers.image.title": "pack-4b7b2beac8ba8812d8d42b6dfd170bfb27c03fa5.pack"
      }
    }
  ],
  "annotations": {
    "org.opencontainers.image.created": "1970-01-01T00:00:00Z"
  }
}
```

##### Config

```console
$ oras manifest fetch-config --plain-http 127.0.0.1:5000/repo/demo:gitoci | jq
```

```json
{
  "heads": {
    "refs/heads/act3-pt/blueprints/render-orphan": {
      "commit": "d2d51a405e5168f1945a02035c8f6089da53cb04",
      "layer": "sha256:fd024234ac54e760f69d4cf35992aaeb9a0f018cd1455e594c6ba00bc66b0801"
    },
    "refs/heads/command-fetch": {
      "commit": "56db9a50f127dae1a0da3563cb205a45ee077208",
      "layer": "sha256:fd024234ac54e760f69d4cf35992aaeb9a0f018cd1455e594c6ba00bc66b0801"
    },
    "refs/heads/command-list": {
      "commit": "8741ea9910f853de6ea90709e2195a0558b5f3ce",
      "layer": "sha256:fd024234ac54e760f69d4cf35992aaeb9a0f018cd1455e594c6ba00bc66b0801"
    },
    "refs/heads/command-push": {
      "commit": "fa789ae0105a816244715e76f692cef04a6701a6",
      "layer": "sha256:fd024234ac54e760f69d4cf35992aaeb9a0f018cd1455e594c6ba00bc66b0801"
    },
    "refs/heads/design-docs": {
      "commit": "e89ee162c4f805fd6ee997ce0e26677083a135cd",
      "layer": "sha256:fd024234ac54e760f69d4cf35992aaeb9a0f018cd1455e594c6ba00bc66b0801"
    },
    "refs/heads/main": {
      "commit": "57309ba1957e98ae2e09fb1b867605310ac87412",
      "layer": "sha256:fd024234ac54e760f69d4cf35992aaeb9a0f018cd1455e594c6ba00bc66b0801"
    },
    "refs/heads/setup-project": {
      "commit": "ced7d65b2dbdd91b9231b8edab51bac53415de2f",
      "layer": "sha256:fd024234ac54e760f69d4cf35992aaeb9a0f018cd1455e594c6ba00bc66b0801"
    }
  },
  "tags": {}
}
```

### Clone Repository from OCI Remote

```console
$ mkdir /tmp/clone1 && cd /tmp/clone1
```

```console
$ git clone oci://127.0.0.1:5000/repo/demo:gitoci
Cloning into 'gitoci'...
```

#### View Clone

Notice how git configures the remote for us. With this configured remote, we can simply use the `origin` shortname when needed.

```console
$ ls
gitoci

$ cd gitoci

$ git remote -v
origin	oci://127.0.0.1:5000/repo/demo:gitoci (fetch)
origin	oci://127.0.0.1:5000/repo/demo:gitoci (push)

$ git status
On branch main
Your branch is up to date with 'origin/main'.

nothing to commit, working tree clean

$ git log
commit 57309ba1957e98ae2e09fb1b867605310ac87412 (HEAD -> main, origin/main)
Author: nathan-joslin <nathan.joslin@udri.udayton.edu>
Date:   Thu Jul 31 23:38:50 2025 -0400

    chore(release): prepare for v0.1.0

...
```

#### Make an extra clone

Create a second clone so we can fetch changes later.

```console
$ mkdir ../../clone2 && cd ../../clone2/gitoci

$ git clone oci://127.0.0.1:5000/repo/demo:gitoci
Cloning into 'gitoci'...
```

### Push Changes to OCI Remote

Make a new commit:

```console
$ echo "foo" > foo.txt && git add --all && git commit -m "add foo.txt"
[main 4648088] add foo.txt
 1 file changed, 1 insertion(+)
 create mode 100644 foo.txt
```

Push to remote:

```console
$ git push
To oci://127.0.0.1:5000/repo/demo:gitoci
   57309ba..4648088  main -> main
```

#### View Updated Remote

`refs/heads/main` has been updated to the new commit and packfile layer.

```console
$ oras manifest fetch-config --plain-http 127.0.0.1:5000/repo/demo:gitoci | jq
```

```json
{
  "heads": {
    "refs/heads/act3-pt/blueprints/render-orphan": {
      "commit": "d2d51a405e5168f1945a02035c8f6089da53cb04",
      "layer": "sha256:fd024234ac54e760f69d4cf35992aaeb9a0f018cd1455e594c6ba00bc66b0801"
    },
    "refs/heads/command-fetch": {
      "commit": "56db9a50f127dae1a0da3563cb205a45ee077208",
      "layer": "sha256:fd024234ac54e760f69d4cf35992aaeb9a0f018cd1455e594c6ba00bc66b0801"
    },
    "refs/heads/command-list": {
      "commit": "8741ea9910f853de6ea90709e2195a0558b5f3ce",
      "layer": "sha256:fd024234ac54e760f69d4cf35992aaeb9a0f018cd1455e594c6ba00bc66b0801"
    },
    "refs/heads/command-push": {
      "commit": "fa789ae0105a816244715e76f692cef04a6701a6",
      "layer": "sha256:fd024234ac54e760f69d4cf35992aaeb9a0f018cd1455e594c6ba00bc66b0801"
    },
    "refs/heads/design-docs": {
      "commit": "e89ee162c4f805fd6ee997ce0e26677083a135cd",
      "layer": "sha256:fd024234ac54e760f69d4cf35992aaeb9a0f018cd1455e594c6ba00bc66b0801"
    },
    "refs/heads/main": {
      "commit": "464808805844d28970f2b66ffc5c92fd6298439c",
      "layer": "sha256:f3660938306a15edc16234676586b602ba89f9a12c5e487f707a4e9fddb8faef"
    },
    "refs/heads/setup-project": {
      "commit": "ced7d65b2dbdd91b9231b8edab51bac53415de2f",
      "layer": "sha256:fd024234ac54e760f69d4cf35992aaeb9a0f018cd1455e594c6ba00bc66b0801"
    }
  },
  "tags": {}
}
```

### Fetch from OCI Remote in Second Repo

```console
$ cd ../../clone1/gitoci

$ git fetch --all
From oci://127.0.0.1:5000/repo/demo:gitoci
   57309ba..4648088  main       -> origin/main

$ git pull
Updating 57309ba..4648088
Fast-forward
 foo.txt | 1 +
 1 file changed, 1 insertion(+)
 create mode 100644 foo.txt

$ git log -n 1
commit 464808805844d28970f2b66ffc5c92fd6298439c (HEAD -> main, origin/main)
Author: nathan-joslin <nathan.joslin@udri.udayton.edu>
Date:   Fri Aug 1 10:20:48 2025 -0400

    add foo.txt
```

### Push with VSCode

```console
$ code .
```

- Create bar.txt
- Commit
- Push

```console
$ git log -n 1
commit 3d21de2ef9f78a4950b62a891c8a9fa992f05f34 (HEAD -> main, origin/main)
Author: nathan-joslin <nathan.joslin@udri.udayton.edu>
Date:   Fri Aug 1 10:31:41 2025 -0400

    add bar.txt
```

#### View Remote Updated with VSCode

`refs/heads/main` has been updated to the new commit and packfile layer.

```console
$ oras manifest fetch-config --plain-http 127.0.0.1:5000/repo/demo:gitoci | jq
```

```json
{
  "heads": {
    "refs/heads/act3-pt/blueprints/render-orphan": {
      "commit": "d2d51a405e5168f1945a02035c8f6089da53cb04",
      "layer": "sha256:fd024234ac54e760f69d4cf35992aaeb9a0f018cd1455e594c6ba00bc66b0801"
    },
    "refs/heads/command-fetch": {
      "commit": "56db9a50f127dae1a0da3563cb205a45ee077208",
      "layer": "sha256:fd024234ac54e760f69d4cf35992aaeb9a0f018cd1455e594c6ba00bc66b0801"
    },
    "refs/heads/command-list": {
      "commit": "8741ea9910f853de6ea90709e2195a0558b5f3ce",
      "layer": "sha256:fd024234ac54e760f69d4cf35992aaeb9a0f018cd1455e594c6ba00bc66b0801"
    },
    "refs/heads/command-push": {
      "commit": "fa789ae0105a816244715e76f692cef04a6701a6",
      "layer": "sha256:fd024234ac54e760f69d4cf35992aaeb9a0f018cd1455e594c6ba00bc66b0801"
    },
    "refs/heads/design-docs": {
      "commit": "e89ee162c4f805fd6ee997ce0e26677083a135cd",
      "layer": "sha256:fd024234ac54e760f69d4cf35992aaeb9a0f018cd1455e594c6ba00bc66b0801"
    },
    "refs/heads/main": {
      "commit": "3d21de2ef9f78a4950b62a891c8a9fa992f05f34",
      "layer": "sha256:5a3508330df9deaf551eb5d2345b359939cb86fb10139cc50d8498a008beb7a8"
    },
    "refs/heads/setup-project": {
      "commit": "ced7d65b2dbdd91b9231b8edab51bac53415de2f",
      "layer": "sha256:fd024234ac54e760f69d4cf35992aaeb9a0f018cd1455e594c6ba00bc66b0801"
    }
  },
  "tags": {}
}
```

### Pull with VSCode

```console
$ git reset HEAD~2
```

- Undo changes
- Pull

```console
$ git log -n 2
commit 3d21de2ef9f78a4950b62a891c8a9fa992f05f34 (HEAD -> main, origin/main)
Author: nathan-joslin <nathan.joslin@udri.udayton.edu>
Date:   Fri Aug 1 10:31:41 2025 -0400

    add bar.txt

commit 464808805844d28970f2b66ffc5c92fd6298439c
Author: nathan-joslin <nathan.joslin@udri.udayton.edu>
Date:   Fri Aug 1 10:20:48 2025 -0400

    add foo.txt
```

## Behind the Scenes Push

In an existing repository: `git push --all oci://127.0.0.1:5000/repo/demo:gitoci`.

On Stdin/Stdout:

- `->` Indicates a command written by Git to `git-remote-oci`.
- `<-` Indicates a response written by `git-remote-oci` to Git.

Cleaned Output:

```console
-> capabilities

<- option
<- fetch
<- push

-> option progress true

<- unsupported

-> option verbosity 1

<- ok

-> list for-push

# git-remote-oci: fetch the OCI config metadata containing all remote head and tag references.

# remote does not yet exist, so no refs are listed. A newline indicates the end of ref listing.
<- \n

# git: provides a batch of push commands, terminated by a newline. Zero or more batches may occur.
-> push refs/heads/act3-pt/blueprints/render-orphan:refs/heads/act3-pt/blueprints/render-orphan
-> push refs/heads/command-fetch:refs/heads/command-fetch
-> push refs/heads/command-list:refs/heads/command-list
-> push refs/heads/command-push:refs/heads/command-push
-> push refs/heads/design-docs:refs/heads/design-docs
-> push refs/heads/main:refs/heads/main
-> push refs/heads/setup-project:refs/heads/setup-project
-> \n

# git-remote-oci: for each push reference, resolve the local commit it refers to
# git-remote-oci: build a packfile containing all objects reachable by referenced commits
# git-remote-oci: for each push reference, update the OCI config appropriately
# git-remote-oci: push  to OCI reference tag
# git-remote-oci: respond to git
<- ok refs/heads/act3-pt/blueprints/render-orphan
<- ok refs/heads/command-fetch
<- ok refs/heads/command-list
<- ok refs/heads/command-push
<- ok refs/heads/design-docs
<- ok refs/heads/main
<- ok refs/heads/setup-project

-> \n
# EOF
# program complete
```

