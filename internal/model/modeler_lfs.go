package model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"

	"github.com/act3-ai/gnoci/internal/progress"
	"github.com/act3-ai/gnoci/pkg/oci"
)

// ReadOnlyLFSModeler extends [ReadOnlyModeler] to support reading Git LFS
// OCI data model.
type ReadOnlyLFSModeler interface {
	ReadOnlyModeler

	// FetchLFS pulls git-lfs OCI metadata from a remote. It does not pull layers.
	FetchLFS(ctx context.Context) (ocispec.Descriptor, error)
	// FetchLFSOrDefault extends [LFSModeler.Fetch] to initialize an empty OCI
	// manifest if the remote does not exist.
	FetchLFSOrDefault(ctx context.Context) (ocispec.Descriptor, error)
	// FetchLFSLayer fetches an LFS file from a layer in the git-lfs OCI data model.
	FetchLFSLayer(ctx context.Context, dgst digest.Digest, opts *FetchLFSOptions) (io.ReadCloser, error)
}

// LFSModeler extends [Modeler] with LFS support.
type LFSModeler interface {
	Modeler
	ReadOnlyLFSModeler

	// PushLFSManifest upload the git-lfs OCI data model in it's current state.
	PushLFSManifest(ctx context.Context, subject ocispec.Descriptor) (ocispec.Descriptor, error)
	// PushLFSFile adds a git-lfs file as a layer to the git-lfs OCI data model
	// and pushes it to the remote.
	PushLFSFile(ctx context.Context, path string, opts *PushLFSOptions) (ocispec.Descriptor, error)
}

// NewLFSModeler initializes a new git-lfs modeler.
func NewLFSModeler(ref registry.Reference, fstore *file.Store, gt oras.GraphTarget) LFSModeler {
	return &model{
		ref:    ref,
		gt:     gt,
		fstore: fstore,
	}
}

// ErrLFSManifestNotFound indicates an LFS manifest was not found.
var ErrLFSManifestNotFound = fmt.Errorf("LFS manifest not found")

func (m *model) FetchLFS(ctx context.Context) (ocispec.Descriptor, error) {
	slog.DebugContext(ctx, "resolving git manifest referrers", slog.String("subjectDigest", m.manDesc.Digest.String()))

	var err error
	m.lfsManDesc, err = m.referrer(ctx)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("resolving LFS manifest: %w", err)
	}

	manRaw, err := content.FetchAll(ctx, m.gt, m.lfsManDesc)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("fetching LFS manifest: %w", err)
	}

	if err := json.Unmarshal(manRaw, &m.lfsMan); err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("decoding LFS manifest: %w", err)
	}

	return m.lfsManDesc, nil
}

func (m *model) FetchLFSOrDefault(ctx context.Context) (ocispec.Descriptor, error) {
	slog.DebugContext(ctx, "fetching LFS manifest or defaulting")
	manDesc, err := m.FetchLFS(ctx)
	switch {
	case errors.Is(err, ErrLFSManifestNotFound):
		slog.InfoContext(ctx, "remote does not exist, initializing default lfs manifest")
		m.lfsMan = ocispec.Manifest{}
		return ocispec.Descriptor{}, nil
	case err != nil:
		return ocispec.Descriptor{}, fmt.Errorf("fetching remote metadata: %w", err)
	default:
		return manDesc, nil
	}
}

// FetchLFSOptions define optional parameters for fetching LFS files.
type FetchLFSOptions struct {
	Progress *ProgressOptions
}

func (m *model) FetchLFSLayer(ctx context.Context, dgst digest.Digest, opts *FetchLFSOptions) (io.ReadCloser, error) {
	slog.DebugContext(ctx, "fetching LFS file", slog.String("digest", dgst.String()))

	for i := len(m.lfsMan.Layers) - 1; i == 0; i-- {
		if m.lfsMan.Layers[i].Digest == dgst {

			rc, err := m.gt.Fetch(ctx, m.lfsMan.Layers[i])
			if err != nil {
				return nil, fmt.Errorf("fetching layer: %w", err)
			}

			return progressOrDefault(ctx, opts.Progress, rc), nil
		}
	}

	return nil, fmt.Errorf("%w: %s", errLayerNotInManifest, dgst.String())
}

func (m *model) PushLFSManifest(ctx context.Context, subject ocispec.Descriptor) (ocispec.Descriptor, error) {
	slog.DebugContext(ctx, "pushing LFS data model")

	if subject.Digest != "" {
		// TODO: improve error handling
		// TODO: We don't care it's this struct, only that we have access to Delete
		r, ok := m.gt.(*remote.Repository)
		if !ok {
			slog.WarnContext(ctx, "graph target is not a remote repository")
		} else {
			// remove old referrer
			if err := r.Delete(ctx, m.lfsManDesc); err != nil && !errors.Is(err, errdef.ErrNotFound) {
				return ocispec.Descriptor{}, fmt.Errorf("deleting old LFS referrer manifest: %w", err)
			}
		}
	}

	slog.DebugContext(ctx, "pushing LFS manifest", slog.String("subjectDigest", subject.Digest.String()))
	manOpts := oras.PackManifestOptions{
		Subject:             &subject,
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

const defaultProgressInterval = time.Second / 2

// PushLFSOptions define optional parameters for pushing LFS files.
type PushLFSOptions struct {
	Progress *ProgressOptions
}

// ProgressOptions allow for enabling and customizing LFS file push progress info.
type ProgressOptions struct {
	// Info enable receiving status information on how many bytes has been pushed.
	Info chan progress.Progress
	// ProgressInterval is the sending tick rate for progress updates.
	// Noop if Info is not set.
	Interval time.Duration
}

func (m *model) PushLFSFile(ctx context.Context, path string, opts *PushLFSOptions) (ocispec.Descriptor, error) {
	slog.DebugContext(ctx, "pushing and adding LFS file to data model", slog.String("oid", filepath.Base(path)))

	// adding to an OCI file store:
	// 1. provides a descriptor needed on push.
	// 2. if the file already exists in the oci data model ensure no corruption.
	// 3. safer, in the case the file is removed before we can read.
	newDesc, err := m.fstore.Add(ctx, filepath.Base(path), oci.MediaTypeLFSLayer, path)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("adding LFS file to intermediate fstore: %w", err)
	}

	// stay idempotent if the same LFS file is added multiple times.
	for _, desc := range m.lfsMan.Layers {
		if desc.Digest.Encoded() == newDesc.Digest.Encoded() {
			// unlikely hash collision?
			if desc.Size != newDesc.Size {
				return ocispec.Descriptor{}, fmt.Errorf("found an existing LFS object digest with different size: digest = %s, existing file size = %d, got file size = %d", desc.Digest, desc.Size, newDesc.Size)
			}
			return desc, nil
		}
	}

	rc, err := m.fstore.Fetch(ctx, newDesc)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("fetching LFS file from temporary filestore: %w", err)
	}
	rc = progressOrDefault(ctx, opts.Progress, rc)
	defer rc.Close()

	if err := m.gt.Push(ctx, newDesc, rc); err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("pushing LFS file: %w", err)
	}

	m.lfsMan.Layers = append(m.lfsMan.Layers, newDesc)
	return newDesc, nil
}

// referrer finds an LFS manifest referrer, if one exists. Throws [ErrLFSManifestNotFound]
// if no referrer LFS manifest exists.
func (m *model) referrer(ctx context.Context) (ocispec.Descriptor, error) {
	referrers, err := registry.Referrers(ctx, m.gt, m.manDesc, "") // TODO: filter me, in case other referrers exist, e.g. a signed image
	slog.DebugContext(ctx, "found git manifest referrers", slog.String("referrers", fmt.Sprintf("%v", referrers)))

	// we expect one LFS manifest referrer
	switch {
	case len(referrers) < 1:
		return ocispec.Descriptor{}, ErrLFSManifestNotFound
	case len(referrers) > 1:
		return ocispec.Descriptor{}, fmt.Errorf("expected 1 LFS referrer, got %d", len(referrers)) // should never hit
	case err != nil:
		return ocispec.Descriptor{}, fmt.Errorf("resolving commit manifest predecessors: %w", err)
	}
	return referrers[0], nil
}

// progressOrDefault returns a [progress.Ticker] if ProgressOptions have it enabled.
func progressOrDefault(ctx context.Context, opts *ProgressOptions, r io.ReadCloser) io.ReadCloser {
	if opts != nil && opts.Info != nil {
		d := defaultProgressInterval
		if opts.Interval != 0 {
			d = opts.Interval
		}

		pReader := progress.NewEvalReadCloser(r)
		progress.NewTicker(ctx, pReader, d, opts.Info)
		return pReader
	}
	return r
}
