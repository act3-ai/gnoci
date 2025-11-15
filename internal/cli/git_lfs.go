package cli

import (
	"github.com/act3-ai/gnoci/internal/actions"
	"github.com/act3-ai/go-common/pkg/config"
	"github.com/spf13/cobra"
)

// NewGitLFSCLI creates the base git-lfs-remote-oci command.
func NewGitLFSCLI(version string) *cobra.Command {
	// cmd represents the base command when called without any subcommands
	cmd := &cobra.Command{
		Use:          "git-lfs-remote-oci",
		Short:        "A git-lfs remote helper for syncing git-lfs files in OCI Registries.",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			action := actions.NewGitLFS(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
				version,
				config.EnvPathOr("GNOCI_CONFIG", config.DefaultConfigSearchPath("gnoci", "config.yaml")),
			)
			return action.Run(cmd.Context())
		},
	}

	return cmd
}
