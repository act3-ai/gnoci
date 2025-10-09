package cli

import (
	"os"

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
		Args:         cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// https://git-scm.com/docs/gitremote-helpers#_invocation
			name := args[0]
			address := name
			if len(args) > 1 {
				address = args[1]
			}

			action := actions.NewGitLFS(cmd.InOrStdin(), cmd.OutOrStdout(), os.Getenv("GIT_DIR"), name, address, version)
			return action.Run(cmd.Context())
		},
	}

	return cmd
}
