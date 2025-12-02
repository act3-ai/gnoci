package main

import (
	"dagger/gnoci/internal/dagger"
)

// Generate with controller-gen.
func (g *Gnoci) Generate(
	// Top level source code directory
	// +defaultPath="/"
	// +ignore=["**", "!go.*", "!pkg", "!internal", "!cmd", "!docs"]
	src *dagger.Directory,
) *dagger.Changeset {
	afterSrc := g.goWithSource(src).
		Generate(dagger.GoWithSourceGenerateOpts{
			Packages: []string{"./..."},
		}).
		Source()
	beforeSrc := dag.Container().WithDirectory("/src", src).Directory("/src")

	return afterSrc.Changes(beforeSrc)
}
