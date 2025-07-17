package oci

import (
	"github.com/opencontainers/go-digest"
)

// Git OCI artifacts.
const (
	// ArtifactTypeGitManifest is the artifact type for an Git manifest.
	ArtifactTypeGitManifest = "application/vnd.ai.act3.git.repo.v1+json"

	// MediaTypeGitConfig is the media type for a Git config.
	MediaTypeGitConfig = "application/vnd.ai.act3.git.config.v1+json"

	// MediaTypePackLayer is the media type for a Git packfile stored as an OCI layer.
	MediaTypePackLayer = "application/vnd.ai.act3.git.pack.v1"

	// AnnotationGitRemoteOCIVersion is the key for the annotation to denote the git-remote-oci version used during the most recent operation.
	AnnotationGitRemoteOCIVersion = "vnd.ai.act3.git-remote-oci.version"
)

// ConfigGit is an OCI manifest config, containing information about a Git repository's references.
type ConfigGit struct {
	// Heads map Git head references to commit OID and layer digest pairs.
	Heads map[string]ReferenceInfo `json:"heads"`

	// Tags map Git tag references to commit OID and layer digest pairs.
	Tags map[string]ReferenceInfo `json:"tags"`
}

// ReferenceInfo holds informations about Git references stored in bundle layers.
type ReferenceInfo struct {
	// Commit pointed to by a reference
	Commit string `json:"commit"`

	// OCI layer, the packfile containing Commit
	Layer digest.Digest `json:"layer"`
}

// LFS OCI artifacts.
// const (
// 	// ArtifactTypeLFSManifest is the artifact type for an Git LFS manifest.
// 	ArtifactTypeLFSManifest = "application/vnd.ai.act3.git-lfs.repo.v1+json"

// 	// MediaTypeLFSConfig is the media type for a Git LFS config.
// 	MediaTypeLFSConfig = "application/vnd.ai.act3.git-lfs.config.v1+json"

// 	// MediaTypeLFSLayer is the media type used for Git LFS layers.
// 	MediaTypeLFSLayer = "application/vnd.ai.act3.git-lfs.object.v1"
// )

// ConfigLFS is an OCI manifest config, containing information about which commits an LFS file is associated with.
// type ConfigLFS struct {
// 	// Refs map Git references to layers containing LFS files.
// 	Refs map[string][]string `json:"refs"`
// }
