package main

import "dagger/gnoci/internal/dagger"

// A collection of test repositories.
func (t *Test) Repos() *Repos {
	return &Repos{
		Test: t,
	}
}

// Repos organizes test repositories.
type Repos struct {
	*Test
}

// All repos return a *dagger.Directory, although *dagger.GitRef may seem to be preferred, we want to make sure we don't restrict the repository to a single branch.

// Simple repo contains a few commits on main.
func (r *Repos) Simple() *dagger.Directory {
	return r.Test.RepoBuilder(nil).
		NewCommit().
		NewCommit().
		NewCommit().
		GitDir()
}
