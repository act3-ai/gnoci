// Package cli exports the Git Remote Helper for OCI Registries command.
package cli

import (
	"github.com/spf13/cobra"

	"github.com/act3-ai/gnoci/internal/cli"
)

// NewGitLFSRemoteHelper creates the base git-remote-oci command.
func NewGitLFSRemoteHelper(version string) *cobra.Command {
	return cli.NewGitLFSCLI(version)
}
