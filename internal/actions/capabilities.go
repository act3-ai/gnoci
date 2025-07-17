package actions

import (
	"context"
	"fmt"
	"log/slog"
)

// Capability defines a git-remote-helper capability.
//
// See https://git-scm.com/docs/gitremote-helpers#_capabilities.
type Capability = string

// Capabilities with a '*' prefix marks them as mandatory.
const (
	CapOption Capability = "option"
	CapFetch  Capability = "fetch"
	// CapPush   Capability = "push"
)

func (action *GitOCI) capabilities(ctx context.Context) error {
	capabilities := []Capability{CapOption, CapFetch}
	slog.DebugContext(ctx, "writing supported capabilities", "capabilities", fmt.Sprintf("%v", capabilities))
	if err := action.batcher.WriteBatch(ctx, capabilities...); err != nil {
		return fmt.Errorf("writing capabilities: %w", err)
	}
	return nil
}
