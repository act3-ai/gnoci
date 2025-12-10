package main

import (
	"context"
	"dagger/gnoci/internal/dagger"
	"fmt"
	"strings"
)

const (
	githubSlug = "act3-ai/gnoci"
)

// ReleasePrepare runs tests, linters, and preps release documentation.
func (g *Gnoci) ReleasePrepare(ctx context.Context,
	// Git reference to release (can use "." for the current commit)
	gitRef *dagger.GitRef,
	// release with a specific version
	// +optional
	version string,
) (*dagger.Changeset, error) {
	src := gitRef.Tree()

	_ = g.BuildAllPlatforms(ctx, src, version, buildPlatforms)

	if err := g.Test(src).All(ctx); err != nil {
		return nil, err
	}

	if _, err := g.Lint().All(ctx, src); err != nil {
		return nil, err
	}

	// generate documentation changes
	release := dag.Release(gitRef)
	if version == "" {
		v, err := release.Version(ctx)
		if err != nil {
			return nil, fmt.Errorf("resolving release version: %w", err)
		}
		version = v
	}

	src = src.WithChanges(release.Prepare(version))
	// src = src.WithChanges(g.Generate(src))
	src = src.WithChanges(g.Test(src).CoverageDocs(ctx))

	return src.Changes(gitRef.Tree()), nil
}

// ReleasePublish finalizes the release.
//
//nolint:wrapcheck
func (g *Gnoci) ReleasePublish(ctx context.Context,
	// Git reference to release (can use "." for the current commit)
	gitRef *dagger.GitRef,
) (string, error) {

	src := gitRef.Tree()

	version, err := src.File("VERSION").Contents(ctx)
	if err != nil {
		return "", err
	}
	version = strings.TrimSpace(version)

	release := dag.Release(gitRef)

	return release.CreateGithub(ctx,
		githubSlug,
		g.Token,
		"v"+version,
		src.File(fmt.Sprintf("releases/v%s.md", version)))

}
