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

// TODO: compareRefs is intended for resolving the min set of hashes needed for a thin packfile.
func (action *GitOCI) compareRefs(localRepo *git.Repository, localRef, remoteRef *plumbing.Reference) (status, digest.Digest, error) {
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

	remoteCommit, err := localRepo.CommitObject(remoteRef.Hash())
	if err != nil {
		return s, "", fmt.Errorf("resolving commit object from hash for remote ref: %w", err)
	}

	localCommit, err := localRepo.CommitObject(localRef.Hash())
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

	layer, err := action.remote.CommitExists(localRepo, localCommit)
	if err != nil {
		return s, "", fmt.Errorf("resolving existance of commit %s in remote: %w", localCommit, err)
	}
	if layer == "" {
		s = s | statusAddCommit
	}

	return s, layer, nil
}

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

	repo, err := git.PlainOpen(action.gitDir)
	if err != nil {
		return fmt.Errorf("opening local repository: %w", err)
	}
	// use pkg dotgit rather than git so we have access to manage packfiles
	// repo2 := dotgit.New(osfs.New(action.gitDir))

	// HACK: For now we're creating an packfile of the entire repo,
	// but we'll keep this around for testing
	// resolve state of refs in remote
	newCommits := make([]plumbing.Hash, 0)
	for _, c := range cmds {
		l, r, err := parseRefPair(c)
		if err != nil {
			return fmt.Errorf("parsing push command: %w", err)
		}

		localRef, err := resolveLocal(ctx, repo, l)
		if err != nil {
			return err
		}
		slog.InfoContext(ctx, "resolved local reference", "ref", l.String(), "hash", localRef.Hash().String())

		remoteRef, err := action.remote.ResolveRef(ctx, r)
		switch {
		case err != nil:
			return err
		case remoteRef == nil:
		default:
			slog.InfoContext(ctx, "resolved remote reference", "ref", l.String(), "hash", remoteRef.Hash().String())
		}

		refStatus, layer, err := action.compareRefs(repo, localRef, remoteRef)
		if err != nil {
			return fmt.Errorf("comparing local ref %sand remote ref %s: %w", localRef.Name().String(), remoteRef.Name().String(), err)
		}

		switch {
		case (refStatus & statusDelete) == statusDelete:
			action.remote.DeleteRef(ctx, remoteRef.Name())
		case (refStatus & statusForce) == statusForce:
			fallthrough
		case (refStatus & statusAddCommit) == statusAddCommit:
			// sanity
			// TODO: ideally, this isn't necessary if we test properly
			if layer == "" {
				return fmt.Errorf("expected OCI layer with ref status add commit")
			}
			newCommits = append(newCommits, localRef.Hash())
			fallthrough
		case (refStatus & statusUpdateRef) == statusUpdateRef:
			action.remote.UpdateRef(ctx, *plumbing.NewHashReference(remoteRef.Name(), localRef.Hash()), layer)
		default:
			// where did we go wrong?
			// return fmt.Errorf("insufficient handling of reference comparison for local ref %s and remote ref %s", localRef.Name().String(), remoteRef.Name().String())
			// TODO: add a "skip" status when refs are skipped due to lack of support for its type?
			// without it, the above error hits those cases where we log the skip elsewhere
		}

	}

	// TODO: resolve common ancestors for thin pack

	// TODO: if not common ancestors (bad object?) then we must pull down everything from OCI, rebuild into a repo, and resolve. OR we could just require the user to force push; isn't this what Git requires anyhow?

	// HACK
	packHash, err := packAll(repo)
	if err != nil {
		return fmt.Errorf("building packfile: %w", err)
	}

	// TODO: hopefully this isn't necessary, and we can open a reader using go-git methods
	packPath := path.Join(action.gitDir, "objects", "pack", fmt.Sprintf("pack-%s.pack", packHash.String()))
	// idxPath := path.Join(action.gitDir, "objects", "pack", fmt.Sprintf("pack-%s.idx", packHash.String()))

	buildOCI(ctx, action.remote, packPath)

	return fmt.Errorf("not implemented")
}

func buildOCI(ctx context.Context, remote model.Modeler, packPath string) {
	remote.AddPack(ctx, packPath)
}

// resolveLocal resolves the hash of a local reference.
func resolveLocal(_ context.Context, repo *git.Repository, ref plumbing.ReferenceName) (*plumbing.Reference, error) {
	localRef, err := repo.Reference(ref, true)
	if err != nil {
		return nil, fmt.Errorf("resolving hash of local reference %s: %w", ref.String(), err)
	}
	return localRef, nil
}

// HACK: having trouble creating packfiles, let alone thin packs, so we'll do the entire repo for now. If needed, we can fallback to shelling out and contribute to go-git later.
func packAll(repo *git.Repository) (h plumbing.Hash, err error) {
	err = repo.RepackObjects(&git.RepackConfig{UseRefDeltas: true})
	if err != nil {
		return h, fmt.Errorf("repacking all objects: %w", err)
	}

	pos, ok := repo.Storer.(storer.PackedObjectStorer)
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
func createPack(repo *git.Repository, hashes []plumbing.Hash) (h plumbing.Hash, err error) {
	// reference implementation: https://github.com/go-git/go-git/blob/v5.16.2/repository.go#L1815
	pfw, ok := repo.Storer.(storer.PackfileWriter)
	if !ok {
		return h, fmt.Errorf("repository storer is not a storer.PackfileWriter")
	}
	wc, err := pfw.PackfileWriter()
	if err != nil {
		return h, fmt.Errorf("initializing packfile writer: %w", err)
	}

	// TODO: What is a ref delta?
	enc := packfile.NewEncoder(wc, repo.Storer, true)
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
