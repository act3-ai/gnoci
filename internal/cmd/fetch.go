package cmd

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-git/go-git/v5/plumbing/format/packfile"
	"github.com/go-git/go-git/v5/plumbing/storer"

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

	_, err = comm.ParseFetchRequestBatch()
	if err != nil {
		return fmt.Errorf("parsing fetch request batch: %w", err)
	}

	// HACK: Performance here is terrible, we always fetch all packfiles to ensure
	// all history is complete. The main difficulty here is we don't know what's
	// in the packfiles, calling for an update to the data model.
	for rc, err := range remote.FetchLayersReverse(ctx) {
		if err != nil {
			return fmt.Errorf("fetching packfile: %w", err)
		}
		defer rc.Close() //nolint:revive

		st, ok := local.Storer().(storer.Storer)
		if !ok {
			return fmt.Errorf("repository storer is not a storer.Storer")
		}

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
