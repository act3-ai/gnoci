package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/packfile"
	"github.com/go-git/go-git/v5/plumbing/revlist"
	"github.com/go-git/go-git/v5/plumbing/storer"

	"github.com/act3-ai/gnoci/internal/git"
	"github.com/act3-ai/gnoci/internal/model"
	"github.com/act3-ai/gnoci/internal/refcomp"
	gittypes "github.com/act3-ai/gnoci/pkg/protocol/git"
	"github.com/act3-ai/gnoci/pkg/protocol/git/comms"
)

// HandlePush executes a batch of push commands.
func HandlePush(ctx context.Context, local git.Repository, localDir string, remote model.Modeler, comm comms.Communicator) error {
	reqs, err := comm.ParsePushRequestBatch()
	if err != nil {
		return fmt.Errorf("parsing push request batch: %w", err)
	}

	// compare local refs to remote
	newCommits, refsInNewPack, results := compareRefs(ctx, local, remote, reqs)

	// resolve new reachable objects from new commit set
	newReachableObjs, err := reachableObjs(local, remote, newCommits)
	if err != nil {
		return fmt.Errorf("resolving reachable objects not already in remote: %w", err)
	}

	// make temp repo for writing the packfile, without affecting the true local
	tmpDir, err := os.MkdirTemp("", "*")
	if err != nil {
		return fmt.Errorf("initializing temp directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			slog.ErrorContext(ctx, "removing temporary git repository", slog.String("error", err.Error()))
		}
	}()

	tmpRepo, err := gogit.PlainInitWithOptions(tmpDir, &gogit.PlainInitOptions{})
	if err != nil {
		return fmt.Errorf("initializing temp repository for packfile storage: %w", err)
	}

	packHash, err := createPack(local, git.NewRepository(tmpRepo), newReachableObjs)
	if err != nil {
		return fmt.Errorf("creating packfile: %w", err)
	}

	// TODO: hopefully this isn't necessary, and we can open a reader using go-git methods
	packPath, err := filepath.Abs(path.Join(tmpDir, ".git", "objects", "pack", fmt.Sprintf("pack-%s.pack", packHash.String())))
	if err != nil {
		return fmt.Errorf("resolving absolute path: %w", err)
	}

	_, err = remote.AddPack(ctx, packPath, refsInNewPack...)
	switch {
	case errors.Is(err, model.ErrUnsupportedReferenceType):
		// TODO: this should be reported to git, but we need to change how the errors a propagated as we need to report them by reference, not a single error
		slog.ErrorContext(ctx, "failed to update remote with unsupported reference", slog.String("error", err.Error()))
	case err != nil:
		return fmt.Errorf("adding packfile to OCI data model: %w", err)
	}

	var referrerUpdates []model.ReferrerUpdater
	lfsModeler, ok := remote.(model.LFSModeler)
	if ok {
		referrerUpdates = append(referrerUpdates, model.UpdateLFSReferrer(lfsModeler))
	}

	desc, err := remote.Push(ctx, referrerUpdates...)
	if err != nil {
		return fmt.Errorf("pushing to remote: %w", err)
	}
	slog.InfoContext(ctx, "successfully pushed to remote", "address", remote.Ref(), "digest", desc.Digest, "size", desc.Size)

	if err := comm.WritePushResponse(results); err != nil {
		return fmt.Errorf("writing push response: %w", err)
	}

	return nil
}

// compareRefs compares all references in the set of push cmds between the local
// and remote repositories, returning a set of new commit hashes, references to
// commits in the to-be-created packfile, and a list of results to be written to Git.
func compareRefs(ctx context.Context, local git.Repository, remote model.Modeler, reqs []gittypes.PushRequest) ([]plumbing.Hash, []*plumbing.Reference, []gittypes.PushResponse) {
	rc := refcomp.NewCachedRefComparer(local, remote)

	// resolve state of refs in remote
	newCommitTips := make(map[plumbing.Hash]struct{})
	refsInNewPack := make([]*plumbing.Reference, 0) // len <= newCommites
	results := make([]gittypes.PushResponse, 0, len(reqs))
	for _, req := range reqs {
		rp, err := rc.Compare(ctx, req.Force, req.Src, req.Remote)
		if errors.Is(err, model.ErrUnsupportedReferenceType) {
			result := gittypes.PushResponse{
				Remote: req.Remote,
				Error:  fmt.Errorf("encountered unsupported reference type when comparing local to remote ref: %w", err),
			}
			results = append(results, result)
			continue
		}
		if err != nil {
			result := gittypes.PushResponse{
				Remote: req.Remote,
				Error:  fmt.Errorf("comparing local ref to remote ref: %w", err),
			}
			results = append(results, result)
			continue
		}

		switch {
		case (rp.Status & refcomp.StatusDelete) == refcomp.StatusDelete:
			err := remote.DeleteRef(ctx, req.Remote)
			if errors.Is(err, model.ErrUnsupportedReferenceType) {
				result := gittypes.PushResponse{
					Remote: req.Remote,
					Error:  fmt.Errorf("encountered unsupported reference type when deleting remote ref: %w", err),
				}
				results = append(results, result)
				continue
			}
			if err != nil {
				result := gittypes.PushResponse{
					Remote: req.Remote,
					Error:  fmt.Errorf("deleting reference from remote: %w", err),
				}
				results = append(results, result)
				continue
			}
		case (rp.Status & refcomp.StatusForce) == refcomp.StatusForce:
			fallthrough
		case (rp.Status & refcomp.StatusAddCommit) == refcomp.StatusAddCommit:
			newCommitTips[rp.Local.Hash()] = struct{}{}
			fallthrough
		case (rp.Status & refcomp.StatusUpdateRef) == refcomp.StatusUpdateRef:
			if rp.Layer == "" {
				// defer the ref update until we know the packfile layer digest
				refsInNewPack = append(refsInNewPack, plumbing.NewHashReference(rp.Remote.Name(), rp.Local.Hash()))
				result := gittypes.PushResponse{Remote: req.Remote}
				results = append(results, result)
				continue
			}
			// update remote ref's commit to local ref's
			err := remote.UpdateRef(ctx, plumbing.NewHashReference(rp.Remote.Name(), rp.Local.Hash()), rp.Layer)
			if errors.Is(err, model.ErrUnsupportedReferenceType) {
				result := gittypes.PushResponse{
					Remote: req.Remote,
					Error:  fmt.Errorf("encountered unsupported reference type when updating remote ref: %w", err),
				}
				results = append(results, result)
				continue
			}
			if err != nil {
				result := gittypes.PushResponse{
					Remote: req.Remote,
					Error:  fmt.Errorf("updating remote reference: %w", err),
				}
				results = append(results, result)
				continue
			}
		default:
			// where did we go wrong?
			// return fmt.Errorf("insufficient handling of reference comparison for local ref %s and remote ref %s", localRef.Name().String(), remoteRef.Name().String())
			// TODO: add a "skip" Status when refs are skipped due to lack of support for its type?
			// without it, the above error hits those cases where we log the skip elsewhere
		}
		result := gittypes.PushResponse{Remote: req.Remote}
		results = append(results, result)
	}

	dedupNewCommits := make([]plumbing.Hash, 0, len(newCommitTips))
	for commit := range newCommitTips {
		dedupNewCommits = append(dedupNewCommits, commit)
	}
	slog.DebugContext(ctx, "resolved new commits", "newCommits", fmt.Sprintf("%v", dedupNewCommits))

	return dedupNewCommits, refsInNewPack, results
}

// reachableObjs resolves ALL commits reachable from newCommits, excluding those
// existing in the remote.
func reachableObjs(local git.Repository, remote model.Modeler, newCommits []plumbing.Hash) ([]plumbing.Hash, error) {
	headRefs := remote.HeadRefs()
	tagRefs := remote.TagRefs()
	ignoreCommits := make([]plumbing.Hash, 0, len(tagRefs)+len(headRefs))

	for _, refInfo := range headRefs {
		ignoreCommits = append(ignoreCommits, plumbing.NewHash(refInfo.Commit)) // TODO: is NewHash what we want?
	}
	// TODO: are tags necessary? We may be able to get away without them.
	for _, refInfo := range tagRefs {

		ignoreCommits = append(ignoreCommits, plumbing.NewHash(refInfo.Commit))
	}

	newReachableObjs, err := revlist.Objects(local.Storer(), newCommits, ignoreCommits)
	if err != nil {
		return nil, fmt.Errorf("resolving new reachable objects: %w", err)
	}

	return newReachableObjs, nil
}

// createPack builds a packfile using a set of hashes.
func createPack(local, tmp git.Repository, hashes []plumbing.Hash) (h plumbing.Hash, err error) {
	// reference implementation: https://github.com/go-git/go-git/blob/v5.16.2/repository.go#L1815
	pfw, ok := tmp.Storer().(storer.PackfileWriter)
	if !ok {
		return h, fmt.Errorf("repository storer is not a storer.PackfileWriter")
	}
	wc, err := pfw.PackfileWriter()
	if err != nil {
		return h, fmt.Errorf("initializing packfile writer: %w", err)
	}
	defer wc.Close()

	// TODO: What is a ref delta?
	enc := packfile.NewEncoder(wc, local.Storer(), true)
	h, err = enc.Encode(hashes, 10) // default window
	if err != nil {
		return h, fmt.Errorf("encoding packfile: %w", err)
	}

	return h, nil
}
