// Package gen defines auto-generation directives.
package gen

// Generates DeepCopy functions needed for KRM
//go:generate tool/controller-gen object paths=./...

// Generates JSON Schema definitions for configuration types
// The generated schemas are embedded in the binary
//go:generate go run internal/gen/main.go docs/apis/schemas

// Generates API documentation in markdown format
// The generated docs are embedded in the binary
//go:generate ./internal/gen/crd-ref-docs.sh pkg/apis docs/apis
