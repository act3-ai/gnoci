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

// Type is an implemented git-remote-helper command provided by Git.
//
// See https://git-scm.com/docs/gitremote-helpers#_commands.
type Type string

// https://git-scm.com/docs/gitremote-helpers#_commands
const (
	// Git conventions.
	Capabilities Type = "capabilities"
	List         Type = "list"
	ListForPush  Type = "for-push"
	Push         Type = "push"
	Fetch        Type = "fetch"

	// not a Git convention.
	Empty Type = "empty" // marks empty line - a separator
	Done  Type = "done"  // marks end of input
)

// Commands defines all implemented git-remote-helper commands.
var Commands = []Type{
	Capabilities,
	Empty,
	List,
	Push,
	Fetch,
}

// Git represents a parsed command received from Git. It may include a
// subcommand.
type Git struct {
	Cmd    Type
	SubCmd Type // not all commands include a subcommand
	Data   []string
}

// String formats the Git command as a string.
func (g *Git) String() string {
	return fmt.Sprintf("%s %s %v", g.Cmd, g.SubCmd, g.Data)
}

// SupportedCommand returns true if a Command is supported.
func SupportedCommand(name Type) bool {
	return slices.Contains(Commands, name)
}
