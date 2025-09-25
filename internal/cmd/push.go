package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/packfile"
	"github.com/go-git/go-git/v5/plumbing/revlist"
	"github.com/go-git/go-git/v5/plumbing/storer"

	"github.com/act3-ai/gnoci/internal/ociutil/model"
	"github.com/act3-ai/gnoci/internal/refcomp"
)

// HandlePush executes a batch of push commands.
func HandlePush(ctx context.Context, local *git.Repository, localDir string, remote model.Modeler, remoteAddress string, cmds []Git, bw BatchWriter) error {
	// compare local refs to remote
	newCommits, refsInNewPack, results := compareRefs(ctx, local, remote, cmds)

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

	tmpRepo, err := git.PlainInitWithOptions(tmpDir, &git.PlainInitOptions{})
	if err != nil {
		return fmt.Errorf("initializing temp repository for packfile storage: %w", err)
	}

	packHash, err := createPack(local, tmpRepo, newReachableObjs)
	if err != nil {
		return fmt.Errorf("creating packfile: %w", err)
	}

	// TODO: hopefully this isn't necessary, and we can open a reader using go-git methods
	packPath, err := filepath.Abs(path.Join(tmpDir, ".git", "objects", "pack", fmt.Sprintf("pack-%s.pack", packHash.String())))
	if err != nil {
		return fmt.Errorf("resolving absolute path: %w", err)
	}
	// idxPath := path.Join(action.gitDir, "objects", "pack", fmt.Sprintf("pack-%s.idx", packHash.String()))

	_, err = remote.AddPack(ctx, packPath, refsInNewPack...)
	switch {
	case errors.Is(err, model.ErrUnsupportedReferenceType):
		// TODO: this should be reported to git, but we need to change how the errors a propagated as we need to report them by reference, not a single error
		slog.ErrorContext(ctx, "failed to update remote with unsupported reference", slog.String("error", err.Error()))
	case err != nil:
		return fmt.Errorf("adding packfile to OCI data model: %w", err)
	}

	// TODO: we're silently ignoring this error
	if err != nil && !errors.Is(err, model.ErrUnsupportedReferenceType) {
		return fmt.Errorf("adding packfile to OCI data model: %w", err)
	}

	desc, err := remote.Push(ctx, remoteAddress)
	if err != nil {
		return fmt.Errorf("pushing to remote: %w", err)
	}
	slog.InfoContext(ctx, "successfully pushed to remote", "address", remoteAddress, "digest", desc.Digest, "size", desc.Size)

	if err := bw.WriteBatch(ctx, results...); err != nil {
		return fmt.Errorf("writing push results to git: %w", err)
	}

	return nil
}

// compareRefs compares all references in the set of push cmds between the local
// and remote repositories, returning a set of new commit hashes, references to
// commits in the to-be-created packfile, and a list of results to be written to Git.
func compareRefs(ctx context.Context, local *git.Repository, remote model.Modeler, cmds []Git) ([]plumbing.Hash, []*plumbing.Reference, []string) {
	rc := refcomp.NewCachedRefComparer(local, remote)

	// resolve state of refs in remote
	newCommitTips := make(map[plumbing.Hash]struct{})
	refsInNewPack := make([]*plumbing.Reference, 0) // len <= newCommites
	results := make([]string, 0, len(cmds))
	for _, c := range cmds {
		l, r, force, err := parseRefPair(c)
		if err != nil {
			results = append(results, fmtResult(false, r, fmt.Errorf("parsing push command: %w", err).Error()))
			continue
		}

		rp, err := rc.Compare(ctx, force, l, r)
		if errors.Is(err, model.ErrUnsupportedReferenceType) {
			results = append(results, fmtResult(false, r, fmt.Errorf("encountered unsupported reference type when comparing local to remote ref: %w", err).Error()))
			continue
		}
		if err != nil {
			results = append(results, fmtResult(false, r, fmt.Errorf("comparing local ref to remote ref: %w", err).Error()))
			continue
		}

		switch {
		case (rp.Status & refcomp.StatusDelete) == refcomp.StatusDelete:
			err := remote.DeleteRef(ctx, r)
			if errors.Is(err, model.ErrUnsupportedReferenceType) {
				results = append(results, fmtResult(false, r, fmt.Errorf("encountered unsupported reference type when deleting remote ref: %w", err).Error()))
				continue
			}
			if err != nil {
				results = append(results, fmtResult(false, r, fmt.Errorf("deleting reference from remote: %w", err).Error()))
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
				results = append(results, fmtResult(true, r, ""))
				continue
			}
			// update remote ref's commit to local ref's
			err := remote.UpdateRef(ctx, plumbing.NewHashReference(rp.Remote.Name(), rp.Local.Hash()), rp.Layer)
			if errors.Is(err, model.ErrUnsupportedReferenceType) {
				results = append(results, fmtResult(false, r, fmt.Errorf("encountered unsupported reference type when updating remote ref: %w", err).Error()))
				continue
			}
			if err != nil {
				results = append(results, fmtResult(false, r, fmt.Errorf("updating remote reference: %w", err).Error()))
				continue
			}
		default:
			// where did we go wrong?
			// return fmt.Errorf("insufficient handling of reference comparison for local ref %s and remote ref %s", localRef.Name().String(), remoteRef.Name().String())
			// TODO: add a "skip" Status when refs are skipped due to lack of support for its type?
			// without it, the above error hits those cases where we log the skip elsewhere
		}
		results = append(results, fmtResult(true, r, ""))
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
func reachableObjs(local *git.Repository, remote model.Modeler, newCommits []plumbing.Hash) ([]plumbing.Hash, error) {
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

	newReachableObjs, err := revlist.Objects(local.Storer, newCommits, ignoreCommits)
	if err != nil {
		return nil, fmt.Errorf("resolving new reachable objects: %w", err)
	}

	return newReachableObjs, nil
}

// HACK: having trouble creating packfiles, let alone thin packs, so we'll do the entire repo for now. If needed, we can fallback to shelling out and contribute to go-git later.
// func packAll(local *git.Repository) (h plumbing.Hash, err error) {
// 	err = local.RepackObjects(&git.RepackConfig{UseRefDeltas: true})
// 	if err != nil {
// 		return h, fmt.Errorf("repacking all objects: %w", err)
// 	}

// 	pos, ok := local.Storer.(storer.PackedObjectStorer)
// 	if !ok {
// 		return h, fmt.Errorf("repository storer is not a storer.PackedObjectStorer")
// 	}

// 	hs, err := pos.ObjectPacks()
// 	switch {
// 	case err != nil:
// 		return h, fmt.Errorf("listing local object packs: %w", err)

// 	case len(hs) != 1:
// 		return h, fmt.Errorf("expected 1 packfile, got %d", len(hs))
// 	default:
// 		return hs[0], nil
// 	}
// }

// createPack builds a packfile using a set of hashes.
func createPack(local, tmp *git.Repository, hashes []plumbing.Hash) (h plumbing.Hash, err error) {
	// reference implementation: https://github.com/go-git/go-git/blob/v5.16.2/repository.go#L1815
	pfw, ok := tmp.Storer.(storer.PackfileWriter)
	if !ok {
		return h, fmt.Errorf("repository storer is not a storer.PackfileWriter")
	}
	wc, err := pfw.PackfileWriter()
	if err != nil {
		return h, fmt.Errorf("initializing packfile writer: %w", err)
	}
	defer wc.Close()

	// TODO: What is a ref delta?
	enc := packfile.NewEncoder(wc, local.Storer, true)
	h, err = enc.Encode(hashes, 10) // default window
	if err != nil {
		return h, fmt.Errorf("encoding packfile: %w", err)
	}

	return h, nil
}

// parseRefPair validates a reference pair, <local>:<remote>, returning the local and remote references respectively.
// The returned boolean indicates a force update should be performed..
func parseRefPair(c Git) (plumbing.ReferenceName, plumbing.ReferenceName, bool, error) {
	if c.Data == nil {
		return "", "", false, fmt.Errorf("no arguments in push command")
	}

	pair := c.Data[0]
	if pair == "" {
		return "", "", false, errors.New("empty reference pair string, expected <local>:<remote>")
	}

	s := strings.Split(pair, ":")
	if len(s) != 2 {
		return "", "", false, fmt.Errorf("failed to split reference pair string, got %s, expected <local>:<remote>", pair)
	}
	local := s[0]
	remote := s[1]

	var force bool
	if strings.HasPrefix(local, "+") {
		force = true
		local = strings.TrimPrefix(local, "+")
	}

	return plumbing.ReferenceName(local), plumbing.ReferenceName(remote), force, nil
}

// fmtResult aids in formating a result string written to git. Why is unnecessary if ok == true.
func fmtResult(ok bool, dst plumbing.ReferenceName, why string) string {
	if ok {
		return fmt.Sprintf("ok %s", dst.String())
	}
	return fmt.Sprintf("error %s %s?", dst.String(), why)
}
