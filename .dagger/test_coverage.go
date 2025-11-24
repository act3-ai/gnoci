package main

import (
	"context"
	"dagger/gnoci/internal/dagger"
	"fmt"
)

const (
	coverageFile         = "coverage.out"
	coverageFilteredFile = "coverage.filtered"
	coverageTreemapFile  = "coverage-treemap.svg"
	coverageBadgeFile    = "coverage.svg"

	coverageFilterScript = "filter-coverage.sh"
	coverageValueScript  = "coverage-value.sh"

	// code coverage values for badge color
	redThreshold    = "50"
	yellowThreshold = "80"
	greenThreshold  = "100"
)

// CoverageDocs generates all code coverage documentation.
func (t *Test) CoverageDocs(ctx context.Context) *dagger.Directory {
	coverage := t.Coverage()

	return dag.Directory().
		WithFile(coverageTreemapFile, t.CoverageTreeMap(ctx, coverage)).
		WithFile(coverageBadgeFile, t.CoverageBadge(ctx, coverage))
}

// Coverage generates a code coverage file.
func (t *Test) Coverage() *dagger.File {
	// TODO: filter for better caching, had issues with embed.go
	return t.goWithSource(t.Source).
		WithExec([]string{"go", "test", "-count=1", "-timeout=30s", "./...", "-coverprofile", coverageFile, "-coverpkg=./..."}).
		Container().
		WithExec([]string{"./" + coverageFilterScript}, dagger.ContainerWithExecOpts{
			RedirectStdin:  coverageFile,
			RedirectStdout: coverageFilteredFile,
		}).
		File(coverageFilteredFile)
}

// CoverageTreeMap builds a visual aid for viewing code coverage.
func (t *Test) CoverageTreeMap(ctx context.Context,
	// coverage is the output file of go test with coverage.
	coverage *dagger.File,
) *dagger.File {
	src := t.Source.WithFile(coverageFilteredFile, coverage) // TODO: filter for better caching, had issues with embed.go

	svg, _ := t.goWithSource(src).
		WithExec([]string{"go", "install", "github.com/nikolaydubina/go-cover-treemap@latest"}).
		Container().
		WithExec([]string{"./bin/go-cover-treemap", "-coverprofile", coverageFilteredFile}).
		Stdout(ctx)

	return dag.File(coverageTreemapFile, svg)
}

func (t *Test) CoverageBadge(ctx context.Context,
	// coverage is the output file of go test with coverage.
	coverage *dagger.File,
) *dagger.File {
	coverageValue, _ := t.goWithSource(t.Source).
		Container().
		WithFile(coverageValueScript, t.Source.File(coverageValueScript)).
		WithFile(coverageFilteredFile, t.Coverage()).
		WithExec([]string{"./" + coverageValueScript, coverageFilteredFile},
			dagger.ContainerWithExecOpts{
				RedirectStdin: coverageFilteredFile,
			}).
		Stdout(ctx)

	return dag.Python().
		Container().
		WithFile(coverageFilteredFile, coverage).
		WithExec([]string{"pip", "install", "anybadge"}).
		WithExec([]string{"anybadge",
			"--label", "coverage",
			"--value", coverageValue,
			"--file", coverageBadgeFile,
			fmt.Sprintf("%s=red", redThreshold),
			fmt.Sprintf("%s=yellow", yellowThreshold),
			fmt.Sprintf("%s=green", greenThreshold),
		}).
		File(coverageBadgeFile)
}
