// Package gen defines auto-generation directives.
package gen

// Generates DeepCopy functions needed for KRM
//go:generate go tool controller-gen object paths=./...
