package git

import (
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
)

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
	if r.Force {
		return fmt.Sprintf("%s +%s:%s", r.Cmd, r.Src, r.Remote)
	}
	return fmt.Sprintf("%s +%s:%s", r.Cmd, r.Src, r.Remote)

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

// PushResponse is a status indicating if a push request was handled successfully.
type PushResponse struct {
	// remote reference
	Remote plumbing.ReferenceName
	// nil if successful
	Error error
}

// String condenses the response into a format readable by Git.
func (r *PushResponse) String() string {
	if r.Error != nil {
		return fmt.Sprintf("error %s %s", r.Remote.String(), r.Error.Error())
	}
	return fmt.Sprintf("ok %s", r.Remote.String())
}
