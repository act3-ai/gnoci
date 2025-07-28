// Package model provides utility functions for modeling a git repository in OCI.
package model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/act3-ai/gitoci/internal/ociutil"
	"github.com/act3-ai/gitoci/pkg/oci"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/errdef"
)

// Modeler allows for fetching, updating, and pushing a Git OCI data model to
// an OCI registry.
// TODO: Is this interface overloaded?
type Modeler interface {
	// Cleanup closes and removes any files opened during modeling or transfers.
	Cleanup() error
	// Fetch pulls Git OCI metadata from a remote. It does not pull layers.
	Fetch(ctx context.Context, ref string) error
	// FetchOrDefault extends Fetch to initialize an empty OCI manifest and config
	// if the remote ref does not exist.
	FetchOrDefault(ctx context.Context, ref string) error
	// Push uploads the Git OCI data model in its current state.
	Push(ctx context.Context, ref string) (ocispec.Descriptor, error)
	// AddPack adds a packfile as a layer to the Git OCI data model. refs are the
	// updated remote references.
	AddPack(ctx context.Context, path string, refs ...plumbing.Reference) (ocispec.Descriptor, error)
	// UpdateRef updates a Git reference and the object it points to in the
	// Git OCI data model. Useful for updating a reference where its object
	// is within an existing packfile.
	UpdateRef(ctx context.Context, gitRef plumbing.Reference, ociLayer digest.Digest)
	// ResolveRef resolves the commit hash a remote reference refers to. Returns nil, nil if
	// the ref does not exist or if not supported (head or tag ref).
	ResolveRef(ctx context.Context, ref plumbing.ReferenceName) (*plumbing.Reference, error)
	// DeleteRef removes a reference from the remote. The commit remains.
	DeleteRef(ctx context.Context, ref plumbing.ReferenceName)
	// CommitExists uses a local repository to resolve the best known OCI layer containing the
	CommitExists(localRepo *git.Repository, commit *object.Commit) (digest.Digest, error)

	// TODO: LFS Support
	// AddLFSFile(path string) (ocispec.Descriptor, error)
}

// NewModeler initializes a new modeler. It is the caller's responsibility to
// Cleanup().
func NewModeler(gt oras.GraphTarget) (Modeler, error) {
	fstorePath, err := os.MkdirTemp(os.TempDir(), "GitOCI-fstore-*")
	if err != nil {
		return nil, fmt.Errorf("creating temporary directory for intermediate git repository: %w", err)
	}

	fstore, err := file.New(fstorePath)
	if err != nil {
		return nil, fmt.Errorf("initializing shared filestore: %w", err)
	}

	return &model{
		gt:         gt,
		fstore:     fstore,
		fstorePath: fstorePath,
	}, nil
}

// model implements Modeler.
//
// Note: updates to Git OCI metadata are not concurrency safe.
type model struct {
	gt         oras.GraphTarget
	fstore     *file.Store
	fstorePath string

	man      ocispec.Manifest
	cfg      oci.ConfigGit
	newPacks []ocispec.Descriptor
}

func (m *model) Cleanup() error {
	if err := m.fstore.Close(); err != nil {
		return fmt.Errorf("closing intermediate OCI file store: %w", err)
	}

	if err := os.RemoveAll(m.fstorePath); err != nil {
		return fmt.Errorf("removing intermediate OCI file store, path = %s: %w", m.fstorePath, err)
	}

	return nil
}

func (m *model) Fetch(ctx context.Context, ref string) error {
	gt, err := ociutil.NewGraphTarget(ctx, ref)
	if err != nil {
		return err
	}

	slog.DebugContext(ctx, "resolving manifest descriptor")
	manDesc, err := gt.Resolve(ctx, ref)
	if err != nil {
		return fmt.Errorf("resolving manifest descriptor: %w", err)
	}

	slog.DebugContext(ctx, "fetching manifest")
	manRaw, err := content.FetchAll(ctx, gt, manDesc)
	if err != nil {
		return fmt.Errorf("fetching manifest: %w", err)
	}

	if err := json.Unmarshal(manRaw, &m.man); err != nil {
		return fmt.Errorf("decoding manifest: %w", err)
	}

	slog.DebugContext(ctx, "fetching config")
	cfgRaw, err := content.FetchAll(ctx, gt, m.man.Config)
	if err != nil {
		return fmt.Errorf("fetching config: %w", err)
	}

	if err := json.Unmarshal(cfgRaw, &m.cfg); err != nil {
		return fmt.Errorf("decoding config: %w", err)
	}

	return nil
}

func (m *model) FetchOrDefault(ctx context.Context, ref string) error {
	err := m.Fetch(ctx, ref)
	switch {
	case errors.Is(err, errdef.ErrNotFound):
		slog.InfoContext(ctx, "remote does not exist, initializing default manifest and config")
		m.cfg = oci.ConfigGit{
			Heads: make(map[string]oci.ReferenceInfo, 0),
			Tags:  make(map[string]oci.ReferenceInfo, 0),
		}
		m.man = ocispec.Manifest{
			MediaType:    ocispec.MediaTypeImageManifest,
			ArtifactType: oci.ArtifactTypeGitManifest,
			// annotations set on push
		}
		return nil
	case err != nil:
		return fmt.Errorf("fetching remote metadata: %w", err)
	default:
		return nil
	}
}

func (m *model) Push(ctx context.Context, ref string) (ocispec.Descriptor, error) {
	// TODO: Perhaps we could make this more efficient, ONLY in the case where
	// multiple packfiles are added, if we make a custom oras.CopyGraphOptions to
	// skip existing packfiles - but that may be tricky as I believe the HEAD
	// comes before the copy

	for _, desc := range m.newPacks {
		slog.DebugContext(ctx, "pushing packfile", "digest", desc.Digest.String())
		if err := oras.CopyGraph(ctx, m.fstore, m.gt, desc, oras.DefaultCopyGraphOptions); err != nil {
			return ocispec.Descriptor{}, fmt.Errorf("copying packfile layer to target repository: %w", err)
		}
	}

	cfgRaw, err := json.Marshal(m.cfg)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("encoding base manifest config")
	}
	slog.DebugContext(ctx, "Pushing base config")
	cfgDesc, err := oras.PushBytes(ctx, m.gt, oci.MediaTypeGitConfig, cfgRaw)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("pushing base config to repository: %w", err)
	}

	slog.DebugContext(ctx, "Pushing base manifest")
	manOpts := oras.PackManifestOptions{
		Layers:              m.man.Layers, // if a new bundle was made, it was already added to the manifest
		ConfigDescriptor:    &cfgDesc,
		ManifestAnnotations: map[string]string{ocispec.AnnotationCreated: "1970-01-01T00:00:00Z"}, // POSIX epoch
		// TODO: add user agent/version to annotations?
	}

	manDesc, err := oras.PackManifest(ctx, m.gt, oras.PackManifestVersion1_1, oci.ArtifactTypeGitManifest, manOpts)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("packing and pushing base manifest: %w", err)
	}

	return manDesc, nil
}

func (m *model) AddPack(ctx context.Context, path string, refs ...plumbing.Reference) (ocispec.Descriptor, error) {
	slog.DebugContext(ctx, "adding packfile to Git OCI manifest", "path", path)
	// filepath.Base adds an annotation for the filename, without exposing a user's filesystem
	desc, err := m.fstore.Add(ctx, filepath.Base(path), oci.MediaTypePackLayer, path)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("adding packfile to intermediate file store: %w", err)
	}
	m.man.Layers = append(m.man.Layers, desc)

	for _, ref := range refs {
		m.UpdateRef(ctx, ref, desc.Digest)
	}

	return desc, nil
}

func (m *model) UpdateRef(ctx context.Context, ref plumbing.Reference, ociLayer digest.Digest) {
	switch {
	case ref.Name().IsBranch():
		m.cfg.Heads[ref.String()] = oci.ReferenceInfo{Commit: ref.Hash().String(), Layer: ociLayer}
	case ref.Name().IsTag():
		m.cfg.Tags[ref.String()] = oci.ReferenceInfo{Commit: ref.Hash().String(), Layer: ociLayer}
	default:
		slog.WarnContext(ctx, "skipping unknown remote reference type", "reference", ref.String())
	}
}

func (m *model) ResolveRef(ctx context.Context, ref plumbing.ReferenceName) (*plumbing.Reference, error) {
	var ok bool
	var rInfo oci.ReferenceInfo
	switch {
	case ref.IsBranch():
		rInfo, ok = m.cfg.Heads[ref.String()]
	case ref.IsTag():
		rInfo, ok = m.cfg.Tags[ref.String()]
	default:
		slog.WarnContext(ctx, "skipping resolution unknown remote reference type", "reference", ref.String())
		return nil, nil
	}

	if ok {
		return plumbing.NewHashReference(ref, plumbing.NewHash(rInfo.Commit)), nil
	}
	return nil, nil
}

func (m *model) DeleteRef(ctx context.Context, ref plumbing.ReferenceName) {
	slog.InfoContext(ctx, "deleting reference from remote", "ref", ref.String())

	switch {
	case ref.IsBranch():
		delete(m.cfg.Heads, ref.String())
	case ref.IsTag():
		delete(m.cfg.Tags, ref.String())
	default:
		slog.WarnContext(ctx, "skipping deletion unknown remote reference type", "reference", ref.String())
		return
	}
}

func (m *model) CommitExists(localRepo *git.Repository, commit *object.Commit) (digest.Digest, error) {
	// TODO: rebuilding this map each time is inefficient
	resolver := m.sortRefsByLayer()

	// most efficient with a relatively new base layer containing few refs
	// TODO: room for optimization?
	for _, layer := range m.man.Layers {
		for _, c := range resolver[layer.Digest] {
			existingCommit, err := localRepo.CommitObject(c)
			if err != nil {
				return "", fmt.Errorf("resolving commit object for remote head commit %s: %w", c.String(), err)
			}

			isAncestor, err := commit.IsAncestor(existingCommit)
			if err != nil {
				return "", fmt.Errorf("resolving ancestral status of commit to remote head commit %s: %w", c.String(), err)
			}
			if isAncestor {
				return layer.Digest, nil
			}
		}
	}

	return "", nil
}

// sortRefsByLayer organizes the refs in the current config by layer,
// returning a map of layer digests to a slice of commit hashes contained in that layer.
func (m *model) sortRefsByLayer() map[digest.Digest][]plumbing.Hash {
	layerResolver := make(map[digest.Digest][]plumbing.Hash) // layer digest : []commits
	for _, info := range m.cfg.Heads {
		layerResolver[info.Layer] = append(layerResolver[info.Layer], plumbing.NewHash(info.Commit))
	}
	for _, info := range m.cfg.Tags {
		layerResolver[info.Layer] = append(layerResolver[info.Layer], plumbing.NewHash(info.Commit))
	}

	return layerResolver
}
