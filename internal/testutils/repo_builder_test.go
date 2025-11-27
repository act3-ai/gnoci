// Package testutils provides utility functions for building testdata.
package testutils

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
)

func TestNewRepoBuilder(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		dir := t.TempDir()
		rb, err := NewRepoBuilder(dir)
		assert.NoError(t, err)
		assert.NotNil(t, rb)
		assert.NotNil(t, rb.Repo())

		_, statErr := os.Stat(filepath.Join(dir, ".git"))
		assert.NoError(t, statErr)
	})

	t.Run("Bad Permissions", func(t *testing.T) {
		testPath := filepath.Join(t.TempDir(), "test")
		err := os.Mkdir(testPath, 0222)
		assert.NoError(t, err)

		rb, err := NewRepoBuilder(testPath)
		assert.ErrorIs(t, err, os.ErrPermission)
		assert.Nil(t, rb)
	})

	// [git.PlainInit] succeeds if dir dne
}

func TestRepoBuilder_Repo(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		dir := t.TempDir()
		rb, err := NewRepoBuilder(dir)
		assert.NoError(t, err)
		assert.NotNil(t, rb)
		assert.NotNil(t, rb.Repo())

		_, statErr := os.Stat(filepath.Join(dir, ".git"))
		assert.NoError(t, statErr)

		repo := rb.Repo()
		assert.Equal(t, rb.repo, repo)
	})
}

func TestRepoBuilder_CreateRandomCommit(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		rb, err := NewRepoBuilder(t.TempDir())
		assert.NoError(t, err)

		hash, err := rb.CreateRandomCommit(256)
		assert.NoError(t, err)
		assert.NotEqual(t, plumbing.ZeroHash, hash)

		// validate commit exists
		_, err = rb.repo.CommitObject(hash)
		assert.NoError(t, err)

		// validate worktree status
		wt, err := rb.repo.Worktree()
		assert.NoError(t, err)

		status, err := wt.Status()
		assert.NoError(t, err)
		assert.True(t, status.IsClean())
	})

	t.Run("Invalid Size", func(t *testing.T) {
		rb, err := NewRepoBuilder(t.TempDir())
		assert.NoError(t, err)

		hash, err := rb.CreateRandomCommit(-1)
		assert.Error(t, err)
		assert.Equal(t, plumbing.ZeroHash, hash)

		// validate worktree status
		wt, err := rb.repo.Worktree()
		assert.NoError(t, err)

		status, err := wt.Status()
		assert.NoError(t, err)
		assert.True(t, status.IsClean())
	})
}

func TestRepoBuilder_CreateBranch(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		b, err := NewRepoBuilder(t.TempDir())
		assert.NoError(t, err)

		commitHash, err := b.CreateRandomCommit(256)
		assert.NoError(t, err)

		branchName := "feature/test"

		ref, err := b.CreateBranch(branchName, commitHash)
		assert.NoError(t, err)
		assert.Equal(t, plumbing.NewBranchReferenceName(branchName), ref.Name())
	})

	// TODO: Why does go-git not check that commitHash is real?
}

func TestRepoBuilder_DeleteBranch(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		b, err := NewRepoBuilder(t.TempDir())
		assert.NoError(t, err)

		commitHash, err := b.CreateRandomCommit(256)
		assert.NoError(t, err)

		branchName := "feature/test"

		ref, err := b.CreateBranch(branchName, commitHash)
		assert.NoError(t, err)
		assert.Equal(t, plumbing.NewBranchReferenceName(branchName), ref.Name())

		err = b.DeleteBranch(branchName)
		assert.NoError(t, err)

		_, err = b.Repo().Reference(plumbing.NewBranchReferenceName(branchName), false)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, plumbing.ErrReferenceNotFound))
	})
}

func TestRepoBuilder_CreateTag(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		b, err := NewRepoBuilder(t.TempDir())
		assert.NoError(t, err)

		commitHash, err := b.CreateRandomCommit(256)
		assert.NoError(t, err)

		tagName := "v1.0.0"

		ref, err := b.CreateTag(tagName, commitHash)
		assert.NoError(t, err)
		assert.Equal(t, plumbing.NewTagReferenceName(tagName), ref.Name())
		assert.Equal(t, commitHash, ref.Hash())
	})
}

func TestRepoBuilder_DeleteTag(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		b, err := NewRepoBuilder(t.TempDir())
		assert.NoError(t, err)

		commitHash, err := b.CreateRandomCommit(256)
		assert.NoError(t, err)

		tagName := "v1.0.0"

		ref, err := b.CreateTag(tagName, commitHash)
		assert.NoError(t, err)
		assert.Equal(t, plumbing.NewTagReferenceName(tagName), ref.Name())
		assert.Equal(t, commitHash, ref.Hash())

		err = b.DeleteTag(tagName)
		assert.NoError(t, err)
	})
}
