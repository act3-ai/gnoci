package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"

	"github.com/act3-ai/gnoci/internal/ociutil/model"
)

// HandleList executes the list command. Lists refs one per line.
func HandleList(ctx context.Context, local *git.Repository, remote model.Modeler, forPush bool, g Git, w Writer) error {
	var headRef *plumbing.Reference
	var err error
	if !forPush && local != nil {
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

	// list remote branch references
	for k, v := range remote.HeadRefs() {
		// list HEAD if one exists locally
		if headRef != nil && (k.String() == strings.TrimPrefix(headRef.Name().String(), "refs/heads/")) {
			if err := w.Write(ctx, fmt.Sprintf("@%s HEAD", headRef.Name())); err != nil {
				return fmt.Errorf("writing HEAD ref to Git: %w", err)
			}
		}

		if err := w.Write(ctx, fmt.Sprintf("%s %s", v.Commit, k)); err != nil {
			return fmt.Errorf("writing branch ref to Git: %w", err)
		}
	}

	// list remote tag references
	for k, v := range remote.TagRefs() {
		if err := w.Write(ctx, fmt.Sprintf("%s %s", v.Commit, k)); err != nil {
			return fmt.Errorf("writing tag ref to Git: %w", err)
		}
	}

	if err := w.Flush(true); err != nil {
		return fmt.Errorf("flushing writer: %w", err)
	}

	return nil
}
