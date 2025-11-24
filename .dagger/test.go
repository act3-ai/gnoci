package main

import (
	"context"
	"dagger/gnoci/internal/dagger"
	"path/filepath"
	"strings"
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
	coverageFile         = "coverage.out"
	coverageFilteredFile = "coverage.filtered"
	coverageTreemapFile  = "coverage-treemap.svg"
)

// Coverage generates a code coverage file.
func (t *Test) Coverage() *dagger.File {
	// TODO: filter for better caching, had issues with embed.go
	return t.goWithSource(t.Source).
		WithExec([]string{"go", "test", "./...", "-count=1", "-coverprofile", coverageFile, "-coverpkg=./..."}).WithExec([]string{"./filter-coverage.sh", "<", coverageFile, ">", coverageFilteredFile}).
		Container().
		File(coverageFilteredFile)
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
		WithExec([]string{"./bin/go-cover-treemap", "-coverprofile", coverageFilteredFile}).
		Stdout(ctx)

	return dag.File(coverageTreemapFile, svg)
}

// Push pushes a git repository to an OCI registry.
func (t *Test) Push(ctx context.Context,
	// Git reference to test repository
	gitRef *dagger.GitRef,
) (string, error) {
	// start registry
	regService := registryService()
	regService, err := regService.Start(ctx)
	if err != nil {
		return "", err
	}
	defer regService.Stop(ctx)

	regEndpoint, err := regService.Endpoint(ctx, dagger.ServiceEndpointOpts{Scheme: "http"})
	if err != nil {
		return "", err
	}
	regHost := strings.TrimPrefix(regEndpoint, "http://")

	const srcDir = "src"
	return t.containerWithHelpers(ctx).
		WithDirectory(srcDir, gitRef.Tree(dagger.GitRefTreeOpts{Depth: -1})).
		WithWorkdir(srcDir).
		WithServiceBinding("registry", regService).
		With(configureLFSOCIFunc(regHost)).
		WithExec([]string{"git", "push", "oci://" + regHost + "/repo/test:sync", "--all"}).
		Stdout(ctx)

	// configure git

	// configure git-lfs

	// connect to registry

	// push

	// get metadata

	// return metadata on stdout

	// return "", fmt.Errorf("not implemented")
}

// containerWithHelpers creates a container with the dependencies necessary to test
// git-remote-oci and git-lfs-remote-oci.
func (t *Test) containerWithHelpers(ctx context.Context) *dagger.Container {
	platform := dagger.Platform("linux/amd64")
	version := "test-dev"

	return dag.Alpine(dagger.AlpineOpts{Packages: []string{"git", "git-lfs"}}).
		Container().
		WithFile(filepath.Join("usr", "local", "bin", gitExecName), t.BuildGit(ctx, version, platform)).
		WithFile(filepath.Join("usr", "local", "bin", gitLFSExecName), t.BuildGitLFS(ctx, version, platform)).
		WithExec([]string{"git", "config", "--global", "user.name", "dev-test"}).
		WithExec([]string{"git", "config", "--global", "user.email", "devtest@example.com"}).
		WithExec([]string{"git", "config", "--global", "init.defaultbranch", "main"})
}

// configureLFSOCIFunc configures a git repository with an OCI remote.
func configureLFSOCIFunc(ociRemote string) func(c *dagger.Container) *dagger.Container {
	return func(c *dagger.Container) *dagger.Container {
		return c.WithExec([]string{"git", "config", "lfs.standalonetransferagent", "oci"}).
			WithExec([]string{"git", "config", "lfs.customtransfer.oci.path", gitLFSExecName}).
			WithExec([]string{"git", "config", "lfs.customtransfer.oci.batch", "false"}).
			WithExec([]string{"git", "config", "lfs.customtransfer.oci.concurrent", "false"}).
			WithExec([]string{"git", "config", "lfs.url", "oci://" + ociRemote})
	}
}

// https://hub.docker.com/_/registry
const (
	imageRegistry = "docker.io/library/registry:3.0"
	registryPort  = 5000
)

func registryService() *dagger.Service {
	return dag.Container().
		From(imageRegistry).
		WithExposedPort(registryPort).
		AsService()
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
