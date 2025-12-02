package main

import (
	"dagger/gnoci/internal/dagger"
)

const (
	goControllerGen = "sigs.k8s.io/controller-tools/cmd/controller-gen@v0.19.0"
)

// Generate with controller-gen.
func (g *Gnoci) Generate(
	src *dagger.Directory,
) *dagger.Changeset {
	return dag.Go().
		WithSource(src.Filter(dagger.DirectoryFilterOpts{Exclude: []string{".dagger/*"}})).
		WithEnvVariable("GOBIN", "/work/src/tool").
		Exec([]string{"go", "install", goControllerGen}).
		WithExec([]string{"go", "generate", "./..."}).
		Directory(".").
		Changes(src)
}
