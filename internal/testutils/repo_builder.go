// Package testutils provides utility functions for building testdata.
package testutils

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// RepoBuilder provides methods for building a git repository.
type RepoBuilder struct {
	repo *git.Repository
}

// NewRepoBuilder initializes a RepoBuilder.
func NewRepoBuilder(dir string) (*RepoBuilder, error) {
	// will create if dir dne
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		return nil, fmt.Errorf("initializing plain git repository: %w", err)
	}

	return &RepoBuilder{repo: repo}, nil
}

// Repo returns the underlying git repository.
func (b *RepoBuilder) Repo() *git.Repository {
	return b.repo
}

// CreateRandomCommit creates a commit with random file data of given size.
func (b *RepoBuilder) CreateRandomCommit(size int64) (plumbing.Hash, error) {
	if size < 0 {
		return plumbing.ZeroHash, fmt.Errorf("invalid file size %d expected > 0", size)
	}
	wt, err := b.repo.Worktree()
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("getting repository worktree: %w", err)
	}

	filename := fmt.Sprintf("file_%s.txt", rand.Text())
	f, err := wt.Filesystem.OpenFile(wt.Filesystem.Join(filename), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, io.LimitReader(rand.Reader, size)); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("writing random data to file: %w", err)
	}
	if err := f.Close(); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("closing file: %w", err)
	}

	if _, err := wt.Add(filename); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("adding file to worktree: %w", err)
	}

	hash, err := wt.Commit("test commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("committing file: %w", err)
	}

	return hash, nil
}

// CreateBranch creates a new branch.
func (b *RepoBuilder) CreateBranch(branchName string, commit plumbing.Hash) (*plumbing.Reference, error) {
	ref := plumbing.NewHashReference(plumbing.NewBranchReferenceName(branchName), commit)
	if err := b.repo.Storer.SetReference(ref); err != nil {
		return nil, fmt.Errorf("creating branch reference: %w", err)
	}
	return ref, nil
}

// DeleteBranch deletes a branch.
func (b *RepoBuilder) DeleteBranch(branchName string) error {
	refName := plumbing.NewBranchReferenceName(branchName)
	if err := b.repo.Storer.RemoveReference(refName); err != nil {
		return fmt.Errorf("deleting branch reference: %w", err)
	}
	return nil
}

// CreateTag creates a lightweight tag.
func (b *RepoBuilder) CreateTag(tagName string, commit plumbing.Hash) (*plumbing.Reference, error) {
	ref := plumbing.NewHashReference(plumbing.NewTagReferenceName(tagName), commit)
	if err := b.repo.Storer.SetReference(ref); err != nil {
		return nil, fmt.Errorf("creating tag reference: %w", err)
	}
	return ref, nil
}

// DeleteTag deletes a tag.
func (b *RepoBuilder) DeleteTag(tagName string) error {
	refName := plumbing.NewTagReferenceName(tagName)
	if err := b.repo.Storer.RemoveReference(refName); err != nil {
		return fmt.Errorf("deleting tag reference: %w", err)
	}
	return nil
}
