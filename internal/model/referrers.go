package model

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// ReferrerUpdater updates referring manifests to a new subject descriptor.
type ReferrerUpdater func(ctx context.Context, subject ocispec.Descriptor) error

// UpdateLFSReferrer updates the subject of an existing LFS referrer manifest
// to a new subject descriptor.
func UpdateLFSReferrer(m LFSModeler) ReferrerUpdater {
	return func(ctx context.Context, subject ocispec.Descriptor) error {
		// update LFS if it exists
		lfsManDesc, err := m.FetchLFS(ctx) // fetch LFS from old git descriptor
		switch {
		case errors.Is(err, ErrLFSManifestNotFound):
			slog.DebugContext(ctx, "LFS manifest not found")
		case err != nil:
			slog.ErrorContext(ctx, "failed to fetch LFS manifest", slog.String("error", err.Error()))
		default:
			_, err = m.PushLFSManifest(ctx, subject)
			if err != nil {
				return fmt.Errorf("pushing LFS manifest: %w", err)
			}
		}
		slog.DebugContext(ctx, "successfully updated LFS referrer subject", slog.String("subjectDigest", subject.Digest.String()), slog.String("referrerDigest", lfsManDesc.Digest.String()))
		return nil
	}
}
