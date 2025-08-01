// Package cli exports the Git Remote Helper for OCI Registries command.
package cli

import (
	"github.com/spf13/cobra"

	"github.com/act3-ai/gitoci/internal/cli"
)

// NewCLI creates the base gitoci command.
func NewCLI(version string) *cobra.Command {
	return cli.NewCLI(version)
}
