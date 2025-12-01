package cmd

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/act3-ai/gnoci/pkg/protocol/git"
	"github.com/act3-ai/gnoci/pkg/protocol/git/comms"
)

// HandleCapabilities executes the capabilities command by listing supported
// capabilities to Git.
func HandleCapabilities(ctx context.Context, comm comms.Communicator) error {
	// reset comm in case of lookahead
	_, err := comm.ParseCapabilitiesRequest()
	if err != nil {
		return fmt.Errorf("parsing capabilities request: %w", err)
	}

	capabilities := []git.Capability{git.CapabilityOption, git.CapabilityFetch, git.CapabilityPush}

	slog.DebugContext(ctx, "writing supported capabilities", "capabilities", fmt.Sprintf("%v", capabilities))
	if err := comm.WriteCapabilitiesResponse(capabilities); err != nil {
		return fmt.Errorf("writing capabilities: %w", err)
	}
	return nil
}
