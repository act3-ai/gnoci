package main

import (
	"context"
)

// Run tests.
//
//nolint:staticcheck
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
//
//nolint:staticcheck
func (t *Test) All(ctx context.Context) (string, error) {
	unitResults, unitErr := t.Unit(ctx)

	// TODO: add functional  tests here

	out := "Unit Test Results:\n" + unitResults

	return out, unitErr // TODO: use errors.Join when functional tests are added
}

// Run unit tests.
//
//nolint:staticcheck
func (t *Test) Unit(ctx context.Context) (string, error) {
	return dag.Go(). //nolint:wrapcheck
				WithSource(t.Source).
				Container().
				WithExec([]string{"go", "test", "./..."}).
				Stdout(ctx)
}
