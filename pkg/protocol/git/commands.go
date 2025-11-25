package git

import (
	"errors"
	"slices"
)

var (
	// ErrBadRequest indicates a request does not contain sufficient information
	// to handle.
	ErrBadRequest = errors.New("request is invalid")
	// ErrUnexpectedRequest indicates a request was not the expected type.
	ErrUnexpectedRequest = errors.New("unexpected request")
	// ErrUnsupportedRequest indicates a request is not supported by this package.
	ErrUnsupportedRequest = errors.New("unsupported git-remote-helper request")
	// ErrEmptyRequest is an indicator that Git is either breaking up a batch of
	// requests or has finished sending requests.
	ErrEmptyRequest = errors.New("empty request received from Git")
)

// Command is an implemented git-remote-helper command provided by Git.
//
// https://git-scm.com/docs/gitremote-helpers#_commands.
type Command string

// Supported Git commands.
const (
	// Git conventions.
	Capabilities Command = "capabilities"
	Options      Command = "option"
	List         Command = "list"
	ListForPush  Command = "for-push"
	Push         Command = "push"
	Fetch        Command = "fetch"

	// not a Git convention.
	Empty Command = "empty" // marks empty line - a separator
	Done  Command = "done"  // marks end of input
)

// SupportedCommand returns true if a [Command] is supported.
func SupportedCommand(name Command) bool {
	cmds := []Command{
		Capabilities,
		Options,
		Empty,
		List,
		Push,
		Fetch,
	}
	return slices.Contains(cmds, name)
}

// Option is an implemented git-remote-helper option sub command provided by Git.
//
// https://git-scm.com/docs/gitremote-helpers#_options.
type Option string

// Supported Git options.
const (
	Verbosity Option = "verbosity"
)

// SupportedOption returns true if an [Option] is supported.
func SupportedOption(option Option) bool {
	opts := []Option{
		Verbosity,
	}

	return slices.Contains(opts, option)
}
