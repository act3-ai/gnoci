package actions

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/act3-ai/gnoci/internal/cmd"
)

// Option defines a supported option capability.
type Option struct{}

// option handles and responds to the option subcommands.
func (action *GnOCI) option(ctx context.Context, c cmd.Git) error {
	const (
		ok          = "ok"
		unsupported = "unsupported"
	)

	value := c.Data[0]

	// https://git-scm.com/docs/gitremote-helpers#Documentation/gitremote-helpers.txt-optionnamevalue
	var result string
	err := action.handleOption(ctx, c.SubCmd, value)
	switch {
	case errors.Is(err, cmd.ErrUnsupportedCommand):
		slog.DebugContext(ctx, "received unsupported option command", "command", c.SubCmd)
		result = unsupported
	case err != nil:
		slog.ErrorContext(ctx, "failed to handle option command", "command", c.SubCmd)
		result = err.Error()
	default:
		slog.DebugContext(ctx, "successfully handled option command", "command", c.SubCmd)
		result = ok
	}

	if err := action.batcher.Write(ctx, result); err != nil {
		return fmt.Errorf("writing option response %s: %w", c.SubCmd, err)
	}
	// Git will print a warning to stderr if a newline is written
	if err := action.batcher.Flush(false); err != nil {
		return fmt.Errorf("flushing option writes: %w", err)
	}
	return nil
}

// handleOption fulfills and option command if supported.
func (action *GnOCI) handleOption(ctx context.Context, name cmd.Type, value string) error {
	if !cmd.SupportedOption(name) {
		return errors.Join(cmd.ErrUnsupportedCommand, fmt.Errorf("unsupported option %s", name))
	}

	switch name {
	case cmd.OptionVerbosity:
		return action.verbosity(value)
	default:
		// sanity, should never happen
		slog.DebugContext(ctx, "handleOption not able to handle supposedly supported option command", "command", name)
	}

	return nil
}

// verbosity handles the `option verbosity` command.
//
// https://git-scm.com/docs/gitremote-helpers#Documentation/gitremote-helpers.txt-optionverbosityn
func (action *GnOCI) verbosity(value string) error {
	val, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("converting verbosity value to int: %w", err)
	}

	var lvl slog.Level
	switch {
	case val <= 0:
		lvl = slog.LevelError
	case val == 1:
		lvl = slog.LevelWarn
	case val == 2:
		lvl = slog.LevelInfo
	default:
		lvl = slog.LevelDebug
	}

	slog.SetLogLoggerLevel(lvl)

	return nil
}
