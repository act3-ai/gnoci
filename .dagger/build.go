package main

import (
	"context"
	"dagger/gnoci/internal/dagger"
	"fmt"
)

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
		Include: []string{"**/*.go"},
	})

	ldflags := []string{"-s", "-w", `-extldflags "-static"`}
	if version != "" {
		ldflags = append(ldflags, fmt.Sprintf("-X 'main.version=%s'", version))
	}
	return g.goWithSource(src).
		Build(dagger.GoWithSourceBuildOpts{
			Pkg:      "./cmd/git-remote-oci",
			Platform: platform,
			Trimpath: true,
			Ldflags:  ldflags,
		}).
		WithName("git-remote-oci")
}

// Initializes a container with Go and the source.
func (g *Gnoci) goWithSource(src *dagger.Directory) *dagger.GoWithSource {
	return dag.Go().
		WithSource(src).
		WithCgoDisabled().
		WithEnvVariable("GOFIPS140", "latest")
}
