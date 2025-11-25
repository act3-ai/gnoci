// Package git defines types used in the Git remote helpers protocol.
//
// Contrary to the git-remote-helpers documentation, this package refers to
// commands sent by Git as "Requests" for code readability.
//
// See https://git-scm.com/docs/gitremote-helpers#_commands.
package git

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
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

// Parsable is a parsable request received from Git.
type Parsable interface {
	Parse([]string) error // TODO: limit any?
}

// CapabilitiesRequest is a command received from Git requesting a list of
// supported capabilities
//
// https://git-scm.com/docs/gitremote-helpers#Documentation/gitremote-helpers.txt-capabilities.
type CapabilitiesRequest struct {
	Cmd Command
}

// Parse decodes request fields ensuring the [CapabilitiesRequest] is of the correct type and
// contains sufficient information to handle the request.
//
// Implements [Parsable].
func (r *CapabilitiesRequest) Parse(fields []string) error {
	if len(fields) != 1 {
		return fmt.Errorf("%w: invalid fields for capabilities request: got %v", ErrBadRequest, fields)
	}

	cmd := Command(fields[0])
	if cmd != Capabilities {
		return fmt.Errorf("%w: got %s, want %s", ErrUnexpectedRequest, fields[0], Capabilities)
	}
	r.Cmd = cmd

	return nil
}

// String condenses [CapabilitiesRequest] into a string, the raw request received from Git.
func (r *CapabilitiesRequest) String() string {
	return string(r.Cmd)
}

// OptionRequest is a command received from Git requesting an option to be set.
//
// https://git-scm.com/docs/gitremote-helpers#Documentation/gitremote-helpers.txt-optionnamevalue.
type OptionRequest struct {
	Cmd   Command
	Opt   Option
	Value string
}

// Parse decodes request fields ensuring the [OptionRequest] is of the correct type, is supported,
// and has a valid value.
//
// Implements [Parsable].
func (r *OptionRequest) Parse(fields []string) error {
	if len(fields) < 3 {
		return fmt.Errorf("%w: invalid fields for options request: got %v", ErrBadRequest, fields)
	}

	cmd := Command(fields[0])
	opt := Option(fields[1])
	val := fields[2]

	if cmd != Options {
		return fmt.Errorf("%w: got %s, want %s", ErrUnexpectedRequest, cmd, Options)
	}
	r.Cmd = cmd

	switch opt {
	case Verbosity:
		// ensure valid int, if it's obsurd we'll convert it to a sane value when handled
		_, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("unable to convert verbosity value to int: %w", err)
		}
	default:
		return fmt.Errorf("%w: option %s is not supported", ErrUnsupportedRequest, opt)
	}
	r.Opt = opt

	if val == "" {
		return fmt.Errorf("%w: missing value for option %s", ErrBadRequest, opt)
	}
	r.Value = val

	return nil
}

// String condenses [OptionRequest] into a string, the raw request received from Git.
func (r *OptionRequest) String() string {
	return fmt.Sprintf("%s %s %s", r.Cmd, r.Opt, r.Value)
}

// ListRequest is a command received from Git requesting a list of references.
//
// https://git-scm.com/docs/gitremote-helpers#Documentation/gitremote-helpers.txt-list.
type ListRequest struct {
	Cmd     Command
	ForPush bool
}

// Parse decodes request fields ensuring the [ListRequest] is of the correct type, is supported,
// and has a valid value.
//
// Implements [Parsable].
func (r *ListRequest) Parse(fields []string) error {
	if len(fields) != 1 || len(fields) != 2 {
		return fmt.Errorf("%w: invalid fields for list request: got %v", ErrBadRequest, fields)
	}

	cmd := Command(fields[0])
	if cmd != List {
		return fmt.Errorf("%w: got %s, want %s", ErrUnexpectedRequest, cmd, List)
	}
	r.Cmd = cmd

	if len(fields) == 2 {
		if fields[2] != "for-push" {
			return fmt.Errorf("%w: invalid option for list request: got %v", ErrBadRequest, fields)
		}
		r.ForPush = true
	}

	return nil
}

// String condenses [ListRequest] into a string, the raw request received from Git.
func (r *ListRequest) String() string {
	str := string(r.Cmd)
	if r.ForPush {
		str += " for-push"
	}
	return str
}

// FetchRequest is a command received from Git requesting a fetch operation.
//
// https://git-scm.com/docs/gitremote-helpers#Documentation/gitremote-helpers.txt-fetchsha1name.
type FetchRequest struct {
	Cmd Command
	Ref *plumbing.Reference
}

// String condenses [FetchRequest] into a string, the raw request received from Git.
func (r *FetchRequest) String() string {
	return fmt.Sprintf("%s %s %s", r.Cmd, r.Ref.Hash(), r.Ref.Name())
}

// Parse decodes request fields ensuring the [FetchRequest] is of the correct type, is supported,
// and has a valid value.
//
// Implements [Parsable].
func (r *FetchRequest) Parse(fields []string) error {
	if len(fields) < 3 {
		return fmt.Errorf("%w: invalid fields for fetch request: got %v", ErrBadRequest, fields)
	}

	cmd := Command(fields[0])
	if cmd != Fetch {
		return fmt.Errorf("%w: got %s, want %s", ErrUnexpectedRequest, cmd, Fetch)
	}

	hash := fields[0]
	name := fields[1]
	r.Ref = plumbing.NewHashReference(
		plumbing.ReferenceName(name),
		plumbing.NewHash(hash),
	)

	return nil
}

// PushRequest is a command received from Git requesting a push operation.
//
// https://git-scm.com/docs/gitremote-helpers#Documentation/gitremote-helpers.txt-pushsrcdst.
type PushRequest struct {
	Cmd    Command
	Force  bool
	Src    plumbing.ReferenceName
	Remote plumbing.ReferenceName
}

// String condenses [PushRequest] into a string, the raw request received from Git.
func (r *PushRequest) String() string {
	str := fmt.Sprintf("%s %s:%s", r.Cmd, r.Src, r.Remote)
	if r.Force {
		str = "+" + str
	}
	return str
}

// Parse decodes request fields ensuring the [PushRequest] is of the correct type, is supported,
// and has a valid value.
//
// Implements [Parsable].
func (r *PushRequest) Parse(fields []string) error {
	if len(fields) < 2 {
		return fmt.Errorf("%w: invalid fields for push request: got %v", ErrBadRequest, fields)
	}

	cmd := Command(fields[0])
	if cmd != Push {
		return fmt.Errorf("%w: got %s, want %s", ErrUnexpectedRequest, cmd, Push)
	}

	pair := fields[1]
	s := strings.Split(pair, ":")
	if len(s) != 2 {
		return fmt.Errorf("failed to split reference pair string, got %s, expected <local>:<remote>", pair)
	}
	local := s[0]
	remote := s[1]

	if strings.HasPrefix(local, "+") {
		r.Force = true
		local = strings.TrimPrefix(local, "+")
	}
	r.Src = plumbing.ReferenceName(local)
	r.Remote = plumbing.ReferenceName(remote)

	return nil
}

// ParseRequest decodes a Git request.
func ParseRequest[T Parsable](line string) (T, error) {
	var t T

	fields := strings.Fields(line)
	if len(fields) < 1 {
		return t, ErrEmptyRequest
	}

	err := t.Parse(fields)
	return t, err
}
