package actions

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/act3-ai/gitoci/internal/cmd"
)

// GitOCI represents the base action
type GitOCI struct {
	// TODO: Could be dangerous when storing in struct like this... mutex?
	batcher cmd.BatchReadWriter

	// local repository
	gitDir string

	// OCI remote
	name   string // may have same value as address
	addess string

	Option

	version string
}

// NewGitOCI creates a new Tool with default values
func NewGitOCI(in io.Reader, out io.Writer, gitDir, shortname, address, version string) *GitOCI {
	return &GitOCI{
		batcher: cmd.NewBatcher(in, out),
		gitDir:  gitDir,
		name:    shortname,
		addess:  strings.TrimPrefix(address, "oci://"),
		version: version,
	}
}

// Runs the Hello action
func (action *GitOCI) Run(ctx context.Context) error {
	// first command is always "capabilities"
	c, err := action.batcher.Read(ctx)
	switch {
	case err != nil:
		return fmt.Errorf("reading initial command: %w", err)
	case c.Cmd != cmd.Capabilities:
		return fmt.Errorf("unexpected first command %s, expected 'capabilities'", c.Cmd)
	default:
		if err := action.capabilities(ctx); err != nil {
			return err
		}
	}

	var done bool
	for !done {
		c, err := action.batcher.Read(ctx)
		if err != nil {
			return fmt.Errorf("reading next line: %w", err)
		}

		switch c.Cmd {
		case cmd.Empty:
			// done = true
			continue
		case cmd.Capabilities:
			// Git shouldn't need to do this again, but let's be safe
			if err := action.capabilities(ctx); err != nil {
				return fmt.Errorf("running capabilities command: %w", err)
			}
		case cmd.Option:
			if err := action.option(ctx, c); err != nil {
				return fmt.Errorf("running option command: %w", err)
			}
		case cmd.List:
			if err := action.list(ctx); err != nil {
				return fmt.Errorf("running list command: %w", err)
			}
		default:
			return fmt.Errorf("default case hit")
		}
	}

	// // TODO: Next command is 'list', can be read in a batch
	// slog.InfoContext(ctx, "reading batch")
	// _, err = action.batcher.ReadBatch(ctx)
	// if err != nil {
	// 	return fmt.Errorf("reading batch input: %w", err)
	// }

	return fmt.Errorf("not implemented")
}
