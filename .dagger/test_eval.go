package main

import (
	"context"
	"dagger/gnoci/internal/dagger"
	"errors"
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
func (e *Eval) Refs(ctx context.Context,
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

	return strings.Split(strings.TrimSpace(out), "\n"), nil
}

// Heads returns a slice of all head references (branches) and their commits, as "<commit> SP <reference>".
func (e *Eval) Heads(ctx context.Context,
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
func (e *Eval) Tags(ctx context.Context,
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

// ValidateResults ensures the expected references and their commits ae in the result git repository.
func (e *Eval) ValidateResult(ctx context.Context,
	// expected list of '<commit> SP <ref>' (output of git show-ref)
	expectedCommitRefs []string,
	// evaluated git repository
	result *dagger.Directory,
) error {
	srcResolver := make(map[string]string, len(expectedCommitRefs))
	for _, commitRef := range expectedCommitRefs {
		commit, ref, err := splitRefPair(commitRef)
		if err != nil {
			return fmt.Errorf("splitting expected ref pair: %w", err)
		}
		srcResolver[ref] = commit
	}

	resultRefs, err := e.Refs(ctx, result)
	if err != nil {
		return fmt.Errorf("getting result references: %w", err)
	}

	// check expected is valid, if not expected disregard
	var errs []error
	for _, commitRef := range resultRefs {
		commit, ref, err := splitRefPair(commitRef)
		if err != nil {
			return fmt.Errorf("splitting result ref pair: %w", err)
		}

		expectedCommit, ok := srcResolver[ref]
		switch {
		case !ok:
			continue
		case expectedCommit != commit:
			errs = append(errs, fmt.Errorf("evaluating reference %s: expected commit %s, got %s", ref, expectedCommit, commit))
		default:
			// success!
		}
	}

	return errors.Join(errs...)
}

// refsFromRefPair removes the commit portion of a 'commit SP ref'
// pair, returning a slice of only references.
func refsFromRefPair(commitRefs []string) ([]string, error) {
	result := make([]string, 0, len(commitRefs))
	for _, pair := range commitRefs {
		_, ref, err := splitRefPair(pair)
		if err != nil {
			return nil, fmt.Errorf("removing commit from reference pair: %w", err)
		}
		result = append(result, ref)
	}
	return result, nil
}

// splitRefPair returns the commit and reference that make up
// a reference pair 'commit SP ref'.
func splitRefPair(pair string) (string, string, error) {
	fields := strings.Fields(pair)
	if len(fields) != 2 {
		return "", "", fmt.Errorf("invalid commit reference pair %s, expected 'commit SP ref'", pair)
	}
	commit, ref := fields[0], fields[1]
	return commit, ref, nil
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
