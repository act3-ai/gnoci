package main

import (
	"context"
	"dagger/gnoci/internal/dagger"
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
