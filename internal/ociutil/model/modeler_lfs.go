package model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sourcegraph/conc/pool"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry"

	"github.com/act3-ai/gnoci/pkg/oci"
)

// LFSModeler extends [Modeler] with LFS support.
type LFSModeler interface {
	Modeler

	// FetchLFS pulls git-lfs OCI metadata from a remote. It does not pull layers.
	FetchLFS(ctx context.Context) error
	FetchLFSOrDefault(ctx context.Context) error
	// PushLFS upload the git-lfs OCI data model in it's current state.
	PushLFS(ctx context.Context) (ocispec.Descriptor, error)
	// AddLFSFile adds a git-lfs file as a layer to the git-lfs OCI data model.
	AddLFSFile(ctx context.Context, path string) (ocispec.Descriptor, error)
	// FetchLFSLayer fetches an LFS file from a layer in the git-lfs OCI data model.
	FetchLFSLayer(ctx context.Context, dgst digest.Digest) (io.ReadCloser, error)
}

// NewLFSModeler initializes a new git-lfs modeler.
func NewLFSModeler(ociRemote string, fstore *file.Store, gt oras.GraphTarget) LFSModeler {
	return &model{
		ociRemote: ociRemote,
		gt:        gt,
		fstore:    fstore,
	}
}

// ErrLFSManifestNotFound indicates an LFS manifest was not found.
var ErrLFSManifestNotFound = fmt.Errorf("LFS manifest not found")

func (m *model) FetchLFS(ctx context.Context) error {
	slog.DebugContext(ctx, "resolving git manifest referrers", slog.String("subjectDigest", m.manDesc.Digest.String()))
	referrers, err := registry.Referrers(ctx, m.gt, m.manDesc, "")
	slog.DebugContext(ctx, "found git manifest referrers", slog.String("referrers", fmt.Sprintf("%v", referrers)))

	// we expect one LFS manifest referrer
	switch {
	case len(referrers) < 1:
		return ErrLFSManifestNotFound
	case len(referrers) > 1:
		return fmt.Errorf("expected 1 LFS referrer, got %d", len(referrers)) // should never hit
	case err != nil:
		return fmt.Errorf("resolving commit manifest predecessors: %w", err)
	}
	m.lfsManDesc = referrers[0]

	manRaw, err := content.FetchAll(ctx, m.gt, m.lfsManDesc)
	if err != nil {
		return fmt.Errorf("fetching LFS manifest: %w", err)
	}

	if err := json.Unmarshal(manRaw, &m.lfsMan); err != nil {
		return fmt.Errorf("decoding LFS manifest: %w", err)
	}

	return nil
}

func (m *model) FetchLFSLayer(ctx context.Context, dgst digest.Digest) (io.ReadCloser, error) {
	slog.DebugContext(ctx, "fetching LFS file", slog.String("digest", dgst.String()))
	// TODO: reverse iter? it is more likely we'll want to fetch newer layers
	for _, desc := range m.lfsMan.Layers {
		if desc.Digest == dgst {
			rc, err := m.gt.Fetch(ctx, desc)
			if err != nil {
				return nil, fmt.Errorf("fetching layer: %w", err)
			}
			return rc, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", errLayerNotInManifest, dgst.String())
}

func (m *model) FetchLFSOrDefault(ctx context.Context) error {
	err := m.FetchLFS(ctx)
	switch {
	case errors.Is(err, ErrLFSManifestNotFound):
		slog.InfoContext(ctx, "remote does not exist, initializing default lfs manifest")
		m.lfsMan = ocispec.Manifest{}
		return nil
	case err != nil:
		return fmt.Errorf("fetching remote metadata: %w", err)
	default:
		return nil
	}
}

func (m *model) PushLFS(ctx context.Context) (ocispec.Descriptor, error) {
	slog.DebugContext(ctx, "pushing LFS data model")

	// if len(m.newLFS) < 1 {
	// 	return *m.lfsMan.Subject, nil
	// }

	// TODO: plumb concurrency here
	p := pool.New().WithErrors().WithContext(ctx)
	for _, desc := range m.newLFS {
		slog.DebugContext(ctx, "pushing LFS file", "digest", desc.Digest.String())
		p.Go(func(ctx context.Context) error {
			rc, err := m.fstore.Fetch(ctx, desc)
			if err != nil {
				return fmt.Errorf("fetching LFS file from temporary filestore: %w", err)
			}
			defer rc.Close()

			if err := m.gt.Push(ctx, desc, rc); err != nil {
				return fmt.Errorf("pushing LFS file: %w", err)
			}

			return nil
		})

	}
	if err := p.Wait(); err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("pushing LFS files: %w", err)
	}

	// if deleteOld {
	// 	// TODO: improve error handling
	// 	r, ok := m.gt.(*remote.Repository)
	// 	if !ok {
	// 		return ocispec.Descriptor{}, fmt.Errorf("graph target is not a remote repository")
	// 	}

	// 	// remove old referrer, in case this is an LFS only push
	// 	if err := r.Delete(ctx, m.lfsManDesc); err != nil && !errors.Is(err, errdef.ErrNotFound) {
	// 		return ocispec.Descriptor{}, fmt.Errorf("deleting old LFS referrer manifest: %w", err)
	// 	}
	// }

	// if m.manDesc.Digest == "" {
	// 	slog.DebugContext(ctx, "pushing temporary git manifest")
	// 	// first push, we'll push a temporary unique manifest
	// 	// TODO: is there a more reliable way to generate a unique manifest?
	// 	m.cfg.Heads[tempGitManifest] = oci.ReferenceInfo{Commit: time.Now().String()}

	// 	var err error
	// 	m.manDesc, err = m.Push(ctx)
	// 	if err != nil {
	// 		return ocispec.Descriptor{}, fmt.Errorf("pushing temporary git manifest: %w", err)
	// 	}
	// }

	slog.DebugContext(ctx, "pushing LFS manifest", slog.String("subjectDigest", m.manDesc.Digest.String()))
	manOpts := oras.PackManifestOptions{
		Subject:             &m.manDesc,
		Layers:              m.lfsMan.Layers,
		ConfigDescriptor:    nil,                                                                  // oras handles for us
		ManifestAnnotations: map[string]string{ocispec.AnnotationCreated: "1970-01-01T00:00:00Z"}, // POSIX epoch
		// TODO: add user agent/version to annotations?
	}

	lfsManDesc, err := oras.PackManifest(ctx, m.gt, oras.PackManifestVersion1_1, oci.ArtifactTypeLFSManifest, manOpts)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("packing and pushing LFS manifest: %w", err)
	}
	slog.DebugContext(ctx, "pushed LFS manifest", slog.String("digest", lfsManDesc.Digest.String()))

	return lfsManDesc, nil
}

func (m *model) AddLFSFile(ctx context.Context, path string) (ocispec.Descriptor, error) {
	slog.DebugContext(ctx, "adding LFS file", slog.String("oid", filepath.Base(path)))

	// dedup
	for _, desc := range m.lfsMan.Layers {
		// HACK: this check smells bad
		if desc.Digest.Encoded() == filepath.Base(path) {
			return desc, nil
		}
	}

	desc, err := m.fstore.Add(ctx, filepath.Base(path), oci.MediaTypeLFSLayer, path)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("adding LFS file to intermediate fstore: %w", err)
	}

	// TODO: safe to assume the LFS manifest has already been pulled?
	// TODO: this is a bug, we need to ensure deduplication of lfs files
	m.lfsMan.Layers = append(m.lfsMan.Layers, desc)
	m.newLFS = append(m.newLFS, desc)
	return desc, nil
}
