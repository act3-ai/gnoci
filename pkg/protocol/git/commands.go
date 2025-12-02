package git

import (
	"slices"
)

// Parsable is a parsable command received from Git.
type Parsable interface {
	Parse([]string) error
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
	Push         Command = "push"
	Fetch        Command = "fetch"
)

// SupportedCommand returns true if a [Command] is supported.
func SupportedCommand(name Command) bool {
	cmds := []Command{
		Capabilities,
		Options,
		List,
		Push,
		Fetch,
	}
	return slices.Contains(cmds, name)
}
