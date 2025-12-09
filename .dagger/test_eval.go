package main

import (
	"context"
	"dagger/gnoci/internal/dagger"
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"time"
)

// A collection of utilities for evaluating test results.
func (t *Test) Eval() *Eval {
	return &Eval{
		Test: t,
	}
}

// Eval organizes test result evaluation.
type Eval struct {
	*Test
}

// Refs returns a slice of all head and tag references and their commits, as "<commit> SP <reference>".
func (t *Eval) Refs(ctx context.Context,
	// repo is a git repository
	repo *dagger.Directory,
) ([]string, error) {
	out, err := ctrWithGit().
		WithDirectory(srcDir, repo).
		WithWorkdir(srcDir).
		WithExec([]string{"git", "show-ref", "--heads", "--tags"}).
		Stdout(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting git head and tag references: %w", err)
	}

	return strings.Split(out, "\n"), nil
}

// Heads returns a slice of all head references (branches) and their commits, as "<commit> SP <reference>".
func (t *Eval) Heads(ctx context.Context,
	// repo is a git repository
	repo *dagger.Directory,
) ([]string, error) {
	out, err := ctrWithGit().
		WithDirectory(srcDir, repo).
		WithWorkdir(srcDir).
		WithExec([]string{"git", "show-ref", "--heads"}).
		Stdout(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting git head references: %w", err)
	}

	return strings.Split(out, "\n"), nil
}

// Tags returns a slice of all tag references and their commits, as "<commit> SP <reference>".
func (t *Eval) Tags(ctx context.Context,
	// repo is a git repository
	repo *dagger.Directory,
) ([]string, error) {
	out, err := ctrWithGit().
		WithDirectory(srcDir, repo).
		WithWorkdir(srcDir).
		WithExec([]string{"git", "show-ref", "--tags"}).
		Stdout(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting git tag references: %w", err)
	}

	return strings.Split(out, "\n"), nil
}

// subset reduces a slice to a random subset.
func subset[T any](full []T, size int) []T {
	if size > len(full) || size < 0 {
		size = len(full) / 2
	}

	clone := slices.Clone(full)

	r := rand.New(rand.NewSource(time.Now().UnixNano())) // TODO: global?
	r.Shuffle(len(clone), func(i, j int) {
		clone[i], clone[j] = clone[j], clone[i]
	})
	return clone[:size]
}
