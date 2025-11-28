// Package gitmock mocks go-git concrete type custom interfaces.
package gitmock

//go:generate go tool mockgen -typed -package gitmock -destination ./gitmock.gen.go github.com/act3-ai/gnoci/internal/git Repository
