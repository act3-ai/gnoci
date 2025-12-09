# Git as OCI Artifact Specification

The specification for storing Git repositories in OCI compliant registries follows the [OCI Image Specification](https://github.com/opencontainers/image-spec/blob/main/spec.md) and [OCI Distribution Specification](https://github.com/opencontainers/distribution-spec/blob/main/spec.md). In particular, the specification is packaged as defined by the [OCI Image Manifest Specification: Guidelines for Artifact Usage](https://github.com/opencontainers/image-spec/blob/main/manifest.md#guidelines-for-artifact-usage) (decision tree number 3).

It is strongly recommended readers of this document are familiar with the following subset of the [OCI Image Specification](https://github.com/opencontainers/image-spec/blob/main/spec.md):

- [OCI Image Manifest Specification](https://github.com/opencontainers/image-spec/blob/main/manifest.md), particularly the [Guidelines for Artifact Usage](https://github.com/opencontainers/image-spec/blob/main/manifest.md#guidelines-for-artifact-usage)
- [OCI Content Descriptors](https://github.com/opencontainers/image-spec/blob/main/descriptor.md#oci-content-descriptors)
- [OCI Referrers API](https://github.com/opencontainers/distribution-spec/blob/main/spec.md#listing-referrers)

## Table of Contents

- [Git as OCI Artifact Specification](#git-as-oci-artifact-specification)
  - [Table of Contents](#table-of-contents)
  - [Notational Conventions](#notational-conventions)
  - [Overview](#overview)
  - [Specification](#specification)
    - [OCI Manifest](#oci-manifest)
      - [Example OCI Manifest](#example-oci-manifest)
    - [OCI Config](#oci-config)
      - [Config Format](#config-format)
      - [Example OCI Config](#example-oci-config)
    - [OCI Layer](#oci-layer)
    - [LFS OCI Artifact Manifest](#lfs-oci-artifact-manifest)
      - [Example LFS OCI Manifest](#example-lfs-oci-manifest)
    - [LFS Artifact Config](#lfs-artifact-config)
    - [LFS Artifact Layers](#lfs-artifact-layers)

## Notational Conventions

As done by the [OCI image-spec](https://github.com/opencontainers/image-spec/blob/main/spec.md), this specification defines the following notational conventions:

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "NOT RECOMMENDED", "MAY", and "OPTIONAL" are to be interpreted as described in [RFC 2119](https://tools.ietf.org/html/rfc2119) (Bradner, S., "Key words for use in RFCs to Indicate Requirement Levels", BCP 14, RFC 2119, March 1997).

An implementation is not compliant if it fails to satisfy one or more of the MUST, MUST NOT, REQUIRED, SHALL, or SHALL NOT requirements for the protocols it implements.
An implementation is compliant if it satisfies all the MUST, MUST NOT, REQUIRED, SHALL, and SHALL NOT requirements for the protocols it implements.

## Overview

Packaging a Git repository as an OCI image artifact uses one or more Git [packfiles](https://git-scm.com/docs/pack-format) as OCI layers, with Git references stored in a custom OCI config.

## Specification

### OCI Manifest

The manifest specificification follows the [Guidelines for Artifact Usage](https://github.com/opencontainers/image-spec/blob/main/manifest.md#guidelines-for-artifact-usage), specifically decision tree number 3.

A Git OCI artifact manifest:

- MUST set `mediaType` to `application/vnd.oci.image.manifest.v1+json`.
- MUST set `artifactType` to `application/vnd.ai.act3.git.repo.v1+json`.
- MUST set `config.mediaType` to `application/vnd.ai.act3.git.config.v1+json`.
- MUST contain one or more layers with `mediaType` set to `application/vnd.ai.act3.git.pack.v1`.
  - Layers MUST contain a Git [packfile](https://git-scm.com/docs/pack-format).
  - The first layer MUST be a fully qualified packfile.
  - Any additional layers SHOULD be [thin packfiles](https://git-scm.com/docs/git-pack-objects#Documentation/git-pack-objects.txt---thin).
    - If so, these packfiles MUST contain a complete Git tree for layer ranges `[0:n]`, i.e. no dangling leaves.

Git OCI artifact manifest annotations MAY be used as desired.

#### Example OCI Manifest

```json
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "artifactType": "application/vnd.ai.act3.git.repo.v1+json",
  "config": {
    "mediaType": "application/vnd.ai.act3.git.config.v1+json",
    "digest": "sha256:28262ebd6a50230a95ddfbe9b55172121c721f214887107c6052355a6e6da5a9",
    "size": 1168
  },
  "layers": [
    {
      "mediaType": "application/vnd.ai.act3.git.pack.v1",
      "digest": "sha256:297b82b44c1c86e088cc95a68fd1d525878e4f430e48053ab6074e7cfe5c6d83",
      "size": 3191893,
      "annotations": {
        "org.opencontainers.image.title": "pack-53e257abe0aeabfb40bae4bb7e37b4466f988481.pack"
      }
    },
    {
      "mediaType": "application/vnd.ai.act3.git.pack.v1",
      "digest": "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
      "size": 641,
      "annotations": {
        "org.opencontainers.image.title": "pack-nfe347abe0aeabd40bagh4bb7e43b4466f988481.pack"
      }
    }
  ],
  "annotations": {
    "org.opencontainers.image.created": "1970-01-01T00:00:00Z",
    "vnd.ai.act3.git-remote-oci.version": "v0.0.0-alpha"
  }
}
```

### OCI Config

A Git OCI artifact config:

- MUST be identified by the `mediaType` `application/vnd.ai.act3.git.config.v1+json`.
- MUST use the [defined config format](#config-format).
- MUST contain at least one head reference to the default branch.
- MAY contain zero or more tag references.

#### Config Format

The format of a Git OCI artifact config is a JSON object with two maps containing head and tag references

- Config Object
  - `heads` : map of branch names to objects containing the referenced commit and the OCI manifest packfile layer containing the latest updates for the reference.
  - `tags` : map of tag names to objects containing the referenced commit and the OCI manifest packfile layer containing the latest updates for the reference.

Additional reference types, such as notes, may be added at a later date.

#### Example OCI Config

This config corresponds with the example manifest above. All references were included in the initial packfile, with only the `main` branch updated in the second "thin" packfile layer.

```json
{
  "heads": {
    "refs/heads/command-fetch": {
      "commit": "56db9a50f127dae1a0da3563cb205a45ee077208",
      "layer": "sha256:297b82b44c1c86e088cc95a68fd1d525878e4f430e48053ab6074e7cfe5c6d83"
    },
    "refs/heads/command-list": {
      "commit": "8741ea9910f853de6ea90709e2195a0558b5f3ce",
      "layer": "sha256:297b82b44c1c86e088cc95a68fd1d525878e4f430e48053ab6074e7cfe5c6d83"
    },
    "refs/heads/command-push": {
      "commit": "fa789ae0105a816244715e76f692cef04a6701a6",
      "layer": "sha256:297b82b44c1c86e088cc95a68fd1d525878e4f430e48053ab6074e7cfe5c6d83"
    },
    "refs/heads/design-docs": {
      "commit": "21023d360200012cefcd8f077b3b24aea7cb20f2",
      "layer": "sha256:297b82b44c1c86e088cc95a68fd1d525878e4f430e48053ab6074e7cfe5c6d83"
    },
    "refs/heads/main": {
      "commit": "44ad7a87cd3726cb4c270a4b1ea3fca5185abafc",
      "layer": "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
    },
    "refs/heads/setup-project": {
      "commit": "ced7d65b2dbdd91b9231b8edab51bac53415de2f",
      "layer": "sha256:297b82b44c1c86e088cc95a68fd1d525878e4f430e48053ab6074e7cfe5c6d83"
    }
  },
  "tags": {
    "refs/tags/foobar": {
      "commit": "21023d360200012cefcd8f077b3b24aea7cb20f2",
      "layer": "sha256:297b82b44c1c86e088cc95a68fd1d525878e4f430e48053ab6074e7cfe5c6d83"
    }
  }
}
```

### OCI Layer

A Git OCI artifact layer:

- MUST be identified by the `mediaType` `application/vnd.ai.act3.git.pack.v1`,
- MUST contain a Git [packfile](https://git-scm.com/docs/pack-format).
- The first layer MUST be a fully qualified, self-contained, Git [packfile](https://git-scm.com/docs/pack-format).
  - Any additional layers SHOULD be [thin packfiles](https://git-scm.com/docs/git-pack-objects#Documentation/git-pack-objects.txt---thin).
    - Thin packfiles MUST contain a complete Git tree for layer ranges `[0:n]`, i.e. no dangling leaves.
    - Thin packfiles SHOULD not contain duplicate data among themselves.

### LFS OCI Artifact Manifest

The specification uses the OCI [referrers API](https://github.com/opencontainers/distribution-spec/blob/main/spec.md#listing-referrers) for managing `git-lfs` tracked files. As such, if a local repository has `git-lfs` configured the [Git OCI manifest](#oci-manifest) descriptor is added as a `subject` in the LFS artifact manifest.

A LFS OCI artifact manifest:

- MUST set `mediaType` to `application/vnd.oci.image.manifest.v1+json`.
- MUST set `artifactType` to `application/vnd.ai.act3.git-lfs.repo.v1+json`.
- MUST set `config.mediaType` to `application/vnd.oci.empty.v1+json`.
  - whose digest is `sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a`
  - with size `2`
  - and data `"e30="` (an empty JSON struct `{}`)
- MUST contain one or more layers with `mediaType` set to `application/vnd.ai.act3.git-lfs.object.v1`
- MUST contain a `subject` OCI descriptor that is equal to the [Git OCI Artifact Manifest](#oci-manifest) tagged descriptor.

#### Example LFS OCI Manifest

```json
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "artifactType": "application/vnd.ai.act3.git-lfs.repo.v1+json",
  "config": {
    "mediaType": "application/vnd.oci.empty.v1+json",
    "digest": "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
    "size": 2,
    "data": "e30="
  },
  "layers": [
    {
      "mediaType": "application/vnd.ai.act3.git-lfs.object.v1",
      "digest": "sha256:189aaa97e4312694a44aaf5588827366b3e35869be1b55dde4bef19e0a7d23c8",
      "size": 344646,
      "annotations": {
        "org.opencontainers.image.title": "189aaa97e4312694a44aaf5588827366b3e35869be1b55dde4bef19e0a7d23c8"
      }
    },
    {
      "mediaType": "application/vnd.ai.act3.git-lfs.object.v1",
      "digest": "sha256:e5d4a2a85dc82a1ce91408e0263457e323c51b8a0c8771f11087b5f4bb47006f",
      "size": 197833,
      "annotations": {
        "org.opencontainers.image.title": "e5d4a2a85dc82a1ce91408e0263457e323c51b8a0c8771f11087b5f4bb47006f"
      }
    },
  ],
  "subject": {
    "mediaType": "application/vnd.oci.image.manifest.v1+json",
    "digest": "sha256:cdcfe254d7349f040393ec327d8bdea13c75b12e7f8dce1b72c9a694e082bfe5",
    "size": 637,
    "annotations": {
      "org.opencontainers.image.created": "1970-01-01T00:00:00Z",
      "vnd.ai.act3.git-remote-oci.version": "v0.0.0-alpha"
    },
    "artifactType": "application/vnd.ai.act3.git.repo.v1+json"
  },
  "annotations": {
    "org.opencontainers.image.created": "1970-01-01T00:00:00Z",
    "vnd.ai.act3.git-remote-oci.version": "v0.0.0-alpha"
  }
}
```

### LFS Artifact Config

A LFS OCI artifact config:

- MUST be identified by the `mediaType` `application/vnd.oci.empty.v1+json`.
- MUST be an empty JSON object `{}`.

The specification does not require a configuration, instead the `config` descriptor is set to the empty descriptor.

### LFS Artifact Layers

A LFS OCI artifact layer:

- MUST be identified by the `mediaType` `application/vnd.ai.act3.git-lfs.object.v1`.
- MUST contain the contents of a `git-lfs` tracked file (not a pointer file).
