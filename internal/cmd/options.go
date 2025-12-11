package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/act3-ai/gnoci/pkg/protocol/git"
	"github.com/act3-ai/gnoci/pkg/protocol/git/comms"
)

// HandleOption executes an option command.
func HandleOption(ctx context.Context, comm comms.Communicator) error {
	req, err := comm.ParseOptionRequest()
	if err != nil {
		return fmt.Errorf("parsing option request: %w", err)
	}
	log := slog.With(slog.String("command", req.String()))

	// https://git-scm.com/docs/gitremote-helpers#Documentation/gitremote-helpers.txt-optionnamevalue
	err = handleOption(ctx, req)
	switch {
	case errors.Is(err, git.ErrUnsupportedRequest):
		log.DebugContext(ctx, "received unsupported option command")
		if err := comm.WriteOptionResponse(false); err != nil {
			return fmt.Errorf("writing option response: %w", err)
		}
	case err != nil:
		// unreachable, for now
		log.ErrorContext(ctx, "failed to handle option command")
		return fmt.Errorf("handling option: %w", err)
	default:
		log.DebugContext(ctx, "successfully handled option command")
		if err := comm.WriteOptionResponse(true); err != nil {
			return fmt.Errorf("writing option response: %w", err)
		}
	}

	return nil
}

func handleOption(ctx context.Context, req *git.OptionRequest) error {
	slog.DebugContext(ctx, "handling option", slog.String("command", req.String()))

	switch req.Opt {
	case git.Verbosity:
		return verbosity(req.Value)
	default:
		return fmt.Errorf("%w: %s", git.ErrUnsupportedRequest, req.String())
	}
}

// verbosity handles the verbosity option.
func verbosity(value string) error {
	val, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("converting verbosity value to int: %w", err)
	}

	// TODO: We should tune these
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
