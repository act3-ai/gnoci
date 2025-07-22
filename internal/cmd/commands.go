package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
)

// Error types.
var (
	ErrUnsupportedCommand = errors.New("unsupported git-remote-helper command")
)

// Type is an implemented git-remote-helper command provided by Git.
//
// See https://git-scm.com/docs/gitremote-helpers#_commands.
type Type string

// https://git-scm.com/docs/gitremote-helpers#_commands
const (
	// Git conventions
	Capabilities Type = "capabilities"
	List         Type = "list"
	ListForPush  Type = "for-push"
	Push         Type = "push"
	// Fetch                = "fetch"

	// not a Git convention, marks end of input
	Empty Type = "empty"
)

// Commands defines all implemented git-remote-helper commands.
var Commands = []Type{
	Capabilities,
	Empty,
	List,
	Push,
}

// https://git-scm.com/docs/gitremote-helpers#_options
const (
	Option          Type = "option"
	OptionVerbosity Type = "verbosity"
)

// Options defines all supported option subcommands.
var Options = []Type{
	Option,
	OptionVerbosity,
}

// Git represents a parsed command received from Git. It may include a
// subcommand.
type Git struct {
	Cmd    Type
	SubCmd Type // not all commands include a subcommand
	Data   []string
}

// SupportedOption returns true if an option is supported.
func SupportedOption(name Type) bool {
	// `option` by itself is a capability, we're really checking for subcommands.
	return slices.Contains(Options[1:], name)
}

// SupportedCommand returns true if a Command is supported.
func SupportedCommand(name Type) bool {
	return slices.Contains(Commands, name)
}

// parse parses a single line received from Git, turning it into a cmd.Git
// easily identified by Type.
func parse(ctx context.Context, line string) (Git, error) {
	fields := strings.Fields(line)
	if len(fields) < 1 {
		return Git{
			Cmd: Empty,
		}, nil
	}

	cmd := Type(fields[0])
	switch cmd {
	case Capabilities:
		return Git{
			Cmd: Capabilities,
		}, nil
	case Option:
		if err := validOption(ctx, fields...); err != nil {
			return Git{}, err
		}

		return Git{
			Cmd:    Option,
			SubCmd: Type(fields[1]),
			Data:   fields[2:],
		}, nil
	case List:
		res := Git{
			Cmd: List,
		}
		if len(fields) > 1 {
			res.SubCmd = Type(fields[1])
		}
		return res, nil
	case Push:
		if len(fields) < 2 {
			return Git{}, fmt.Errorf("insufficient args for push command")
		}
		return Git{
			Cmd:  Push,
			Data: fields[1:],
		}, nil
	default:
		return Git{}, fmt.Errorf("%w: %s", ErrUnsupportedCommand, cmd)
	}
}

// validOption ensures an option is properly formed. See SupportedOption() to
// evaluate if an option is supported.
func validOption(ctx context.Context, fields ...string) error {
	// TODO: ideally we return a bool
	// we should try to not make options fatal, but we may have to
	// make an exception for force (or others).
	if len(fields) != 3 {
		slog.ErrorContext(ctx, "invalid number of arguments to option command",
			"got", fmt.Sprintf("%d", len(fields)),
			"want", "3")
		return fmt.Errorf("invalid number of args to option command")
	}
	return nil
}
