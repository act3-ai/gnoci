package main

import (
	"dagger/gnoci/internal/dagger"
	"fmt"

	"github.com/google/uuid"
)

const srcDir = "src"

// Git repository builder.
func (t *Test) RepoBuilder(
	// base git repository
	// +optional
	base *dagger.Directory,
) *RepoBuilder {
	ctr := dag.Alpine(dagger.AlpineOpts{Packages: []string{"git"}}).
		Container().
		With(withGitConfig()).
		With(func(r *dagger.Container) *dagger.Container {
			if base != nil {
				return r.WithDirectory(srcDir, base)
			}
			return r.WithWorkdir(srcDir).
				WithExec([]string{"git", "init"})
		})

	return &RepoBuilder{
		Test: t,
		ctr:  ctr,
	}
}

// RepoBuilder organizes testing helper utilities.
type RepoBuilder struct {
	*Test

	ctr *dagger.Container
}

// NewCommit adds a commit from HEAD.
func (h *RepoBuilder) NewCommit() *RepoBuilder {
	uuidStr := uuid.NewString()
	filename := fmt.Sprintf("file-%s.txt", uuidStr)

	h.ctr = h.ctr.WithNewFile(filename, uuidStr).
		WithExec([]string{"git", "add", "--all"}).
		WithExec([]string{"git", "commit", "-m", "adding commit"})

	return h
}

// Checkout checks out an existing branch.
func (h *RepoBuilder) Checkout(branchName string) *RepoBuilder {
	h.ctr = h.ctr.WithExec([]string{"git", "checkout", branchName})
	return h
}

// Branch creates a new branch at HEAD, it does not create a new commit.
func (h *RepoBuilder) Branch(branchName string) *RepoBuilder {
	h.ctr = h.ctr.WithExec([]string{"git", "switch", "-c", branchName})
	return h
}

// DeleteBranch removes a branch.
func (h *RepoBuilder) DeleteBranch(branchName string) *RepoBuilder {
	h.ctr = h.ctr.WithExec([]string{"git", "branch", "-D", branchName})
	return h
}

// Tag creates a new tag at HEAD.
func (h *RepoBuilder) Tag(tag string) *RepoBuilder {
	h.ctr = h.ctr.WithExec([]string{"git", "tag", tag})
	return h
}

// DeleteTag deletes a tag.
func (h *RepoBuilder) DeleteTag(tag string) *RepoBuilder {
	h.ctr = h.ctr.WithExec([]string{"git", "tag", "-d", tag})
	return h
}

// GitDir returns the built repository in it's current state.
func (h *RepoBuilder) GitDir() *dagger.Directory {
	return h.ctr.Directory(".")
}
