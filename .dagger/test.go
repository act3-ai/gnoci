package main

import (
	"context"
	"dagger/gnoci/internal/dagger"
	"fmt"
	"path/filepath"

	"github.com/sourcegraph/conc/pool"
)

// Run tests.
func (g *Gnoci) Test() *Test {
	return &Test{
		Gnoci: g,
	}
}

// Test organizes testing operations.
type Test struct {
	*Gnoci
}

// Run all tests.
func (t *Test) All(ctx context.Context) (string, error) {
	unitResults, unitErr := t.Unit(ctx)

	// TODO: add functional  tests here

	out := "Unit Test Results:\n" + unitResults

	return out, unitErr // TODO: use errors.Join when functional tests are added
}

// Run unit tests.
func (t *Test) Unit(ctx context.Context) (string, error) {
	return dag.Go(). //nolint:wrapcheck
				WithSource(t.Source).
				Container().
				WithExec([]string{"go", "test", "./..."}).
				Stdout(ctx)
}

const (
	coverageFile        = "coverage.out"
	coverageTreemapFile = "coverage-treemap.svg"
)

// Coverage generates a code coverage file.
func (t *Test) Coverage() *dagger.File {
	// TODO: filter for better caching, had issues with embed.go
	return t.goWithSource(t.Source).
		WithExec([]string{"go", "test", "./...", "-coverprofile", coverageFile}). // TODO: other options?
		Container().
		File(coverageFile)
}

// CoverageTreeMap builds a visual aid for viewing code coverage.
func (t *Test) CoverageTreeMap(ctx context.Context,
	// coverage is the output file of go test with coverage.
	coverage *dagger.File,
) *dagger.File {
	src := t.Source.WithFile(coverageFile, coverage) // TODO: filter for better caching, had issues with embed.go

	svg, _ := t.goWithSource(src).
		WithExec([]string{"go", "install", "github.com/nikolaydubina/go-cover-treemap@latest"}).
		Container().
		WithExec([]string{"./bin/go-cover-treemap", "-coverprofile", coverageFile}).
		Stdout(ctx)

	return dag.File(coverageTreemapFile, svg)
}

func (t *Test) PushFetch(ctx context.Context,
	// Git reference to test repository
	gitRef *dagger.GitRef,
) (string, error) {
	const srcDir = "test-src"
	src := gitRef.Tree()

	// get container with git-remote-oci and git-lfs-remote-oci
	ctr, err := t.containerWithHelpers(ctx)
	if err != nil {
		return "", fmt.Errorf("creating test container: %w", err)
	}

	// add test source
	ctr = ctr.WithDirectory(srcDir, src).
		WithWorkdir(srcDir)

	// configure git

	// configure git-lfs

	// connect to registry

	// push

	// pull (in second dir or another container?)

	// get metadata

	// return metadata on stdout

	return "", fmt.Errorf("not implemented")
}

// containerWithHelpers creates a container with git-remote-oci and git-lfs-remote-oci.
func (t *Test) containerWithHelpers(ctx context.Context) (*dagger.Container, error) {
	platform := dagger.Platform("linux/amd64")

	var gitHelper, gitLFSHelper *dagger.File
	var gitHelperName, gitLFSHelperName string
	p := pool.New().WithContext(ctx)
	p.Go(func(ctx context.Context) error {
		gitHelper = t.BuildGit(ctx, "test-dev", platform)

		var err error
		gitHelperName, err = gitHelper.Name(ctx)
		if err != nil {
			return fmt.Errorf("determining name of git helper exec: %w", err)
		}
		return nil
	})

	p.Go(func(ctx context.Context) error {
		gitLFSHelper = t.BuildGitLFS(ctx, "test-dev", platform)

		var err error
		gitLFSHelperName, err = gitLFSHelper.Name(ctx)
		if err != nil {
			return fmt.Errorf("determining name of git-lfs helper exec: %w", err)
		}
		return nil
	})

	_ = p.Wait() // throw away err, as we can't get one

	return dag.Alpine(dagger.AlpineOpts{
		Packages: []string{"git", "git-lfs"},
	}).
		Container().
		WithFile(filepath.Join("usr", "local", "bin", gitHelperName), gitHelper).
		WithFile(filepath.Join("usr", "local", "bin", gitLFSHelperName), gitLFSHelper), nil
}

// func (t *Test) TestCtr(ctx context.Context) (*dagger.Container, error) {
// 	ctr, err := t.containerWithHelpers(ctx)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create container: %w", err)
// 	}

// 	ctr = ctr.Terminal()

// 	ctr.WithExec([]string{"git", "--version"})

// 	return ctr, nil
// }
