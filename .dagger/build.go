package main

import (
	"context"
	"dagger/gnoci/internal/dagger"
	"fmt"
	"path"
	"strings"
	"sync"

	"github.com/sourcegraph/conc/pool"
)

const (
	// git remote helper executable name
	gitExecName    = "git-remote-oci"
	gitLFSExecName = "git-lfs-remote-oci"
)

// Build an git-remote-oci executable.
func (g *Gnoci) BuildGit(ctx context.Context,
	// Source code directory
	// +defaultPath="/"
	src *dagger.Directory,
	// Build version
	// +optional
	version string,
	// Build target platform
	// +optional
	// +default="linux/amd64"
	platform dagger.Platform,
) *dagger.File {
	src = g.dirFilterGo(src)

	ldflags := []string{"-s", "-w", `-extldflags "-static"`}
	if version != "" {
		ldflags = append(ldflags, fmt.Sprintf("-X 'main.version=%s'", version))
	}
	return g.goWithSource(src).
		Build(dagger.GoWithSourceBuildOpts{
			Pkg:      fmt.Sprintf("./cmd/%s", gitExecName),
			Platform: platform,
			Trimpath: true,
			Ldflags:  ldflags,
		}).
		WithName(gitExecName)
}

// Build an git-lfs-remote-oci executable.
func (g *Gnoci) BuildGitLFS(ctx context.Context,
	// Source code directory
	// +defaultPath="/"
	src *dagger.Directory,
	// Build version
	// +optional
	version string,
	// Build target platform
	// +optional
	// +default="linux/amd64"
	platform dagger.Platform,
) *dagger.File {
	src = g.dirFilterGo(src)

	ldflags := []string{"-s", "-w", `-extldflags "-static"`}
	if version != "" {
		ldflags = append(ldflags, fmt.Sprintf("-X 'main.version=%s'", version))
	}
	return g.goWithSource(src).
		Build(dagger.GoWithSourceBuildOpts{
			Pkg:      fmt.Sprintf("./cmd/%s", gitLFSExecName),
			Platform: platform,
			Trimpath: true,
			Ldflags:  ldflags,
		}).
		WithName(gitLFSExecName)
}

// Build git-remote-oci binaries for multiple platforms, nested in directories
// named by platform.
func (g *Gnoci) BuildGitPlatforms(ctx context.Context,
	// Source code directory
	// +defaultPath="/"
	src *dagger.Directory,
	// Build version
	// +optional
	version string,
	// build platforms
	// +default=["linux/amd64","linux/arm64","darwin/arm64"]
	platforms []dagger.Platform,
) *dagger.Directory {
	var mux sync.Mutex
	builds := dag.Directory()

	p := pool.New().WithContext(ctx)

	for _, platform := range platforms {
		p.Go(func(ctx context.Context) error {
			ctx, span := Tracer().Start(ctx, fmt.Sprintf("Build git-remote-oci for %s", platform))
			defer span.End()

			bin := g.BuildGit(ctx, src, version, platform)

			mux.Lock()
			defer mux.Unlock()
			builds = builds.WithFile(
				path.Join(strings.ReplaceAll(string(platform), "/", "-"), gitExecName),
				bin)

			return nil
		})
	}
	_ = p.Wait() // throw away err, as we can't get one

	return builds
}

// Build git-lfs-remote-oci binaries for multiple platforms, nested in directories
// named by platform.
func (g *Gnoci) BuildGitLFSPlatforms(ctx context.Context,
	// Source code directory
	// +defaultPath="/"
	src *dagger.Directory,
	// Build version
	// +optional
	version string,
	// build platforms
	// +default=["linux/amd64","linux/arm64","darwin/arm64"]
	platforms []dagger.Platform,
) *dagger.Directory {
	var mux sync.Mutex
	builds := dag.Directory()

	p := pool.New().WithContext(ctx)

	for _, platform := range platforms {
		p.Go(func(ctx context.Context) error {
			ctx, span := Tracer().Start(ctx, fmt.Sprintf("Build git-lfs-remote-oci for %s", platform))
			defer span.End()

			bin := g.BuildGitLFS(ctx, src, version, platform)

			mux.Lock()
			defer mux.Unlock()
			builds = builds.WithFile(
				path.Join(strings.ReplaceAll(string(platform), "/", "-"), gitLFSExecName),
				bin)

			return nil
		})
	}
	_ = p.Wait() // throw away err, as we can't get one

	return builds
}

// Build a git-remote-oci and git-lfs-remote-oci binaries for multiple platforms,
// nested in directories named by platform
func (g *Gnoci) BuildAllPlatforms(ctx context.Context,
	// Source code directory
	// +defaultPath="/"
	src *dagger.Directory,
	// Build version
	// +optional
	version string,
	// build platforms
	// +default=["linux/amd64","linux/arm64","darwin/arm64"]
	platforms []dagger.Platform,
) *dagger.Directory {
	ctx, span := Tracer().Start(ctx, fmt.Sprintf("Building remote helpers for git and git-lfs for %v", platforms))
	defer span.End()

	var mux sync.Mutex
	builds := dag.Directory()

	p := pool.New().WithContext(ctx)

	for _, platform := range platforms {
		p.Go(func(ctx context.Context) error {
			buildCtx, span := Tracer().Start(ctx, fmt.Sprintf("Build git-remote-oci for %s", platform))
			defer span.End()

			bin := g.BuildGit(buildCtx, src, version, platform)

			mux.Lock()
			defer mux.Unlock()
			builds = builds.WithFile(
				path.Join(strings.ReplaceAll(string(platform), "/", "-"), gitExecName),
				bin)

			return nil
		})

		p.Go(func(ctx context.Context) error {
			buildCtx, span := Tracer().Start(ctx, fmt.Sprintf("Build git-lfs-remote-oci for %s", platform))
			defer span.End()

			bin := g.BuildGitLFS(buildCtx, src, version, platform)

			mux.Lock()
			defer mux.Unlock()
			builds = builds.WithFile(
				path.Join(strings.ReplaceAll(string(platform), "/", "-"), gitLFSExecName),
				bin)

			return nil
		})
	}
	_ = p.Wait() // throw away err, as we can't get one

	return builds
}

// Initializes a container with Go and the source.
func (g *Gnoci) goWithSource(src *dagger.Directory) *dagger.GoWithSource {
	return dag.Go().
		WithSource(src).
		WithCgoDisabled().
		WithEnvVariable("GOFIPS140", "latest").
		WithEnvVariable("GOBIN", "/work/src/bin")
}

// dirFilterGo filters a directory to include only go files.
func (g *Gnoci) dirFilterGo(d *dagger.Directory) *dagger.Directory {
	return d.Filter(dagger.DirectoryFilterOpts{
		Exclude: []string{"**/*_test.go"},
		Include: []string{"**/*.go", "go.mod", "go.sum"},
	})
}
