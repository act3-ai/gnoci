package git

import (
	"fmt"

	"github.com/go-git/go-git/v5/plumbing"
)

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
