package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
)

// https://git-scm.com/docs/gitremote-helpers#_options
const (
	Option          Command = "option"
	OptionVerbosity Command = "verbosity"
)

// Options defines all supported option subcommands.
var Options = []Command{
	Option,
	OptionVerbosity,
}

// HandleOption executes an option command.
func HandleOption(ctx context.Context, g Git, w Writer) error {
	const (
		ok          = "ok"
		unsupported = "unsupported"
	)

	// https://git-scm.com/docs/gitremote-helpers#Documentation/gitremote-helpers.txt-optionnamevalue
	var result string
	err := handleOption(ctx, g)
	switch {
	case errors.Is(err, ErrUnsupportedCommand):
		slog.DebugContext(ctx, "received unsupported option command", "command", g.SubCmd)
		result = unsupported
	case err != nil:
		slog.ErrorContext(ctx, "failed to handle option command", "command", g.SubCmd)
		result = err.Error()
	default:
		slog.DebugContext(ctx, "successfully handled option command", "command", g.SubCmd)
		result = ok
	}

	if err := w.Write(ctx, result); err != nil {
		return fmt.Errorf("writing option response %s: %w", g.SubCmd, err)
	}
	// Git will print a warning to stderr if a newline is written
	if err := w.Flush(false); err != nil {
		return fmt.Errorf("flushing option writes: %w", err)
	}
	return nil
}

func handleOption(ctx context.Context, g Git) error {
	slog.DebugContext(ctx, "handling option", slog.String("command", g.String()))

	switch g.SubCmd {
	case OptionVerbosity:
		return verbosity(g.Data)
	default:
		// sanity, should never happen
		return fmt.Errorf("%w: %s", ErrUnsupportedCommand, g.String())
	}
}

// verbosity handles the verbosity option.
func verbosity(args []string) error {
	if len(args) != 1 {
		return errors.New("missing value from verbosity command")
	}
	value := args[0]

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

// validOption ensures an option is properly formed.
func validOption(ctx context.Context, fields ...string) error {
	if len(fields) != 3 {
		slog.ErrorContext(ctx, "invalid number of arguments to option command",
			"got", fmt.Sprintf("%d", len(fields)),
			"want", "3")
		return fmt.Errorf("invalid number of args to option command")
	}
	return nil
}
