package actions

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path"
	"path/filepath"
	"strings"

	"github.com/act3-ai/gitoci/internal/cmd"
	"github.com/act3-ai/gitoci/internal/ociutil/model"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/packfile"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// TODO: passing around *git.Repository and oci.ConfigGit may beg for an interface

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

	rc := newRefComparer(action.localRepo, action.remote)

	// resolve state of refs in remote
	newCommits := make([]plumbing.Hash, 0)          // TODO: not used properly yet, but will be when thin packs are handled properly
	refsInNewPack := make([]*plumbing.Reference, 0) // len <= newCommites
	for _, c := range cmds {
		// TODO: split this monstrosity
		l, r, force, err := parseRefPair(c)
		if err != nil {
			return fmt.Errorf("parsing push command: %w", err)
		}

		rp, err := rc.Compare(ctx, force, l, r)
		if errors.Is(err, model.ErrUnsupportedReferenceType) {
			slog.WarnContext(ctx, "encountered unsupported reference type when resolving remote reference", "err", err.Error())
			continue
		}
		if err != nil {
			return fmt.Errorf("comparing local ref %s to remote ref %s: %w", l.String(), r.String(), err)
		}

		switch {
		case (rp.status & statusDelete) == statusDelete:
			err := action.remote.DeleteRef(ctx, r)
			if errors.Is(err, model.ErrUnsupportedReferenceType) {
				slog.WarnContext(ctx, "encountered unsupported reference type when deleting remote reference", "ref", r.String())
				continue
			}
			if err != nil {
				return err
			}
		case (rp.status & statusForce) == statusForce:
			fallthrough
		case (rp.status & statusAddCommit) == statusAddCommit:
			newCommits = append(newCommits, rp.local.Hash())
			if rp.layer == "" {
				// defer the ref update until we know the packfile layer digest
				refsInNewPack = append(refsInNewPack, plumbing.NewHashReference(rp.remote.Name(), rp.local.Hash()))
				continue
			}
			fallthrough
		case (rp.status & statusUpdateRef) == statusUpdateRef:
			// update remote ref's commit to local ref's
			err := action.remote.UpdateRef(ctx, plumbing.NewHashReference(rp.remote.Name(), rp.local.Hash()), rp.layer)
			if errors.Is(err, model.ErrUnsupportedReferenceType) {
				slog.WarnContext(ctx, "encountered unsupported reference type when updating remote reference", "ref", rp.remote.Name().String())
				continue
			}
			if err != nil {
				return err
			}
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
	packHash, err := action.packAll()
	if err != nil {
		return fmt.Errorf("building packfile: %w", err)
	}

	// TODO: hopefully this isn't necessary, and we can open a reader using go-git methods
	packPath, err := filepath.Abs(path.Join(action.gitDir, "objects", "pack", fmt.Sprintf("pack-%s.pack", packHash.String())))
	if err != nil {
		return fmt.Errorf("resolving absolute path: %w", err)
	}
	// idxPath := path.Join(action.gitDir, "objects", "pack", fmt.Sprintf("pack-%s.idx", packHash.String()))

	_, err = action.remote.AddPack(ctx, packPath, refsInNewPack...)
	// TODO: we're silently ignoring this error
	if err != nil && !errors.Is(err, model.ErrUnsupportedReferenceType) {
		return fmt.Errorf("adding packfile to OCI data model: %w", err)
	}

	desc, err := action.remote.Push(ctx, action.addess)
	if err != nil {
		return fmt.Errorf("pushing to remote: %w", err)
	}
	slog.InfoContext(ctx, "successfully pushed to remote", "address", action.addess, "digest", desc.Digest, "size", desc.Size)

	return nil
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
// The returned boolean indicates a force update should be performed..
func parseRefPair(c cmd.Git) (plumbing.ReferenceName, plumbing.ReferenceName, bool, error) {
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
