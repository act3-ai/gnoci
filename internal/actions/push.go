package actions

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path"
	"strings"

	"github.com/act3-ai/gitoci/internal/cmd"
	"github.com/act3-ai/gitoci/internal/ociutil/model"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/packfile"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/opencontainers/go-digest"
)

// TODO: passing around *git.Repository and oci.ConfigGit may beg for an interface

type status uint8

const (
	// statusDelete indicates a ref should be removed from the remote
	statusDelete status = 1 << iota
	// statusUpdateRef indicates a statusUpdateRef should be updated in the remote
	statusUpdateRef
	// statusAddCommit indicates the ref's commit object should be added to the remote
	statusAddCommit
	// statusForce indicates a statusForce update should be performed
	statusForce
	// rewritten indicates history has been rewritten
	// TODO: necessary?
	// rewritten
)

// func sample(localRepo *git.Repository) error {
// 	tmpRepo := filesystem.NewStorageWithOptions(
// 		osfs.New(os.TempDir()),
// 		cache.NewObjectLRUDefault(),
// 		filesystem.Options{AlternatesFS: localRepo.Storer})
// 	git.InitWithOptions()

// }

// push handles the `push` command.
func (action *GitOCI) push(ctx context.Context, cmds []cmd.Git) error {
	if err := action.remote.FetchOrDefault(ctx, action.addess); err != nil {
		return fmt.Errorf("fetching remote metadta: %w", err)
	}

	var err error
	action.localRepo, err = git.PlainOpen(action.gitDir)
	if err != nil {
		return fmt.Errorf("opening local repository: %w", err)
	}

	// use pkg dotgit rather than git so we have access to manage packfiles
	// repo2 := dotgit.New(osfs.New(action.gitDir))

	// resolve state of refs in remote
	newCommits := make([]plumbing.Hash, 0)
	for _, c := range cmds {
		l, r, err := parseRefPair(c)
		if err != nil {
			return fmt.Errorf("parsing push command: %w", err)
		}

		localRef, err := action.localRepo.Reference(l, true)
		if err != nil {
			return fmt.Errorf("resolving hash of local reference %s: %w", l.String(), err)
		}
		slog.InfoContext(ctx, "resolved local reference", "ref", l.String(), "hash", localRef.Hash().String())

		remoteRef, err := action.remote.ResolveRef(ctx, r)
		switch {
		case errors.Is(err, model.ErrReferenceNotFound):
			remoteRef = plumbing.NewHashReference(r, plumbing.ZeroHash) // hash irrelavent, later we use the local hash
		case errors.Is(err, model.ErrUnsupportedReferenceType):
			slog.WarnContext(ctx, "encountered unsupported reference type when resolving remote reference", "ref", r.String())
			continue
		case err != nil:
			return err
		default:
			slog.InfoContext(ctx, "resolved remote reference", "ref", l.String(), "hash", remoteRef.Hash().String())
		}

		// update metadata as needed
		newCommit, err := action.updateRemoteMetadata(ctx, localRef, remoteRef)
		switch {
		case err != nil:
			return fmt.Errorf("updating remote metadata: %w", err)
		case newCommit.IsZero():
			// reference-only update
		default:
			newCommits = append(newCommits, newCommit)
		}
	}

	// TODO: resolve common ancestors for thin pack

	// TODO: if not common ancestors (bad object?) then we must pull down everything from OCI, rebuild into a repo, and resolve. OR we could just require the user to force push; isn't this what Git requires anyhow?

	// HACK
	packHash, err := action.packAll()
	if err != nil {
		return fmt.Errorf("building packfile: %w", err)
	}

	// TODO: hopefully this isn't necessary, and we can open a reader using go-git methods
	packPath := path.Join(action.gitDir, "objects", "pack", fmt.Sprintf("pack-%s.pack", packHash.String()))
	// idxPath := path.Join(action.gitDir, "objects", "pack", fmt.Sprintf("pack-%s.idx", packHash.String()))

	_, err = action.remote.AddPack(ctx, packPath)
	if err != nil {
		return fmt.Errorf("adding packfile to OCI data model: %w", err)
	}

	desc, err := action.remote.Push(ctx, action.addess)
	if err != nil {
		return fmt.Errorf("pushing to remote: %w", err)
	}
	slog.InfoContext(ctx, "successfully pushed to remote", "address", action.addess, "digest", desc.Digest, "size", desc.Size)

	return nil
}

func (action *GitOCI) updateRemoteMetadata(ctx context.Context, localRef, remoteRef *plumbing.Reference) (plumbing.Hash, error) {
	refStatus, layer, err := action.compareRefs(localRef, remoteRef)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("comparing local ref %sand remote ref %s: %w", localRef.Name().String(), remoteRef.Name().String(), err)
	}

	var newCommit plumbing.Hash
	switch {
	case (refStatus & statusDelete) == statusDelete:
		err := action.remote.DeleteRef(ctx, remoteRef.Name())
		if errors.Is(err, model.ErrUnsupportedReferenceType) {
			slog.WarnContext(ctx, "encountered unsupported reference type when deleting remote reference", "ref", remoteRef.Name().String())
			return plumbing.ZeroHash, nil
		}
		if err != nil {
			return plumbing.ZeroHash, err
		}
	case (refStatus & statusForce) == statusForce:
		fallthrough
	case (refStatus & statusAddCommit) == statusAddCommit:
		// sanity
		// TODO: ideally, this isn't necessary if we test properly
		if layer == "" {
			return plumbing.ZeroHash, fmt.Errorf("expected OCI layer with ref status add commit")
		}
		newCommit = localRef.Hash()
		fallthrough
	case (refStatus & statusUpdateRef) == statusUpdateRef:
		// update remote ref's commit to local ref's
		err := action.remote.UpdateRef(ctx, *plumbing.NewHashReference(remoteRef.Name(), localRef.Hash()), layer)
		if errors.Is(err, model.ErrUnsupportedReferenceType) {
			slog.WarnContext(ctx, "encountered unsupported reference type when updating remote reference", "ref", remoteRef.Name().String())
			return plumbing.ZeroHash, nil
		}
		if err != nil {
			return plumbing.ZeroHash, err
		}
	default:
		// where did we go wrong?
		// return fmt.Errorf("insufficient handling of reference comparison for local ref %s and remote ref %s", localRef.Name().String(), remoteRef.Name().String())
		// TODO: add a "skip" status when refs are skipped due to lack of support for its type?
		// without it, the above error hits those cases where we log the skip elsewhere
	}

	return newCommit, nil
}

// compareRefs resolves what's needed for a remoteRef to be updated to localRef.
// TODO: compareRefs is intended for resolving the min set of hashes needed for a thin packfile.
func (action *GitOCI) compareRefs(localRef, remoteRef *plumbing.Reference) (status, digest.Digest, error) {
	// TODO: this implementation attempts to resolve as much information about a ref comparison
	// as possible, but this is likely overkill. It may be better to short-circuit, e.g. if force
	// we don't care to resolve ancestral status of the remote & local refs.
	var s status

	// if local is empty, status += delete
	if localRef == nil {
		s = s | statusDelete
	}

	// TODO: uncomment when force is supported
	// if action.Force {
	// 	s = s | statusForce
	// }

	remoteCommit, err := action.localRepo.CommitObject(remoteRef.Hash())
	if err != nil {
		return s, "", fmt.Errorf("resolving commit object from hash for remote ref: %w", err)
	}

	localCommit, err := action.localRepo.CommitObject(localRef.Hash())
	if err != nil {
		return s, "", fmt.Errorf("resolving commit object from hash for local ref: %w", err)
	}

	// TODO: something smells off here...
	isAncestor, err := remoteCommit.IsAncestor(localCommit)
	if err != nil {
		return s, "", fmt.Errorf("resolving remote commit ancestor status of local: %w", err)
	}
	if isAncestor {
		s = s | statusUpdateRef
	}

	layer, err := action.remote.CommitExists(action.localRepo, localCommit)
	if err != nil {
		return s, "", fmt.Errorf("resolving existance of commit %s in remote: %w", localCommit, err)
	}
	if layer == "" {
		s = s | statusAddCommit
	}

	return s, layer, nil
}

// HACK: having trouble creating packfiles, let alone thin packs, so we'll do the entire repo for now. If needed, we can fallback to shelling out and contribute to go-git later.
func (action *GitOCI) packAll() (h plumbing.Hash, err error) {
	err = action.localRepo.RepackObjects(&git.RepackConfig{UseRefDeltas: true})
	if err != nil {
		return h, fmt.Errorf("repacking all objects: %w", err)
	}

	pos, ok := action.localRepo.Storer.(storer.PackedObjectStorer)
	if !ok {
		return h, fmt.Errorf("repository storer is not a storer.PackedObjectStorer")
	}

	hs, err := pos.ObjectPacks()
	switch {
	case err != nil:
		return h, err

	case len(hs) != 1:
		return h, fmt.Errorf("expected 1 packfile, got %d", len(hs))
	default:
		return hs[0], nil
	}
}

// createPack builds a packfile using a set of hashes.
// TODO: not used
func (action *GitOCI) createPack(hashes []plumbing.Hash) (h plumbing.Hash, err error) {
	// reference implementation: https://github.com/go-git/go-git/blob/v5.16.2/repository.go#L1815
	pfw, ok := action.localRepo.Storer.(storer.PackfileWriter)
	if !ok {
		return h, fmt.Errorf("repository storer is not a storer.PackfileWriter")
	}
	wc, err := pfw.PackfileWriter()
	if err != nil {
		return h, fmt.Errorf("initializing packfile writer: %w", err)
	}

	// TODO: What is a ref delta?
	enc := packfile.NewEncoder(wc, action.localRepo.Storer, true)
	h, err = enc.Encode(hashes, 10) // default window
	if err != nil {
		return h, fmt.Errorf("encoding packfile: %w", err)
	}
	return h, nil
}

// parseRefPair validates a reference pair, <local>:<remote>, returning the local and remote references respectively.
func parseRefPair(c cmd.Git) (plumbing.ReferenceName, plumbing.ReferenceName, error) {
	if c.Data == nil {
		return "", "", fmt.Errorf("no arguments in push command")
	}

	pair := c.Data[0]
	if pair == "" {
		return "", "", errors.New("empty reference pair string, expected <local>:<remote>")
	}

	s := strings.Split(pair, ":")
	if len(s) != 2 {
		return "", "", fmt.Errorf("failed to split reference pair string, got %s, expected <local>:<remote>", pair)
	}

	return plumbing.ReferenceName(s[0]), plumbing.ReferenceName(s[1]), nil
}
