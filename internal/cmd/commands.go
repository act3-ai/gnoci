package cmd

import (
	"errors"
	"fmt"
	"slices"
)

// Error types.
var (
	ErrUnsupportedCommand = errors.New("unsupported git-remote-helper command")
)

// Command is an implemented git-remote-helper command provided by Git.
//
// See https://git-scm.com/docs/gitremote-helpers#_commands.
type Command string

// https://git-scm.com/docs/gitremote-helpers#_commands
const (
	// Git conventions.
	Capabilities Command = "capabilities"
	List         Command = "list"
	ListForPush  Command = "for-push"
	Push         Command = "push"
	Fetch        Command = "fetch"

	// not a Git convention.
	Empty Command = "empty" // marks empty line - a separator
	Done  Command = "done"  // marks end of input
)

// Commands defines all implemented git-remote-helper commands.
var Commands = []Command{
	Capabilities,
	Empty,
	List,
	Push,
	Fetch,
}

// Git represents a parsed command received from Git. It may include a
// subcommand.
type Git struct {
	Cmd    Command
	SubCmd Command // not all commands include a subcommand
	Data   []string
}

// String formats the Git command as a string.
func (g *Git) String() string {
	return fmt.Sprintf("%s %s %v", g.Cmd, g.SubCmd, g.Data)
}

// SupportedCommand returns true if a Command is supported.
func SupportedCommand(name Command) bool {
	return slices.Contains(Commands, name)
}
