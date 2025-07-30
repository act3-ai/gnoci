package actions

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/act3-ai/gitoci/internal/cmd"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/packfile"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/opencontainers/go-digest"
)

// TODO: look into the keep file lock stuff

func (action *GitOCI) fetch(ctx context.Context, cmds []cmd.Git) error {
	if err := action.remote.Fetch(ctx, action.addess); err != nil {
		return fmt.Errorf("fetching remote metadata: %w", err)
	}

	// fetchRefs := make([]*plumbing.Reference, 0, len(cmds))
	packLayers := make(map[digest.Digest]struct{}, 1)
	for _, cmd := range cmds {
		ref, err := parseFetch(cmd)
		if err != nil {
			return err
		}
		// fetchRefs = append(fetchRefs, ref)

		_, layer, err := action.remote.ResolveRef(ctx, plumbing.ReferenceName(ref.Name().String()))
		if err != nil {
			return fmt.Errorf("resolving remote reference OCI layer: %w", err)
		}
		packLayers[layer] = struct{}{}
	}

	// resolve digests into full descriptors
	// TODO: we should parallelize this, but we don't yet know how safe this
	// is to do concurrently
	for dgst := range packLayers {
		slog.DebugContext(ctx, "fetching packfile", "layerDigest", dgst)
		rc, err := action.remote.FetchLayer(ctx, dgst)
		if err != nil {
			return fmt.Errorf("fetching packfile layer: %w", err)
		}
		defer rc.Close()

		// TODO: we may want to use packfile.WritePackfileToObjectStorage directly
		// what's the difference here? writing the objects themselves rather than the
		// entire packfile? If the objects already exist in another packfile will they be duplicated?
		st, ok := action.localRepo.Storer.(storer.Storer)
		if !ok {
			return fmt.Errorf("repository storer is not a storer.Storer")
		}
		slog.DebugContext(ctx, "updating object storage with packfile")
		if err := packfile.UpdateObjectStorage(st, rc); err != nil {
			return fmt.Errorf("updating object storage with packfile: %w", err)
		}
	}
	slog.InfoContext(ctx, "done fetching packfiles")

	if err := action.batcher.Flush(true); err != nil {
		return fmt.Errorf("writing newline to git after fetch: %w", err)
	}

	// TODO: consider repacking repo objects

	return nil
}

// parseFetch parses a fetch command received from Git, returning it as a go-git reference.
func parseFetch(c cmd.Git) (*plumbing.Reference, error) {
	if len(c.Data) < 2 {
		return nil, fmt.Errorf("insufficient number of arguments in fetch command got %d, expected 2", len(c.Data))
	}
	hash := c.Data[0]
	name := c.Data[1]

	return plumbing.NewHashReference(plumbing.ReferenceName(name), plumbing.NewHash(hash)), nil
}
