package main

import (
	"context"
	"dagger/gnoci/internal/dagger"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// Run tests.
func (g *Gnoci) Test(
	// source code directory
	// +defaultPath="/"
	src *dagger.Directory,
) *Test {
	return &Test{
		Gnoci: g,
		Src:   src,
	}
}

// Test organizes testing operations.
type Test struct {
	*Gnoci

	// +private
	Src *dagger.Directory
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
				WithSource(t.Src).
				Container().
				WithExec([]string{"go", "test", "./..."}).
				Stdout(ctx)
}

func (t *Test) Functional(ctx context.Context) error {
	// start registry
	registry := registryService()
	registry, err := registry.Start(ctx)
	if err != nil {
		return fmt.Errorf("starting registry service: %w", err)
	}
	defer registry.Stop(ctx) //nolint:errcheck

	simpleSlug := "test/simple:src"
	simpleRepo := t.Repos().Simple()
	refPairs, err := t.Eval().Refs(ctx, simpleRepo)
	if err != nil {
		return fmt.Errorf("getting commit reference pairs from simple source repository: %w", err)
	}

	_, refs, err := splitRefPairs(refPairs)
	if err != nil {
		return fmt.Errorf("getting refs from commit reference pairs: %w", err)
	}

	_, err = t.Push(ctx, simpleRepo, refs, registry, simpleSlug)
	if err != nil {
		return fmt.Errorf("failed to push repository to OCI: %w", err)
	}

	err = t.FromRemote(ctx, simpleRepo, refPairs, registry, simpleSlug)
	if err != nil {
		return fmt.Errorf("FromRemote: Simple: %w", err)
	}

	return nil
}

// FromRemote assumes a remote exists and tests "fetching" operations, e.g.
// fetch, pull, and clone.
func (t *Test) FromRemote(ctx context.Context,
	// Git repository
	gitRepo *dagger.Directory,
	// git 'commit SP ref' pairs to fetch and pull
	refPairs []string,
	// registry service
	registry *dagger.Service,
	// git OCI remote repository slug
	ociSlug string,
) error {
	commits, refs, err := splitRefPairs(refPairs)
	if err != nil {
		return fmt.Errorf("getting refs from commit reference pairs: %w", err)
	}

	var errs []error
	fetchResult, err := t.Fetch(ctx, registry, ociSlug, refs)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to fetch repository from OCI: %w", err))
	}
	if err := t.Eval().ValidatePacks(ctx, commits, fetchResult); err != nil {
		errs = append(errs, fmt.Errorf("evaluating fetch result: %w", err))
	}

	pullResult, err := t.Pull(ctx, registry, ociSlug, refs)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to pull repository from OCI: %w", err))
	}
	if err := t.Eval().ValidateRefs(ctx, refPairs, pullResult); err != nil {
		errs = append(errs, fmt.Errorf("evaluating pull result: %w", err))
	}

	cloneResult, err := t.Clone(ctx, registry, ociSlug)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to clone repository from OCI: %w", err))
	}
	if err := t.Eval().ValidateRefs(ctx, refPairs, cloneResult); err != nil {
		errs = append(errs, fmt.Errorf("evaluating clone result: %w", err))
	}

	return errors.Join(errs...)
}

// PushClone evaluates a push to then clone from an OCI registry.
// func (t *Test) PushClone(ctx context.Context,
// 	// Git repository
// 	gitRepo *dagger.Directory,
// 	// git 'commit SP ref' pairs to push
// 	refPairs []string,
// 	// registry service
// 	registry *dagger.Service,
// 	// git OCI remote repository slug
// 	ociSlug string,
// ) error {
// 	refs, err := refsFromRefPair(refPairs)
// 	if err != nil {
// 		return fmt.Errorf("getting refs from commit reference pairs: %w", err)
// 	}

// 	_, err = t.Push(ctx, gitRepo, refs, registry, ociSlug)
// 	if err != nil {
// 		return fmt.Errorf("failed to push repository to OCI: %w", err)
// 	}

// 	result, err := t.Clone(ctx, registry, ociSlug)
// 	if err != nil {
// 		return fmt.Errorf("failed to clone repository from OCI: %w", err)
// 	}

// 	if err := t.Eval().ValidateRefs(ctx, refPairs, result); err != nil {
// 		return fmt.Errorf("evaluating result: %w", err)
// 	}

// 	return nil
// }

// Push pushes a git repository to an OCI registry.
//
//nolint:wrapcheck
func (t *Test) Push(ctx context.Context,
	// Git repository
	gitRepo *dagger.Directory,
	// git references to push
	refs []string,
	// registry service
	registry *dagger.Service,
	// git OCI remote repository slug
	ociSlug string,
) (string, error) {
	regEndpoint, err := registry.Endpoint(ctx, dagger.ServiceEndpointOpts{Scheme: "http"})
	if err != nil {
		return "", err
	}
	regHost := strings.TrimPrefix(regEndpoint, "http://")

	const srcDir = "src"
	return ctrWithGit().
		With(t.withGitRemoteHelper(ctx)).
		With(withGitConfig()).
		With(withGnociConfig(regHost)).
		WithDirectory(srcDir, gitRepo).
		WithWorkdir(srcDir).
		WithServiceBinding("registry", registry).
		WithExec(append([]string{"git", "push", ociRef(regHost, ociSlug)}, refs...)).
		CombinedOutput(ctx)
}

// Fetch fetches a git repository from an OCI registry.
//
//nolint:wrapcheck
func (t *Test) Fetch(ctx context.Context,
	// registry service
	registry *dagger.Service,
	// git OCI remote repository slug
	ociSlug string,
	// references to fetch
	refs []string,
) (*dagger.Directory, error) {
	regEndpoint, err := registry.Endpoint(ctx, dagger.ServiceEndpointOpts{Scheme: "http"})
	if err != nil {
		return nil, err
	}
	regHost := strings.TrimPrefix(regEndpoint, "http://")

	return ctrWithGit().
		With(t.withGitRemoteHelper(ctx)).
		With(withGitConfig()).
		With(withGnociConfig(regHost)).
		WithServiceBinding("registry", registry).
		WithWorkdir(srcDir).
		WithExec([]string{"git", "init"}).
		WithExec(append([]string{"git", "fetch", ociRef(regHost, ociSlug)}, refs...)).
		Directory(".", dagger.ContainerDirectoryOpts{Expand: true}), nil
}

// Pull pulls a git repository from an OCI registry.
//
//nolint:wrapcheck
func (t *Test) Pull(ctx context.Context,
	// registry service
	registry *dagger.Service,
	// git OCI remote repository slug
	ociSlug string,
	// references to pull
	refs []string,
) (*dagger.Directory, error) {
	regEndpoint, err := registry.Endpoint(ctx, dagger.ServiceEndpointOpts{Scheme: "http"})
	if err != nil {
		return nil, err
	}
	regHost := strings.TrimPrefix(regEndpoint, "http://")

	return ctrWithGit().
		With(t.withGitRemoteHelper(ctx)).
		With(withGitConfig()).
		With(withGnociConfig(regHost)).
		WithServiceBinding("registry", registry).
		WithWorkdir(srcDir).
		WithExec([]string{"git", "init"}).
		WithExec(append([]string{"git", "pull", ociRef(regHost, ociSlug)}, refs...)).
		Directory(".", dagger.ContainerDirectoryOpts{Expand: true}), nil
}

// Clone clones a git repository from an OCI registry.
//
//nolint:wrapcheck
func (t *Test) Clone(ctx context.Context,
	// registry service
	registry *dagger.Service,
	// git OCI remote repository slug
	ociSlug string,
) (*dagger.Directory, error) {
	regEndpoint, err := registry.Endpoint(ctx, dagger.ServiceEndpointOpts{Scheme: "http"})
	if err != nil {
		return nil, err
	}
	regHost := strings.TrimPrefix(regEndpoint, "http://")

	_, tag, found := strings.Cut(ociSlug, ":")
	if !found {
		return nil, fmt.Errorf("malformed oci slug %s, expected <repo>:<tag>", ociSlug)
	}

	return ctrWithGit().
		With(t.withGitRemoteHelper(ctx)).
		With(withGitConfig()).
		With(withGnociConfig(regHost)).
		WithServiceBinding("registry", registry).
		WithExec([]string{"git", "clone", ociRef(regHost, ociSlug)}).
		Directory(tag), nil
}

// withGitRemoteHelper builds and installs git-remote-oci in a container.
func (t *Test) withGitRemoteHelper(ctx context.Context) func(c *dagger.Container) *dagger.Container {
	return func(c *dagger.Container) *dagger.Container {
		return c.WithFile(filepath.Join("usr", "local", "bin", gitExecName), t.BuildGit(ctx, t.Src, "test-dev", dagger.Platform("linux/amd64")))
	}
}

// withGitConfig configures a git repository.
func withGitConfig() func(c *dagger.Container) *dagger.Container {
	return func(c *dagger.Container) *dagger.Container {
		return c.WithExec([]string{"git", "config", "--global", "user.name", "dev-test"}).
			WithExec([]string{"git", "config", "--global", "user.email", "devtest@example.com"}).
			WithExec([]string{"git", "config", "--global", "init.defaultbranch", "main"})
	}
}

// withGitLFSRemoteHelper builds and installs git-lfs in an container.
func (t *Test) withGitLFSRemoteHelper(ctx context.Context) func(c *dagger.Container) *dagger.Container {
	return func(c *dagger.Container) *dagger.Container {
		return c.WithFile(filepath.Join("usr", "local", "bin", gitLFSExecName), t.BuildGitLFS(ctx, t.Src, "test-dev", dagger.Platform("linux/amd64")))
	}
}

// withLFSConfig configures a git-lfs enabled repository with an OCI remote.
func withLFSConfig(ociRemote string) func(c *dagger.Container) *dagger.Container {
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

// withGnociConfig injects a gnoci configuration file with HTTP enabled.
func withGnociConfig(regHost string) func(*dagger.Container) *dagger.Container {
	cfgPath := filepath.Join("/.config", "gnoci", "config.yaml")
	cfg := dag.File("config.yaml", fmt.Sprintf("# Git o(n) OCI Configuration\napiVersion: gnoci.act3-ai.io/v1alpha1\nkind: Configuration\nregistryConfig:\n  registries:\n    %s:\n      plainHTTP: true\n", regHost))

	return func(c *dagger.Container) *dagger.Container {
		return c.WithFile(cfgPath, cfg).
			WithEnvVariable("GNOCI_CONFIG", cfgPath)
	}
}

// ociRef builds an oci://<regHost>/<repoSlug/ reference.
func ociRef(regHost string, repoSlug string) string {
	return "oci://" + regHost + "/" + repoSlug
}

func ctrWithGit() *dagger.Container {
	return dag.Alpine(dagger.AlpineOpts{Packages: []string{"git"}}).
		Container()
}
