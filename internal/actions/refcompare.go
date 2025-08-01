package actions

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/act3-ai/gitoci/internal/ociutil/model"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/opencontainers/go-digest"
)

type status uint8

const (
	// statusDelete indicates a ref should be removed from the remote.
	statusDelete status = 1 << iota
	// statusUpdateRef indicates a statusUpdateRef should be updated in the remote.
	statusUpdateRef
	// statusAddCommit indicates the ref's commit object should be added to the remote.
	statusAddCommit
	// statusForce indicates a statusForce update should be performed.
	statusForce
)

// func sample(localRepo *git.Repository) error {
// 	tmpRepo := filesystem.NewStorageWithOptions(
// 		osfs.New(os.TempDir()),
// 		cache.NewObjectLRUDefault(),
// 		filesystem.Options{AlternatesFS: localRepo.Storer})
// 	git.InitWithOptions()

// }

type refComparer interface {
	// Compare resolves the status of a remote reference.
	Compare(ctx context.Context, force bool, localName plumbing.ReferenceName, remoteName plumbing.ReferenceName) (refPair, error)
	// GetStatus returns the status of a remote reference.
	// GetStatus(remoteName plumbing.ReferenceName) (status, bool)
}

// refCompare implements refComparer.
type refCompare struct {
	local  *git.Repository
	remote model.Modeler

	refs map[plumbing.ReferenceName]refPair // key is remote ref name
}

type refPair struct {
	local  *plumbing.Reference
	remote *plumbing.Reference
	status
	layer digest.Digest // only populated if (status ^ statusAddCommit)
}

func newRefComparer(local *git.Repository, remote model.Modeler) refComparer {
	return &refCompare{
		local:  local,
		remote: remote,
		refs:   make(map[plumbing.ReferenceName]refPair, 0),
	}
}

func (rc *refCompare) Compare(ctx context.Context, force bool, localName, remoteName plumbing.ReferenceName) (refPair, error) {
	rp, ok := rc.refs[remoteName]
	if ok {
		return rp, nil
	}

	localRef, err := rc.local.Reference(localName, true)
	if err != nil {
		return refPair{}, fmt.Errorf("resolving hash of local reference %s: %w", localName.String(), err)
	}
	slog.InfoContext(ctx, "resolved local reference", "ref", localName.String(), "hash", localRef.Hash().String())

	remoteRef, _, err := rc.remote.ResolveRef(ctx, remoteName)
	switch {
	case errors.Is(err, model.ErrReferenceNotFound):
		remoteRef = plumbing.NewHashReference(remoteName, plumbing.ZeroHash) // hash irrelevant, later we use the local hash
	case err != nil:
		// model.ErrUnsupportedReferenceType, and other errs, are propagated
		return refPair{}, fmt.Errorf("resolving remote reference: %w", err)
	default:
		slog.InfoContext(ctx, "resolved remote reference", "ref", localName.String(), "hash", remoteRef.Hash().String())
	}

	rp, err = rc.compare(force, localRef, remoteRef)
	if err != nil {
		return refPair{}, fmt.Errorf("comparing local and remote refs: %w", err)
	}
	rc.refs[remoteName] = rp

	return rp, nil
}

func (rc *refCompare) compare(force bool, localRef, remoteRef *plumbing.Reference) (refPair, error) {
	rp := refPair{
		local:  localRef,
		remote: remoteRef,
		status: statusUpdateRef,
	}

	if force {
		rp.status |= statusForce
	}

	// empty local indicates ref deletion
	if localRef == nil {
		rp.status |= statusDelete
	} else {
		localCommit, err := rc.local.CommitObject(localRef.Hash())
		if err != nil {
			return refPair{}, fmt.Errorf("resolving commit object %s from hash for local ref %s: %w", localRef.Hash().String(), localRef.Name().String(), err)
		}

		layer, err := rc.remote.CommitExists(rc.local, localCommit)
		if err != nil {
			return refPair{}, fmt.Errorf("resolving existence of commit %s in remote: %w", localCommit, err)
		}
		if layer.String() != "" {
			rp.layer = layer
		} else {
			rp.status |= statusAddCommit
		}

		if remoteRef.Hash().IsZero() {
			rp.status |= statusUpdateRef
		} else {
			remoteCommit, err := rc.local.CommitObject(remoteRef.Hash())
			if err != nil {
				return refPair{}, fmt.Errorf("resolving commit object from hash for remote ref: %w", err)
			}

			isAncestor, err := remoteCommit.IsAncestor(localCommit)
			if err != nil {
				return refPair{}, fmt.Errorf("resolving remote commit ancestor status of local: %w", err)
			}
			if isAncestor {
				rp.status |= statusUpdateRef
			} else if !force {
				return refPair{}, fmt.Errorf("remote reference %s update is not a fast forward of local ref %s", remoteRef.Name().String(), localRef.Name().String())
			}
		}
	}

	return rp, nil
}
