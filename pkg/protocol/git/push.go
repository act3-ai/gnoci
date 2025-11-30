package git

import (
	"fmt"
	"iter"
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

// PushResult is an indicator whether or not a push was successful.
type PushResult string

const (
	// PushOK indicates a push was completed successfully.
	PushOK = "ok"
	// PushErr indicates an error was encountered during a push.
	PushErr = "error"
)

// PushResponse is a response to a [PushRequest].
type PushResponse struct {
	Result PushResult
	Err    error
}

// PushLister iterates through a set of references.
type PushLister iter.Seq[plumbing.Reference]

// PushResults creates an [iter.Seq] for push results.
func PushResults(refs ...plumbing.Reference) ReferenceLister {
	return func(yield func(plumbing.Reference) bool) {
		for _, item := range refs {
			if !yield(item) {
				return
			}
		}
	}
}
