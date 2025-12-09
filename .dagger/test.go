package main

import (
	"context"
	"dagger/gnoci/internal/dagger"
	"fmt"
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
func (t *Test) All(ctx context.Context,
	src *dagger.Directory,
) (string, error) {
	unitResults, unitErr := t.Unit(ctx, src)

	// TODO: add functional  tests here

	out := "Unit Test Results:\n" + unitResults

	return out, unitErr // TODO: use errors.Join when functional tests are added
}

// Run unit tests.
func (t *Test) Unit(ctx context.Context,
	src *dagger.Directory,
) (string, error) {
	return dag.Go(). //nolint:wrapcheck
				WithSource(src).
				Container().
				WithExec([]string{"go", "test", "./..."}).
				Stdout(ctx)
}

// PushClone push to then clone from an OCI registry.
func (t *Test) PushClone(ctx context.Context,
	// source code directory
	// +defaultPath="/"
	src *dagger.Directory,
	// Git reference to test repository
	gitRef *dagger.GitRef,
) (string, error) {
	// start registry
	registry := registryService()
	registry, err := registry.Start(ctx)
	if err != nil {
		return "", fmt.Errorf("starting registry service: %w", err)
	}
	defer registry.Stop(ctx) //nolint:errcheck

	const ociSlug = "repo/test:clone"

	pushOut, err := t.Push(ctx, src, gitRef, registry, ociSlug)
	if err != nil {
		return pushOut, fmt.Errorf("failed to push repository to OCI: %w", err)
	}

	cloneOut, err := t.Clone(ctx, src, registry, ociSlug)
	if err != nil {
		return pushOut, fmt.Errorf("failed to clone repository from OCI: %w", err)
	}

	return strings.Join([]string{pushOut, cloneOut}, "\n\n"), nil
}

// Push pushes a git repository to an OCI registry.
//
//nolint:wrapcheck
func (t *Test) Push(ctx context.Context,
	// source code directory
	// +defaultPath="/"
	src *dagger.Directory,
	// Git reference to test repository
	gitRef *dagger.GitRef,
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
	return dag.Alpine(dagger.AlpineOpts{Packages: []string{"git"}}).
		Container().
		With(t.withGit(ctx, src)).
		With(withGitConfig()).
		With(withGnociConfig(regHost)).
		WithDirectory(srcDir, gitRef.Tree(dagger.GitRefTreeOpts{Depth: -1})).
		WithWorkdir(srcDir).
		WithServiceBinding("registry", registry).
		WithExec([]string{"git", "push", ociRef(regHost, ociSlug), "--all"}).
		CombinedOutput(ctx)
}

// Clone clones a git repository from an OCI registry.
//
//nolint:wrapcheck
func (t *Test) Clone(ctx context.Context,
	// source code directory
	// +defaultPath="/"
	src *dagger.Directory,
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

	return dag.Alpine(dagger.AlpineOpts{Packages: []string{"git"}}).
		Container().
		With(t.withGit(ctx, src)).
		With(withGitConfig()).
		With(withGnociConfig(regHost)).
		WithWorkdir(srcDir).
		WithServiceBinding("registry", registry).
		WithExec([]string{"git", "clone", ociRef(regHost, ociSlug)}).Terminal().
		CombinedOutput(ctx)
}

// withGit builds and installs git-remote-oci in a container.
func (t *Test) withGit(ctx context.Context, src *dagger.Directory) func(c *dagger.Container) *dagger.Container {
	return func(c *dagger.Container) *dagger.Container {
		return c.WithFile(filepath.Join("usr", "local", "bin", gitExecName), t.BuildGit(ctx, src, "test-dev", dagger.Platform("linux/amd64")))
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

// withGitLFS builds and installs git-lfs in an container.
func (t *Test) withGitLFS(ctx context.Context, src *dagger.Directory) func(c *dagger.Container) *dagger.Container {
	return func(c *dagger.Container) *dagger.Container {
		return c.WithFile(filepath.Join("usr", "local", "bin", gitLFSExecName), t.BuildGitLFS(ctx, src, "test-dev", dagger.Platform("linux/amd64")))
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
