package git

import (
	"fmt"
	"strconv"
)

// Option is an implemented git-remote-helper option sub command provided by Git.
//
// https://git-scm.com/docs/gitremote-helpers#_options.
type Option string

// Supported Git options.
const (
	Verbosity Option = "verbosity"
)

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
		// ensure valid int
		_, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("unable to convert verbosity value to int: %w", err)
		}
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
