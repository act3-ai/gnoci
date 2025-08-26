// Package main is the main CLI package.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/muesli/termenv"
	"github.com/spf13/cobra"

	"github.com/act3-ai/go-common/pkg/logger"
	"github.com/act3-ai/go-common/pkg/runner"
	vv "github.com/act3-ai/go-common/pkg/version"

	"github.com/act3-ai/gnoci/cmd/git-remote-oci/cli"
)

// getVersionInfo retrieves build info.
func getVersionInfo() vv.Info {
	info := vv.Get()
	if version != "" {
		info.Version = version
	}
	return info
}

func main() {
	ctx := context.Background()

	info := getVersionInfo()         // Load the version info from the build
	root := cli.NewCLI(info.Version) // Create the root command
	root.SilenceUsage = true         // Silence usage when root is called

	// Layout of embedded documentation to surface in the help command
	// and generate in the gendocs command
	// embeddedDocs := docs.Embedded(root)

	// Add common commands
	// root.AddCommand(
	// 	commands.NewVersionCmd(info),
	// 	commands.NewGenschemaCmd(docs.Schemas(), docs.SchemaAssociations),
	// 	commands.NewGendocsCmd(embeddedDocs),
	// 	commands.NewInfoCmd(embeddedDocs),
	// )

	// Store persistent pre run function to avoid overwriting it
	persistentPreRun := root.PersistentPreRun

	// The pre run function logs build info and sets the default output writer
	root.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		slog.SetDefault(logger.FromContext(cmd.Context()))                                // Set global slog.Logger
		slog.InfoContext(cmd.Context(), "Software", slog.String("version", info.Version)) // Log version info
		slog.DebugContext(cmd.Context(), "Software details", slog.Any("info", info))      // Log build info
		termenv.SetDefaultOutput(termenv.NewOutput(cmd.OutOrStdout()))                    // Set termenv default output

		if persistentPreRun != nil {
			persistentPreRun(cmd, args)
		}
	}

	// Run the root command
	if err := runner.Run(ctx, root, "GNOCI_VERBOSITY"); err != nil {
		os.Exit(1)
	}
}
