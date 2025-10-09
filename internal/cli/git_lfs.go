package cli

import (
	"github.com/act3-ai/gnoci/internal/actions"
	"github.com/spf13/cobra"
)

// NewGitLFSCLI creates the base git-lfs-remote-oci command.
func NewGitLFSCLI(version string) *cobra.Command {
	// cmd represents the base command when called without any subcommands
	cmd := &cobra.Command{
		Use:          "git-lfs-remote-oci REPOSITORY [URL]",
		Short:        "A git-lfs remote helper for syncing git-lfs files in OCI Registries.",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			action := actions.NewGitLFS(cmd.InOrStdin(), cmd.OutOrStdout(), version)
			return action.Run(cmd.Context())
		},
	}

	return cmd
}
