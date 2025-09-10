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

// executable name
const execName = "git-remote-oci"

// Build an executable.
func (g *Gnoci) Build(ctx context.Context,
	// Build version
	// +optional
	version string,
	// Build target platform
	// +optional
	// +default="linux/amd64"
	platform dagger.Platform,
) *dagger.File {
	// trim source for better caching
	src := g.Source.Filter(dagger.DirectoryFilterOpts{
		Exclude: []string{"**/*_test.go"},
		Include: []string{"**/*.go", "go.mod", "go.sum"},
	})

	ldflags := []string{"-s", "-w", `-extldflags "-static"`}
	if version != "" {
		ldflags = append(ldflags, fmt.Sprintf("-X 'main.version=%s'", version))
	}
	return g.goWithSource(src).
		Build(dagger.GoWithSourceBuildOpts{
			Pkg:      fmt.Sprintf("./cmd/%s", execName),
			Platform: platform,
			Trimpath: true,
			Ldflags:  ldflags,
		}).
		WithName(execName)
}

// Build binaries for multiple platforms, nested in directories
// named by platform.
//
//nolint:staticcheck
func (g *Gnoci) BuildPlatforms(ctx context.Context,
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
			ctx, span := Tracer().Start(ctx, fmt.Sprintf("Build %s", platform))
			defer span.End()

			bin := g.Build(ctx, version, platform)

			mux.Lock()
			defer mux.Unlock()
			builds = builds.WithFile(
				path.Join(strings.ReplaceAll(string(platform), "/", "-"), execName),
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
		WithEnvVariable("GOFIPS140", "latest")
}
