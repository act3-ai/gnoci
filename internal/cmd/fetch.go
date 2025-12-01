package cmd

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-git/go-git/v5/plumbing/format/packfile"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/opencontainers/go-digest"

	"github.com/act3-ai/gnoci/internal/git"
	"github.com/act3-ai/gnoci/internal/model"
	"github.com/act3-ai/gnoci/pkg/protocol/git/comms"
)

// HandleFetch executes a batch of fetch commands.
func HandleFetch(ctx context.Context, local git.Repository, remote model.ReadOnlyModeler, comm comms.Communicator) error {
	_, err := remote.Fetch(ctx)
	if err != nil {
		return fmt.Errorf("fetching remote metadata: %w", err)
	}

	reqs, err := comm.ParseFetchRequestBatch()
	if err != nil {
		return fmt.Errorf("parsing fetch request batch: %w", err)
	}

	packLayers := make(map[digest.Digest]struct{}, 1)
	for _, req := range reqs {
		_, layer, err := remote.ResolveRef(ctx, req.Ref.Name())
		if err != nil {
			return fmt.Errorf("resolving remote reference OCI layer: %w", err)
		}
		packLayers[layer] = struct{}{}
	}

	// resolve digests into full descriptors
	// TODO: we should parallelize this, but we don't yet know how safe this
	// is to do concurrently, consider locking the packfile.UpdateObjectStorage.
	for dgst := range packLayers {
		rc, err := remote.FetchLayer(ctx, dgst)
		if err != nil {
			return fmt.Errorf("fetching packfile layer: %w", err)
		}
		defer rc.Close() //nolint:revive

		// TODO: we may want to use packfile.WritePackfileToObjectStorage directly
		// what's the difference here? writing the objects themselves rather than the
		// entire packfile? If the objects already exist in another packfile will they be duplicated?
		st, ok := local.Storer().(storer.Storer)
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

	if err := comm.WriteFetchResponse(); err != nil {
		return fmt.Errorf("writing fetch response: %w", err)
	}

	return nil
}
