// Package model provides utility functions for modeling a git repository in OCI.
package model

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/act3-ai/gitoci/internal/ociutil"
	"github.com/act3-ai/gitoci/pkg/oci"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"
)

// Modeler allows for fetching, updating, and pushing a Git OCI data model to
// an OCI registry.
// TODO: Is this interface overloaded?
type Modeler interface {
	// Cleanup closes and removes any files opened during modeling or transfers.
	Cleanup() error
	// Fetch pulls Git OCI metadata from a remote. It does not pull layers.
	Fetch(ctx context.Context, ref string) error
	// Push uploads the Git OCI data model in its current state.
	Push(ctx context.Context, ref string) (ocispec.Descriptor, error)
	// AddPack adds a packfile as a layer to the Git OCI data model. refs are the
	// updated remote references.
	AddPack(ctx context.Context, path string, refs ...plumbing.Reference) (ocispec.Descriptor, error)
	// UpdateRef updates a Git reference and the object it points to in the
	// Git OCI data model. Useful for updating a reference where its object
	// is within an existing packfile.
	UpdateRef(ctx context.Context, gitRef plumbing.Reference, ociLayer digest.Digest)

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
	slog.DebugContext(ctx, "adding packfile to Git OCI manifest", "path")
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
