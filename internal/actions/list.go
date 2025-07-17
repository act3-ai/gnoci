package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/act3-ai/gitoci/internal/ociutil"
	"github.com/act3-ai/gitoci/pkg/oci"
	"github.com/go-git/go-git/v5"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
)

// list handles the `list` command. Lists refs, one per line.
func (action *GitOCI) list(ctx context.Context) error {

	gt, err := ociutil.NewGraphTarget(ctx, action.addess)
	if err != nil {
		return err
	}

	slog.DebugContext(ctx, "resolving manifest descriptor")
	manDesc, err := gt.Resolve(ctx, action.addess)
	if err != nil {
		return fmt.Errorf("resolving manifest descriptor: %w", err)
	}

	slog.DebugContext(ctx, "fetching manifest")
	manRaw, err := content.FetchAll(ctx, gt, manDesc)
	if err != nil {
		return fmt.Errorf("fetching manifest: %w", err)
	}

	var manifest ocispec.Manifest
	if err := json.Unmarshal(manRaw, &manifest); err != nil {
		return fmt.Errorf("decoding manifest: %w", err)
	}

	slog.DebugContext(ctx, "fetching config")
	cfgRaw, err := content.FetchAll(ctx, gt, manifest.Config)
	if err != nil {
		return fmt.Errorf("fetching config: %w", err)
	}

	var config oci.ConfigGit
	if err := json.Unmarshal(cfgRaw, &config); err != nil {
		return fmt.Errorf("decoding config: %w", err)
	}

	localRepo, err := git.PlainOpen(action.gitDir)
	if err != nil {
		return fmt.Errorf("opening local repository: %w", err)
	}

	headRef, err := localRepo.Head()
	if err != nil {
		return fmt.Errorf("resolving current HEAD: %w", err)
	}
	slog.InfoContext(ctx, "head ref", "target", headRef.Target().String(), "name", headRef.Name().String())

	// TODO: what about refs/remotes/<shortname>/<ref>
	for k, v := range config.Heads {
		if k == strings.TrimPrefix(headRef.Name().String(), "refs/heads/") {
			s := fmt.Sprintf("@%s HEAD", headRef.Name())
			if err := action.batcher.Write(ctx, s); err != nil {
				return fmt.Errorf("writing ref to Git: %w", err)
			}
		}
		// TODO: head is likely the current branch HEAD, not necessarily main
		s := fmt.Sprintf("%s refs/heads/%s", v.Commit, k)
		if err := action.batcher.Write(ctx, s); err != nil {
			return fmt.Errorf("writing ref to Git: %w", err)
		}
	}

	for k, v := range config.Tags {
		s := fmt.Sprintf("%s refs/tags/%s", v.Commit, k)
		if err := action.batcher.Write(ctx, s); err != nil {
			return fmt.Errorf("writing ref to Git: %w", err)
		}
	}

	if err := action.batcher.Flush(true); err != nil {
		return fmt.Errorf("flushing writer: %w", err)
	}

	return nil
}

// listForPush handles the `list for-push` command.
// Similar to list, except only used to prepare for a push.
// func (action *GitOCI) listForPush() error {

// }

// TODO: This func lists local refs, which we likley don't even need as git should do this for us,
// but we'll keep this code block around as a comment for now incase we need it.
// func (action *GitOCI) list() error {

// 	//TODO: until fetch/push is implemented we'll focus on the output for now.

// 	localRepo, err := git.PlainOpen(action.gitDir)
// 	if err != nil {
// 		return fmt.Errorf("opening local repository: %w", err)
// 	}

// 	// TODO: Is there really no other option than an iter?
// 	rIter, err := localRepo.References()
// 	if err != nil {
// 		return fmt.Errorf("getting reference iter: %w", err)
// 	}

// 	localHashRefs := make([]string, 0, 1)
// 	_ = rIter.ForEach(func(r *plumbing.Reference) error {
// 		localHashRefs = append(localHashRefs,
// 			fmt.Sprintf("%s %s", r.Hash().String(), r.Name().String()))
// 		return nil
// 	})

// 	if err := action.batcher.WriteBatch(localHashRefs...); err != nil {
// 		return fmt.Errorf("writing local refs: %w", err)
// 	}

// 	return fmt.Errorf("not implemented")
// }
