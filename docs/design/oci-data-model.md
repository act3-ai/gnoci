# Git as OCI Artifact Data Model

The data model for storing Git repositories in OCI compliant registries follows the [OCI image-spec](https://github.com/opencontainers/image-spec/blob/main/spec.md). In particular, the data model is packaged as defined by the [image manifest spec guidelines for artifact usage](https://github.com/opencontainers/image-spec/blob/main/manifest.md#guidelines-for-artifact-usage).

## Artifact Manifest

The data model for storing a Git artifact defines a configuration and at least one layer. As such, the data model follows the [guidelines for artifact usage](https://github.com/opencontainers/image-spec/blob/main/manifest.md#guidelines-for-artifact-usage) decision tree number 3 by defining a manifest `artifactType`, a config `mediaType`, and a layer `mediaType`:

- Manifest `artifactType`
  - `application/vnd.ai.act3.git.repo.v1+json`
- Config `mediaType`
  - `application/vnd.ai.act3.git.config.v1+json`
- Layer `mediaType`
  - `application/vnd.ai.act3.git.pack.v1`

Additionally, two annotations are defined:

TODO: Currently packfiles are not reproducible, why? If this is not possible, is there benefit in setting `org.opencontainers.image.created` to the real value?

- `vnd.ai.act3.git-remote-oci.version`
  - The version of the `git-remote-oci` remote helper which created the artifact.
- `org.opencontainers.image.created`
  - Set to `1970-01-01T00:00:00Z`

### Example Manifest

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
    }
  ],
  "annotations": {
    "org.opencontainers.image.created": "1970-01-01T00:00:00Z",
    "vnd.ai.act3.git-remote-oci.version": "v0.0.0-alpha"
  }
}
```

## Artifact Config

The data model utilizes an OCI config for storing Git references. Alongside the commit a reference refers to, the config stores the OCI layer whose packfile contains the commit. The config defines two maps, containing head and tag references separately. Additional reference types, such as notes, may be added at a later date.

TODO: Is storing the layer containing the ref's commit sufficient? Could we store additional information, such as a merge-base, in the event that more than one packfile is needed to fast-forward local history to the commit?

### Example Config

```json
{
  "heads": {
    "refs/heads/act3-pt/blueprints/render-orphan": {
      "commit": "d2d51a405e5168f1945a02035c8f6089da53cb04",
      "layer": "sha256:ced0ce295f6ba489926bf5cffb27948701855f79b5556deadd01756516a27c53"
    },
    "refs/heads/command-fetch": {
      "commit": "56db9a50f127dae1a0da3563cb205a45ee077208",
      "layer": "sha256:ced0ce295f6ba489926bf5cffb27948701855f79b5556deadd01756516a27c53"
    },
    "refs/heads/command-list": {
      "commit": "8741ea9910f853de6ea90709e2195a0558b5f3ce",
      "layer": "sha256:ced0ce295f6ba489926bf5cffb27948701855f79b5556deadd01756516a27c53"
    },
    "refs/heads/command-push": {
      "commit": "fa789ae0105a816244715e76f692cef04a6701a6",
      "layer": "sha256:ced0ce295f6ba489926bf5cffb27948701855f79b5556deadd01756516a27c53"
    },
    "refs/heads/design-docs": {
      "commit": "21023d360200012cefcd8f077b3b24aea7cb20f2",
      "layer": "sha256:ced0ce295f6ba489926bf5cffb27948701855f79b5556deadd01756516a27c53"
    },
    "refs/heads/main": {
      "commit": "21023d360200012cefcd8f077b3b24aea7cb20f2",
      "layer": "sha256:ced0ce295f6ba489926bf5cffb27948701855f79b5556deadd01756516a27c53"
    },
    "refs/heads/setup-project": {
      "commit": "ced7d65b2dbdd91b9231b8edab51bac53415de2f",
      "layer": "sha256:ced0ce295f6ba489926bf5cffb27948701855f79b5556deadd01756516a27c53"
    }
  },
  "tags": {
    "refs/tags/foobar": {
      "commit": "21023d360200012cefcd8f077b3b24aea7cb20f2",
      "layer": "sha256:ced0ce295f6ba489926bf5cffb27948701855f79b5556deadd01756516a27c53"
    }
  }
}
```

## Artifact Creation and Updates

### Initial Artifact

When pushing the Git OCI artifact to a remote OCI registry for the first time, i.e. the [OCI digest or tag reference](https://github.com/opencontainers/distribution-spec/blob/main/spec.md#checking-if-content-exists-in-the-registry) does not exist, a [packfile](https://git-scm.com/docs/pack-format) is created containing all git objects reachable from the references pushed. In effect, the packfile contains the complete git history for each reference. This packfile serves as the base layer for subsequent updates. Additionally, each tag or head reference pushed is added to the artifact config alongside the digest of the packfile layer created.

### Subsequent Updates

If a Git OCI artifact reference already exists, `git-remote-oci` creates a single ["thin" pack](https://git-scm.com/docs/git-pack-objects#Documentation/git-pack-objects.txt---thin) containing reachable objects not included in any existing packfile layers.
