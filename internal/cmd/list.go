package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"

	"github.com/act3-ai/gnoci/internal/git"
	"github.com/act3-ai/gnoci/internal/model"
	gittypes "github.com/act3-ai/gnoci/pkg/protocol/git"
	"github.com/act3-ai/gnoci/pkg/protocol/git/comms"
)

// HandleList executes the list command. Lists refs one per line.
func HandleList(ctx context.Context, local git.Repository, remote model.Modeler, comm comms.Communicator) error {
	req, err := comm.ParseListRequest()
	if err != nil {
		return fmt.Errorf("parsing list request: %w", err)
	}

	var headRef *plumbing.Reference
	if !req.ForPush && local != nil {
		// discover local HEAD ref, so we know what to resolve in the remote
		headRef, err = local.Head()
		if err != nil {
			// TODO: can we assume main/master if local HEAD DNE?
			slog.InfoContext(ctx, "local HEAD not found")
		} else {
			slog.InfoContext(ctx, "head ref", "target", headRef.Hash().String(), "name", headRef.Name().String())
		}
	}

	// TODO: what about refs/remotes/<shortname>/<ref>

	headRefs := remote.HeadRefs()
	tagRefs := remote.TagRefs()
	results := make([]gittypes.ListResponse, len(headRefs)+len(tagRefs))

	// list remote branch references
	for k, v := range headRefs {
		// list HEAD if one exists locally
		if headRef != nil && (k.String() == strings.TrimPrefix(headRef.Name().String(), "refs/heads/")) {
			result := gittypes.ListResponse{
				Reference: plumbing.ReferenceName(fmt.Sprintf("@%s", headRef.Name())),
				Commit:    "HEAD",
			}
			results = append(results, result)
		}

		result := gittypes.ListResponse{
			Reference: k,
			Commit:    v.Commit,
		}
		results = append(results, result)
	}

	// list remote tag references
	for k, v := range tagRefs {
		result := gittypes.ListResponse{
			Reference: k,
			Commit:    v.Commit,
		}
		results = append(results, result)
	}

	if err := comm.WriteListResponse(results); err != nil {
		return fmt.Errorf("writing list response: %w", err)
	}

	return nil
}
