package main

import (
	"context"
	"dagger/gnoci/internal/dagger"
	"errors"
	"fmt"
	"math/rand"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const packDir = ".git/objects/pack"

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
func (e *Eval) ValidateRefs(ctx context.Context,
	// expected list of '<commit> SP <ref>' (output of git show-ref)
	expectedCommitRefs []string,
	// evaluated git repository
	result *dagger.Directory,
) error {
	expectedResolver := make(map[string]string, len(expectedCommitRefs))
	for _, commitRef := range expectedCommitRefs {
		commit, ref, err := splitRefPair(commitRef)
		if err != nil {
			return fmt.Errorf("splitting expected ref pair: %w", err)
		}
		expectedResolver[ref] = commit
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

		expectedCommit, ok := expectedResolver[ref]
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

func (e *Eval) ValidatePacks(ctx context.Context,
	// expected commit tips
	expectedCommits []string,
	// evaluated git repository
	result *dagger.Directory,
) error {
	expectedResolver := make(map[string]bool, len(expectedCommits))
	for _, commit := range expectedCommits {
		expectedResolver[commit] = false
	}

	packs, err := result.Directory(packDir).
		Filter(dagger.DirectoryFilterOpts{Include: []string{"*.pack"}}).
		Entries(ctx)
	if err != nil {
		return fmt.Errorf("filtering packfiles: %w", err)
	}

	var errs []error
	for _, pack := range packs {
		out, err := ctrWithGit().
			With(withGitConfig()).
			WithDirectory(srcDir, result).
			WithWorkdir(srcDir).
			WithExec([]string{"git", "verify-pack", "-v", filepath.Join(packDir, pack)}).
			Stdout(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("verifying packfile %s: %w", pack, err))
			continue
		}

		// example out:
		// 01a6dfffa2aec52e57f4961cf7de8bb01f6f2e0d blob   36 50 112
		// 0e3c37a3474b43457d77674cf048c7e3be742bd3 tree   73 88 401
		// b49cb06c3c94f5d791a3aa57525ba94c1d180193 commit 174 125 802
		for _, line := range strings.Split(out, "\n") {
			if strings.Contains(line, "commit") {
				fields := strings.Fields(line)
				if len(fields) != 5 {
					return fmt.Errorf("unexpected verify-pack output line: %s", line)
				}
				commit := fields[0]
				_, ok := expectedResolver[commit]
				if ok {
					expectedResolver[commit] = true
				}
			}
		}
	}

	for commit, found := range expectedResolver {
		if !found {
			errs = append(errs, fmt.Errorf("missing expected commit %s", commit))
		}
	}

	return errors.Join(errs...)
}

// splitRefPairs splits a set of 'commit SP ref'
// pairs, returning the commits and references respectively.
func splitRefPairs(commitRefs []string) ([]string, []string, error) {
	commits := make([]string, 0, len(commitRefs))
	refs := make([]string, 0, len(commitRefs))
	for _, pair := range commitRefs {
		commit, ref, err := splitRefPair(pair)
		if err != nil {
			return nil, nil, fmt.Errorf("removing commit from reference pair: %w", err)
		}
		commits = append(commits, commit)
		refs = append(refs, ref)
	}
	return commits, refs, nil
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

// git verify-pack .git/objects/pack/pack-b66b57cb487a7e255ffe8bf51d2121e754b03630.pack -v
// eef5c05d9e550943cf0b68cef3596527259280e6 blob   36 50 12
// 260377dc2980a649a9f6208d2ffb002ec6d53efb blob   36 50 62
// 01a6dfffa2aec52e57f4961cf7de8bb01f6f2e0d blob   36 50 112
// 5968665a871eb5413468e204c664594c7b069a5a tree   219 211 162
// 81b6c0c5fe40666a7ae0673578f816ea3007703c tree   12 28 373 1 5968665a871eb5413468e204c664594c7b069a5a
// 0e3c37a3474b43457d77674cf048c7e3be742bd3 tree   73 88 401
// 1d424e20e318a5d34c086e5ef39b1c614b816de0 commit 222 156 489
// 7d206d570b7e697e59594fb251b08d4d444624cc commit 222 157 645
// b49cb06c3c94f5d791a3aa57525ba94c1d180193 commit 174 125 802
// non delta: 8 objects
// chain length = 1: 1 object
// .git/objects/pack/pack-b66b57cb487a7e255ffe8bf51d2121e754b03630.pack: ok
