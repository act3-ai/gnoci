// Package refcomp provides utilities for comparing local and remote git references.
package refcomp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/opencontainers/go-digest"

	"github.com/act3-ai/gnoci/internal/git"
	"github.com/act3-ai/gnoci/internal/model"
)

// Status represents the result of a reference comparison.
type Status uint8

const (
	// StatusDelete indicates a ref should be removed from the remote.
	StatusDelete Status = 1 << iota
	// StatusUpdateRef indicates a StatusUpdateRef should be updated in the remote.
	StatusUpdateRef
	// StatusAddCommit indicates the ref's commit object should be added to the remote.
	StatusAddCommit
	// StatusForce indicates a StatusForce update should be performed.
	StatusForce
)

// RefComparer provides utilities for comparing local and remote references.
type RefComparer interface {
	// Compare resolves the status of a remote reference.
	Compare(ctx context.Context, force bool, localName plumbing.ReferenceName, remoteName plumbing.ReferenceName) (RefPair, error)
	// GetStatus returns the status of a remote reference.
	// GetStatus(remoteName plumbing.ReferenceName) (status, bool)
}

// refCompareCached implements [RefComparer].
type refCompareCached struct {
	local  git.Repository
	remote model.Modeler

	refs map[plumbing.ReferenceName]RefPair // key is remote ref name
}

// RefPair represents the state of a local and remote reference and their differences.
type RefPair struct {
	// Local is the local Git reference resolved from a [plumbing.ReferenceName].
	Local *plumbing.Reference
	// Remote is the remote Git reference resolved from a [plumbing.ReferenceName].
	Remote *plumbing.Reference
	// Status is the result of comparing [RefPair.Local] to [RefPair.Remote].
	Status
	// Layer is the remote OCI layer that contains the commit referenced by [RefPair.Local], if available.
	Layer digest.Digest // only populated if (status ^ statusAddCommit)
}

// NewCachedRefComparer initializes a RefComparer that caches all ref comparisons.
func NewCachedRefComparer(local git.Repository, remote model.Modeler) RefComparer {
	return &refCompareCached{
		local:  local,
		remote: remote,
		refs:   make(map[plumbing.ReferenceName]RefPair, 0),
	}
}

// Compare compares a local and remote refence by name.
func (rc *refCompareCached) Compare(ctx context.Context, force bool, localName, remoteName plumbing.ReferenceName) (RefPair, error) {
	rp, ok := rc.refs[remoteName]
	if ok {
		return rp, nil
	}

	var localRef *plumbing.Reference
	if localName != "" {
		var err error
		localRef, err = rc.local.Reference(localName, true)
		if err != nil {
			return RefPair{}, fmt.Errorf("resolving hash of local reference %s: %w", localName.String(), err)
		}
		slog.InfoContext(ctx, "resolved local reference", "ref", localName.String(), "hash", localRef.Hash().String())
	}

	remoteRef, _, err := rc.remote.ResolveRef(ctx, remoteName)
	switch {
	case errors.Is(err, model.ErrReferenceNotFound):
		remoteRef = plumbing.NewHashReference(remoteName, plumbing.ZeroHash) // hash irrelevant, later we use the local hash
	case err != nil:
		// model.ErrUnsupportedReferenceType, and other errs, are propagated
		return RefPair{}, fmt.Errorf("resolving remote reference: %w", err)
	default:
		slog.InfoContext(ctx, "resolved remote reference", "ref", remoteName.String(), "hash", remoteRef.Hash().String())
	}

	rp, err = rc.compare(force, localRef, remoteRef)
	if err != nil {
		return RefPair{}, fmt.Errorf("comparing local and remote refs: %w", err)
	}
	rc.refs[remoteName] = rp

	return rp, nil
}

func (rc *refCompareCached) compare(force bool, localRef, remoteRef *plumbing.Reference) (RefPair, error) {
	rp := RefPair{
		Local:  localRef,
		Remote: remoteRef,
		Status: StatusUpdateRef,
	}

	if force {
		rp.Status |= StatusForce
	}

	// empty local indicates ref deletion
	if localRef == nil {
		rp.Status |= StatusDelete
	} else {
		localCommit, err := rc.local.CommitObject(localRef.Hash())
		if err != nil {
			return RefPair{}, fmt.Errorf("resolving commit object %s from hash for local ref %s: %w", localRef.Hash().String(), localRef.Name().String(), err)
		}

		layer, err := rc.remote.CommitExists(rc.local, localCommit)
		if err != nil {
			return RefPair{}, fmt.Errorf("resolving existence of commit %s in remote: %w", localCommit, err)
		}
		if layer.String() != "" {
			rp.Layer = layer
		} else {
			rp.Status |= StatusAddCommit
		}

		if remoteRef.Hash().IsZero() {
			rp.Status |= StatusUpdateRef
		} else {
			remoteCommit, err := rc.local.CommitObject(remoteRef.Hash())
			if err != nil {
				return RefPair{}, fmt.Errorf("resolving commit object from hash for remote ref: %w", err)
			}

			isAncestor, err := remoteCommit.IsAncestor(localCommit)
			if err != nil {
				return RefPair{}, fmt.Errorf("resolving remote commit ancestor status of local: %w", err)
			}
			if isAncestor {
				rp.Status |= StatusUpdateRef
			} else if !force {
				return RefPair{}, fmt.Errorf("remote reference %s update is not a fast forward of local ref %s", remoteRef.Name().String(), localRef.Name().String())
			}
		}
	}

	return rp, nil
}
