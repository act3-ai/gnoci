package cmd

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/act3-ai/gnoci/internal/ociutil/model"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/packfile"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/opencontainers/go-digest"
)

// HandleFetch executes a batch of fetch commands.
func HandleFetch(ctx context.Context, local *git.Repository, remote model.Modeler, remoteAddress string, cmds []Git, w Writer) error {
	if err := remote.Fetch(ctx); err != nil {
		return fmt.Errorf("fetching remote metadata: %w", err)
	}

	// fetchRefs := make([]*plumbing.Reference, 0, len(cmds))
	packLayers := make(map[digest.Digest]struct{}, 1)
	for _, c := range cmds {
		ref, err := parseFetch(c)
		if err != nil {
			return err
		}
		// fetchRefs = append(fetchRefs, ref)

		_, layer, err := remote.ResolveRef(ctx, plumbing.ReferenceName(ref.Name().String()))
		if err != nil {
			return fmt.Errorf("resolving remote reference OCI layer: %w", err)
		}
		packLayers[layer] = struct{}{}
	}

	// resolve digests into full descriptors
	// TODO: we should parallelize this, but we don't yet know how safe this
	// is to do concurrently, consider locking the packfile.UpdateObjectStorage.
	for dgst := range packLayers {
		slog.DebugContext(ctx, "fetching packfile", "layerDigest", dgst)
		rc, err := remote.FetchLayer(ctx, dgst)
		if err != nil {
			return fmt.Errorf("fetching packfile layer: %w", err)
		}
		defer rc.Close() //nolint:revive

		// TODO: we may want to use packfile.WritePackfileToObjectStorage directly
		// what's the difference here? writing the objects themselves rather than the
		// entire packfile? If the objects already exist in another packfile will they be duplicated?
		st, ok := local.Storer.(storer.Storer)
		if !ok {
			return fmt.Errorf("repository storer is not a storer.Storer")
		}
		slog.DebugContext(ctx, "updating object storage with packfile")
		if err := packfile.UpdateObjectStorage(st, rc); err != nil {
			return fmt.Errorf("updating object storage with packfile: %w", err)
		}
		if err := rc.Close(); err != nil {
			return fmt.Errorf("closing packfile reader: %w", err)
		}
	}
	slog.InfoContext(ctx, "done fetching packfiles")

	if err := w.Flush(true); err != nil {
		return fmt.Errorf("writing newline to git after fetch: %w", err)
	}

	// TODO: consider repacking repo objects

	return nil
}

// parseFetch parses a fetch command received from Git, returning it as a go-git reference.
func parseFetch(c Git) (*plumbing.Reference, error) {
	if len(c.Data) < 2 {
		return nil, fmt.Errorf("insufficient number of arguments in fetch command got %d, expected 2", len(c.Data))
	}
	hash := c.Data[0]
	name := c.Data[1]

	return plumbing.NewHashReference(plumbing.ReferenceName(name), plumbing.NewHash(hash)), nil
}
