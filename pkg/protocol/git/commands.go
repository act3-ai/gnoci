package git

import (
	"slices"
)

// Parsable is a parsable command received from Git.
type Parsable interface {
	Parse([]string) error // TODO: limit any?
}

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
