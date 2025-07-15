package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/act3-ai/gitoci/internal/actions"
)

// NewCLI creates the base git-remote-oci command
func NewCLI(version string) *cobra.Command {
	// cmd represents the base command when called without any subcommands
	cmd := &cobra.Command{
		Use:          "git-remote-oci REPOSITORY [URL]",
		Short:        "A Git remote helper for syncing Git repositories in OCI Registries.",
		SilenceUsage: true,
		Args:         cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// https://git-scm.com/docs/gitremote-helpers#_invocation
			name := args[0]
			address := name
			if len(args) > 1 {
				address = args[1]
			}

			gitDir, ok := os.LookupEnv("GIT_DIR")
			if !ok {
				return fmt.Errorf("GIT_DIR not set")
			}

			action := actions.NewGitOCI(cmd.InOrStdin(), cmd.OutOrStdout(), gitDir, name, address, version)
			return action.Run(cmd.Context())
		},
	}

	return cmd
}
